/*
 * This file is subject to the terms and conditions defined in
 * file 'LICENSE.md', which is part of this source code package.
 */

package unitype

import (
	"bytes"
	"errors"
	"io"
	"math"
	"os"
	"sort"

	"github.com/sirupsen/logrus"
)

// Font wraps font for outside access.
type Font struct {
	br *byteReader
	*font
}

// Parse parses the truetype font from `rs` and returns a new Font.
func Parse(rs io.ReadSeeker) (*Font, error) {
	r := newByteReader(rs)

	fnt, err := parseFont(r)
	if err != nil {
		return nil, err
	}

	return &Font{
		br:   r,
		font: fnt,
	}, nil
}

// ParseFile parses the truetype font from file given by path.
func ParseFile(filePath string) (*Font, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	defer f.Close()
	return Parse(f)
}

// ValidateBytes validates the turetype font represented by the byte stream.
func ValidateBytes(b []byte) error {
	r := bytes.NewReader(b)
	br := newByteReader(r)
	fnt, err := parseFont(br)
	if err != nil {
		return err
	}

	return fnt.validate(br)
}

// ValidateFile validates the truetype font given by `filePath`.
func ValidateFile(filePath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	br := newByteReader(f)
	fnt, err := parseFont(br)
	if err != nil {
		return err
	}

	return fnt.validate(br)
}

// GetCmap returns the specific cmap specified by `platformID` and platform-specific `encodingID`.
// If not available, nil is returned. Used in PDF for decoding.
func (f *Font) GetCmap(platformID, encodingID int) map[rune]GlyphIndex {
	if f.cmap == nil {
		return nil
	}

	for _, subt := range f.cmap.subtables {
		if subt.platformID == platformID && subt.encodingID == encodingID {
			return subt.cmap
		}
	}

	return nil
}

// LookupRunes looks up each rune in `rune` and returns a matching slice of glyph indices.
// When a rune is not found, a GID of 0 is used (notdef).
func (f *Font) LookupRunes(runes []rune) []GlyphIndex {
	var maps []map[rune]GlyphIndex
	// Search order (3,1), (1,0), (0,3), (3,10).
	maps = append(maps,
		f.GetCmap(3, 1),
		f.GetCmap(1, 0),
		f.GetCmap(0, 3),
		f.GetCmap(3, 10),
	)

	var indices []GlyphIndex
	for _, r := range runes {
		index := GlyphIndex(0)
		for _, cmap := range maps {
			ind, has := cmap[r]
			if has {
				index = ind
				break
			}
		}
		indices = append(indices, index)
	}
	logrus.Debugf("Runes: %+v %s", runes, string(runes))
	logrus.Debugf("GIDs: %+v", indices)
	return indices
}

// SubsetKeepRunes prunes data for all GIDs except the ones corresponding to `runes`.  The GIDs are
// maintained. Typically reduces glyf table size significantly.
func (f *Font) SubsetKeepRunes(runes []rune) (*Font, error) {
	indices := f.LookupRunes(runes)
	return f.SubsetKeepIndices(indices)
}

// SubsetKeepIndices prunes data for all GIDs outside of `indices`. The GIDs are maintained.
// This typically works well and is a simple way to prune most of the unnecessary data as the
// glyf table is usually the biggest by far.
func (f *Font) SubsetKeepIndices(indices []GlyphIndex) (*Font, error) {
	newfnt := font{}

	// Expand the set of indices if any of the indices are composite
	// glyphs depending on other glyphs.
	gidIncludedMap := make(map[GlyphIndex]struct{}, len(indices))
	for _, gid := range indices {
		gidIncludedMap[gid] = struct{}{}
	}

	toscan := make([]GlyphIndex, 0, len(gidIncludedMap))
	for gid := range gidIncludedMap {
		toscan = append(toscan, gid)
	}

	// Find dependencies of core sets of glyph, and expand until have all relations.
	for len(toscan) > 0 {
		var newgids []GlyphIndex
		for _, gid := range toscan {
			components, err := f.glyf.GetComponents(gid)
			if err != nil {
				logrus.Debugf("Error getting components for %d", gid)
				return nil, err
			}
			for _, gid := range components {
				if _, has := gidIncludedMap[gid]; !has {
					gidIncludedMap[gid] = struct{}{}
					newgids = append(newgids, gid)
				}
			}
		}
		toscan = newgids
	}

	newfnt.ot = &offsetTable{}
	*newfnt.ot = *f.font.ot

	newfnt.trec = &tableRecords{}
	*newfnt.trec = *f.font.trec

	if f.font.head != nil {
		newfnt.head = &headTable{}
		*newfnt.head = *f.font.head
	}

	if f.font.maxp != nil {
		newfnt.maxp = &maxpTable{}
		*newfnt.maxp = *f.font.maxp
	}

	if f.font.hhea != nil {
		newfnt.hhea = &hheaTable{}
		*newfnt.hhea = *f.font.hhea
	}

	if f.font.hmtx != nil {
		newfnt.hmtx = &hmtxTable{}
		*newfnt.hmtx = *f.font.hmtx
		newfnt.optimizeHmtx()
	}

	if f.font.glyf != nil && f.font.loca != nil {
		newfnt.loca = &locaTable{}
		newfnt.glyf = &glyfTable{}
		*newfnt.glyf = *f.font.glyf

		// Empty glyf contents for non-included glyphs.
		for i := range newfnt.glyf.descs {
			if _, has := gidIncludedMap[GlyphIndex(i)]; has {
				continue
			}

			newfnt.glyf.descs[i].raw = nil
		}

		// Update loca offsets.
		isShort := f.font.head.indexToLocFormat == 0
		if isShort {
			newfnt.loca.offsetsShort = make([]offset16, len(newfnt.glyf.descs)+1)
			newfnt.loca.offsetsShort[0] = f.font.loca.offsetsShort[0]
		} else {
			newfnt.loca.offsetsLong = make([]offset32, len(newfnt.glyf.descs)+1)
			newfnt.loca.offsetsLong[0] = f.font.loca.offsetsLong[0]
		}
		for i, desc := range newfnt.glyf.descs {
			if isShort {
				newfnt.loca.offsetsShort[i+1] = newfnt.loca.offsetsShort[i] + offset16(len(desc.raw))/2
			} else {
				newfnt.loca.offsetsLong[i+1] = newfnt.loca.offsetsLong[i] + offset32(len(desc.raw))
			}
		}
	}

	if f.font.prep != nil {
		newfnt.prep = &prepTable{}
		*newfnt.prep = *f.font.prep
	}

	if f.font.cvt != nil {
		newfnt.cvt = &cvtTable{}
		*newfnt.cvt = *f.font.cvt
	}

	if f.font.fpgm != nil {
		newfnt.fpgm = &fpgmTable{}
		*newfnt.fpgm = *f.font.fpgm
	}

	if f.font.name != nil {
		newfnt.name = &nameTable{}
		*newfnt.name = *f.font.name
	}

	if f.font.os2 != nil {
		newfnt.os2 = &os2Table{}
		*newfnt.os2 = *f.font.os2
	}

	if f.font.post != nil {
		newfnt.post = &postTable{}
		*newfnt.post = *f.font.post
	}

	if f.font.cmap != nil {
		newfnt.cmap = &cmapTable{}
		*newfnt.cmap = *f.font.cmap
	}

	subfnt := &Font{
		br:   nil,
		font: &newfnt,
	}

	// Trim down to the first fonts.
	var maxgid GlyphIndex
	for gid := range gidIncludedMap {
		if gid > maxgid {
			maxgid = gid
		}
	}
	// Trim font down to only maximum needed glyphs without changing order.
	maxNeededNum := int(maxgid) + 1
	return subfnt.SubsetFirst(maxNeededNum)
}

// SubsetFirst creates a subset of `f` limited to only the first `numGlyphs` glyphs.
// Prunes out the glyphs from the previous font beyond that number.
// NOTE: If any of the first numGlyphs depend on later glyphs, it can lead to incorrect rendering.
func (f *Font) SubsetFirst(numGlyphs int) (*Font, error) {
	if int(f.maxp.numGlyphs) <= numGlyphs {
		logrus.Debugf("Attempting to subset font with same number of glyphs - Ignoring, returning same back")
		return f, nil
	}
	newfnt := font{}

	newfnt.ot = &offsetTable{}
	*newfnt.ot = *f.font.ot

	newfnt.trec = &tableRecords{}
	*newfnt.trec = *f.font.trec

	if f.font.head != nil {
		newfnt.head = &headTable{}
		*newfnt.head = *f.font.head
	}

	if f.font.maxp != nil {
		newfnt.maxp = &maxpTable{}
		*newfnt.maxp = *f.font.maxp
		newfnt.maxp.numGlyphs = uint16(numGlyphs)
	}
	if f.font.hhea != nil {
		newfnt.hhea = &hheaTable{}
		*newfnt.hhea = *f.font.hhea

		if newfnt.hhea.numberOfHMetrics > uint16(numGlyphs) {
			newfnt.hhea.numberOfHMetrics = uint16(numGlyphs)
		}
	}

	if f.font.hmtx != nil {
		newfnt.hmtx = &hmtxTable{}
		*newfnt.hmtx = *f.font.hmtx

		if len(newfnt.hmtx.hMetrics) > numGlyphs {
			newfnt.hmtx.hMetrics = newfnt.hmtx.hMetrics[0:numGlyphs]
			newfnt.hmtx.leftSideBearings = nil
		} else {
			numKeep := numGlyphs - len(newfnt.hmtx.hMetrics)
			if numKeep > len(newfnt.hmtx.leftSideBearings) {
				numKeep = len(newfnt.hmtx.leftSideBearings)
			}
			newfnt.hmtx.leftSideBearings = newfnt.hmtx.leftSideBearings[0:numKeep]
		}
		newfnt.optimizeHmtx()
	}

	if f.font.glyf != nil && f.font.loca != nil {
		newfnt.loca = &locaTable{}
		newfnt.glyf = &glyfTable{
			descs: f.font.glyf.descs[0:numGlyphs],
		}
		// Update loca offsets.
		isShort := f.font.head.indexToLocFormat == 0
		if isShort {
			newfnt.loca.offsetsShort = make([]offset16, numGlyphs+1)
			newfnt.loca.offsetsShort[0] = f.font.loca.offsetsShort[0]
		} else {
			newfnt.loca.offsetsLong = make([]offset32, numGlyphs+1)
			newfnt.loca.offsetsLong[0] = f.font.loca.offsetsLong[0]
		}
		for i, desc := range newfnt.glyf.descs {
			if isShort {
				newfnt.loca.offsetsShort[i+1] = newfnt.loca.offsetsShort[i] + offset16(len(desc.raw))/2
			} else {
				newfnt.loca.offsetsLong[i+1] = newfnt.loca.offsetsLong[i] + offset32(len(desc.raw))
			}
		}
	}

	if f.font.prep != nil {
		newfnt.prep = &prepTable{}
		*newfnt.prep = *f.font.prep
	}

	if f.font.cvt != nil {
		newfnt.cvt = &cvtTable{}
		*newfnt.cvt = *f.font.cvt
	}

	if f.font.fpgm != nil {
		newfnt.fpgm = &fpgmTable{}
		*newfnt.fpgm = *f.font.fpgm
	}

	if f.font.name != nil {
		newfnt.name = &nameTable{}
		*newfnt.name = *f.font.name
	}

	if f.font.os2 != nil {
		newfnt.os2 = &os2Table{}
		*newfnt.os2 = *f.font.os2
	}

	if f.font.post != nil {
		newfnt.post = &postTable{}
		*newfnt.post = *f.font.post

		if newfnt.post.numGlyphs > 0 {
			newfnt.post.numGlyphs = uint16(numGlyphs)
		}
		if len(newfnt.post.glyphNameIndex) > numGlyphs {
			newfnt.post.glyphNameIndex = newfnt.post.glyphNameIndex[0:numGlyphs]
		}
		if len(newfnt.post.offsets) > numGlyphs {
			newfnt.post.offsets = newfnt.post.offsets[0:numGlyphs]
		}
		if len(newfnt.post.glyphNames) > numGlyphs {
			newfnt.post.glyphNames = newfnt.post.glyphNames[0:numGlyphs]
		}
	}

	if f.font.cmap != nil {
		newfnt.cmap = &cmapTable{
			version:   f.cmap.version,
			subtables: map[string]*cmapSubtable{},
		}

		for _, name := range f.cmap.subtableKeys {
			subt := f.cmap.subtables[name]
			switch t := subt.ctx.(type) {
			case cmapSubtableFormat0:
				for i := range t.glyphIDArray {
					if i > numGlyphs {
						t.glyphIDArray[i] = 0
					}
				}
			case cmapSubtableFormat4:
				newt := cmapSubtableFormat4{}
				// Generates a new table: going from glyph index 0 up to numGlyphs.
				// Makes continous entries with deltas.
				// Does not use glyphIDData, but only the deltas.  Can lead to many segments, but should not
				// be too bad (especially since subsetting).
				charcodes := make([]CharCode, 0, len(subt.charcodeToGID))
				for cc, gid := range subt.charcodeToGID {
					if int(gid) >= numGlyphs {
						continue
					}
					charcodes = append(charcodes, cc)
				}
				sort.Slice(charcodes, func(i, j int) bool {
					return charcodes[i] < charcodes[j]
				})

				segments := 0
				i := 0
				for i < len(charcodes) {
					j := i + 1
					for ; j < len(charcodes); j++ {
						if int(charcodes[j]-charcodes[i]) != j-i ||
							int(subt.charcodeToGID[charcodes[j]]-subt.charcodeToGID[charcodes[i]]) != j-i {
							break
						}
					}
					// from i:j-1 maps to subt.charcodes[i]:subt.charcodes[i]+j-i-1
					startCode := uint16(charcodes[i])
					endCode := uint16(charcodes[i]) + uint16(j-i-1)
					idDelta := uint16(subt.charcodeToGID[charcodes[i]]) - uint16(charcodes[i])

					newt.startCode = append(newt.startCode, startCode)
					newt.endCode = append(newt.endCode, endCode)
					newt.idDelta = append(newt.idDelta, idDelta)
					newt.idRangeOffset = append(newt.idRangeOffset, 0)
					segments++
					i = j
				}

				if segments > 0 && newt.endCode[segments-1] < 65535 {
					newt.endCode = append(newt.endCode, 65535)
					newt.startCode = append(newt.startCode, 65535)
					newt.idDelta = append(newt.idDelta, 1)
					newt.idRangeOffset = append(newt.idRangeOffset, 0)
					segments++
				}

				newt.length = uint16(2*8 + 2*4*segments)
				newt.language = t.language
				newt.segCountX2 = uint16(segments * 2)
				newt.searchRange = 2 * uint16(math.Pow(2, math.Floor(math.Log2(float64(segments)))))
				newt.entrySelector = uint16(math.Log2(float64(newt.searchRange) / 2.0))
				newt.rangeShift = uint16(segments*2) - newt.searchRange
				subt.ctx = newt
			case cmapSubtableFormat6:
				for i := range t.glyphIDArray {
					if int(t.glyphIDArray[i]) >= numGlyphs {
						t.glyphIDArray[i] = 0
					}
				}
			case cmapSubtableFormat12:
				newt := cmapSubtableFormat12{}
				groups := 0

				charcodes := make([]CharCode, 0, len(subt.charcodeToGID))
				for cc, gid := range subt.charcodeToGID {
					if int(gid) >= numGlyphs {
						continue
					}
					charcodes = append(charcodes, cc)
				}
				sort.Slice(charcodes, func(i, j int) bool {
					return charcodes[i] < charcodes[j]
				})

				i := 0
				for i < len(charcodes) {
					j := i + 1
					for ; j < len(charcodes); j++ {
						if int(charcodes[j]-charcodes[i]) != j-i ||
							int(subt.charcodeToGID[charcodes[j]]-subt.charcodeToGID[charcodes[i]]) != j-i {
							break
						}
					}
					// from i:j-1 maps to subt.charcodes[i]:subt.charcodes[i]+j-i-1
					startCharCode := uint32(charcodes[i])
					endCharCode := uint32(charcodes[i]) + uint32(j-i-1)
					startGlyphID := uint32(subt.charcodeToGID[charcodes[i]])

					group := sequentialMapGroup{
						startCharCode: startCharCode,
						endCharCode:   endCharCode,
						startGlyphID:  startGlyphID,
					}
					newt.groups = append(newt.groups, group)
					groups++
					i = j
				}

				newt.length = uint32(2*2 + 3*4 + groups*3*4)
				newt.language = t.language
				newt.numGroups = uint32(groups)
				subt.ctx = newt
			}

			newfnt.cmap.subtableKeys = append(newfnt.cmap.subtableKeys, name)
			newfnt.cmap.subtables[name] = subt
		}
		newfnt.cmap.numTables = uint16(len(newfnt.cmap.subtables))
	}

	subfnt := &Font{
		br:   nil,
		font: &newfnt,
	}
	return subfnt, nil
}

// Subset creates a subset of `f` including only glyph indices specified by `indices`.
// Returns the new subsetted font, a map of old to new GlyphIndex to GlyphIndex as the removal
// of glyphs requires reordering.
func (f *Font) Subset(indices []GlyphIndex) (newf *Font, oldnew map[GlyphIndex]GlyphIndex, err error) {
	// TODO:
	//     1. Make the new cmap for `runes` if `cmap` is nil, using the cmap table and make a []GlyphIndex
	//        with the glyph indices to keep (index prior to subsetting).
	//     2. Go through each table and leave only data for the glyph indices to be kept.
	return nil, nil, errors.New("not implemented yet")
}

// PruneTables prunes font tables `tables` by name from font.
// Currently supports: "cmap", "post", "name".
func (f *Font) PruneTables(tables ...string) error {
	for _, table := range tables {
		switch table {
		case "cmap":
			f.cmap = nil
		case "post":
			f.post = nil
		case "name":
			f.name = nil
		}
	}
	return nil
}

// Optimize does some optimization such as reducing hmtx table.
func (f *Font) Optimize() error {
	f.optimizeHmtx()
	return nil
}

// Write writes the font to `w`.
func (f *Font) Write(w io.Writer) error {
	bw := newByteWriter(w)
	err := f.font.write(bw)
	if err != nil {
		return err
	}
	return bw.flush()
}

// WriteFile writes the font to `outPath`.
func (f *Font) WriteFile(outPath string) error {
	of, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer of.Close()

	return f.Write(of)
}
