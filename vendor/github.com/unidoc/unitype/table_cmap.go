/*
 * This file is subject to the terms and conditions defined in
 * file 'LICENSE.md', which is part of this source code package.
 */

package unitype

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"
)

// cmapTable represents a Character to Glyph Index Mapping Table (cmap).
// This table defines the mapping of character codes to the glyph index values used
// in the font.
// https://docs.microsoft.com/en-us/typography/opentype/spec/cmap
type cmapTable struct {
	version   uint16
	numTables uint16

	// TODO: Only keep records that are used for parsing and writing back out.
	encodingRecords []encodingRecord // len == numTables

	// Processed data:
	subtables    map[string]*cmapSubtable
	subtableKeys []string // "format,platformID,encodingID".
}

type encodingRecord struct {
	platformID uint16
	encodingID uint16
	offset     offset32
}

func (f *font) parseCmap(r *byteReader) (*cmapTable, error) {
	if f.maxp == nil {
		logrus.Debug("Unable to load cmap: maxp table is nil")
		return nil, errRequiredField
	}

	tr, has, err := f.seekToTable(r, "cmap")
	if err != nil {
		return nil, err
	}
	if !has {
		logrus.Debug("cmap table absent")
		return nil, nil
	}

	t := &cmapTable{}
	t.subtables = map[string]*cmapSubtable{}
	err = r.read(&t.version, &t.numTables)
	if err != nil {
		return nil, err
	}

	for i := 0; i < int(t.numTables); i++ {
		var enc encodingRecord
		err = r.read(&enc.platformID, &enc.encodingID, &enc.offset)
		if err != nil {
			return nil, err
		}
		t.encodingRecords = append(t.encodingRecords, enc)
	}

	// Process the encoding subtables.
	for _, enc := range t.encodingRecords {
		// Seek to the subtable.
		err = r.SeekTo(int64(tr.offset) + int64(enc.offset))
		if err != nil {
			return nil, err
		}

		// Header.
		var format uint16
		err = r.read(&format)
		if err != nil {
			return nil, err
		}

		logrus.Debugf("Format: %d", format)
		var cmap *cmapSubtable
		switch format {
		case 0:
			cmap, err = f.parseCmapSubtableFormat0(r, int(enc.platformID), int(enc.encodingID))
		case 4:
			cmap, err = f.parseCmapSubtableFormat4(r, int(enc.platformID), int(enc.encodingID))
		case 6:
			cmap, err = f.parseCmapSubtableFormat6(r, int(enc.platformID), int(enc.encodingID))
		case 12:
			cmap, err = f.parseCmapSubtableFormat12(r, int(enc.platformID), int(enc.encodingID))
		default:
			logrus.Debugf("Unsupported cmap format %d", format)
			continue
		}
		if err != nil {
			logrus.Debugf("Error: %v", err)
			return nil, err
		}
		if cmap != nil {
			key := fmt.Sprintf("%d,%d,%d", format, enc.platformID, enc.encodingID)
			t.subtables[key] = cmap
			t.subtableKeys = append(t.subtableKeys, key)
			logrus.Debugf("KEY: %s <-> %T", key, cmap.ctx)
		}
	}

	return t, nil
}

// cmap subtable data.
type cmapSubtable struct {
	format     int
	platformID int
	encodingID int

	ctx interface{} // The specific subtable, e.g. cmapSubtableFormat0, etc.

	// TODO: Need GID to rune map too? or just a list of runes (with length = numGlyphs, i.e. one rune per gid)
	cmap                map[rune]GlyphIndex
	runes               []rune
	charcodes           []CharCode
	charcodeToGID       map[CharCode]GlyphIndex
	runeToCharcodeBytes map[rune][]byte // Quick for going rune -> encoded bytes (charcodes).
}

// cmapSubtableFormat0 represents format 0: Byte encoding table.
// This is the Apple standard character to glyph index mapping table.
type cmapSubtableFormat0 struct {
	length       uint16
	language     uint16
	glyphIDArray []uint8 // len = 256.
}

func (f *font) parseCmapSubtableFormat0(r *byteReader, platformID, encodingID int) (*cmapSubtable, error) {
	st := cmapSubtableFormat0{}
	err := r.read(&st.length, &st.language)
	if err != nil {
		return nil, err
	}
	if st.length != 262 {
		err := f.recordIncompatibilityf("length != 262 (%d)", st.length)
		if err != nil {
			return nil, err
		}
	}

	err = r.readSlice(&st.glyphIDArray, 256)
	if err != nil {
		return nil, err
	}

	encoding := getCmapEncoding(platformID, encodingID)
	runeDecoder := encoding.GetRuneDecoder()

	// TODO: Reduce these maps to minimum needed. The raw info is the charcode to GID mapping.
	//   Certain implementations preserve minimal info and a conversion function.
	//   The encoding determines the number of bytes per charcode and mapping to rune.
	//   (cmapEncoder).
	cmap := map[rune]GlyphIndex{}
	runes := make([]rune, len(st.glyphIDArray))
	runeToCharcodeBytes := map[rune][]byte{}
	charcodes := make([]CharCode, len(st.glyphIDArray))
	charcodeToGID := map[CharCode]GlyphIndex{}

	for glyphID, code := range st.glyphIDArray {
		charcodeToGID[CharCode(code)] = GlyphIndex(glyphID)
		codeBytes := runeDecoder.ToBytes(uint32(code))
		r := runeDecoder.DecodeRune(codeBytes)
		runes[glyphID] = r
		charcodes[glyphID] = CharCode(code)
		if _, has := cmap[r]; !has {
			// Avoid overwrite, if get same twice, use the earlier entry.
			cmap[r] = GlyphIndex(glyphID)
			runeToCharcodeBytes[r] = codeBytes
		}
	}

	return &cmapSubtable{
		format:              0,
		platformID:          platformID,
		encodingID:          encodingID,
		cmap:                cmap,
		runes:               runes,
		runeToCharcodeBytes: runeToCharcodeBytes,
		charcodes:           charcodes,
		charcodeToGID:       charcodeToGID,
		ctx:                 st,
	}, nil
}

func writeCmapSubtableFormat0(subtable *cmapSubtable, w *byteWriter) error {
	subt := subtable.ctx.(cmapSubtableFormat0)
	var (
		format uint16
	)
	subt.length = 3*2 + 256
	err := w.write(format, subt.length, subt.language)
	if err != nil {
		return err
	}

	return w.writeSlice(subt.glyphIDArray)
}

// cmapSubtableFormat4 represents cmap data format 4: Segment mapping to delta values.
// This is the standard character-to-glyph index mapping for the Windows platform for fonts that
// support Unicode BMP characters.
// https://docs.microsoft.com/en-us/typography/opentype/spec/cmap#format-4-segment-mapping-to-delta-values
// [platformID=3 (Windows)].
type cmapSubtableFormat4 struct {
	length        uint16
	language      uint16
	segCountX2    uint16 // 2 * segCount
	searchRange   uint16
	entrySelector uint16
	rangeShift    uint16
	endCode       []uint16 // len = segCount
	reservedPad   uint16
	startCode     []uint16 // len = segCount. Start character code for each segment.
	idDelta       []uint16 // len = segCount. Delta for all character codes in segment.
	idRangeOffset []uint16 // len = segCount. offsets into glyphIDArray or 0.
	glyphIDArray  []uint16 // len = variable.
}

func (f *font) parseCmapSubtableFormat4(r *byteReader, platformID, encodingID int) (*cmapSubtable, error) {
	//refStart := r.Offset()
	st := cmapSubtableFormat4{}
	err := r.read(&st.length, &st.language, &st.segCountX2, &st.searchRange, &st.entrySelector, &st.rangeShift)
	if err != nil {
		return nil, err
	}

	segCount := int(st.segCountX2 / 2)

	err = r.readSlice(&st.endCode, segCount)
	if err != nil {
		return nil, err
	}
	err = r.read(&st.reservedPad)
	if err != nil {
		return nil, err
	}

	err = r.readSlice(&st.startCode, segCount)
	if err != nil {
		return nil, err
	}
	err = r.readSlice(&st.idDelta, segCount)
	if err != nil {
		return nil, err
	}

	err = r.readSlice(&st.idRangeOffset, segCount)
	if err != nil {
		return nil, err
	}

	glyphIDArrLen := int(st.length-uint16(2*8+2*4*segCount)) / 2
	logrus.Debugf("Parsing cmap format 4, segCount: %d", segCount)
	logrus.Debugf("Table len: %d", st.length)
	logrus.Debugf("glyphIDArrLen: %d", glyphIDArrLen)
	if glyphIDArrLen < 0 {
		return nil, errors.New("invalid length")
	}
	err = r.readSlice(&st.glyphIDArray, glyphIDArrLen)
	if err != nil {
		return nil, err
	}

	encoding := getCmapEncoding(platformID, encodingID)
	runeDecoder := encoding.GetRuneDecoder()

	cmap := map[rune]GlyphIndex{}
	runes := make([]rune, int(f.maxp.numGlyphs))
	charcodes := make([]CharCode, int(f.maxp.numGlyphs))
	charcodeMap := make(map[CharCode]GlyphIndex, f.maxp.numGlyphs)
	logrus.Debugf("Number of glyphs in font: %d\n", f.maxp.numGlyphs)
	for i := 0; i < segCount-1; i++ {
		c1 := st.startCode[i]
		c2 := st.endCode[i]
		d := st.idDelta[i]
		rangeOffset := st.idRangeOffset[i]

		logrus.Debugf("Segment %d/%d, c1: %d, c2: %d, d: %d, rangeOffset: %d", i+1, segCount, c1, c2, d, rangeOffset)

		for c := c1; c <= c2; c++ {
			var gid uint16

			if rangeOffset == 0 {
				gid = (c + d) & 0xFFFF
			} else {
				index := int(rangeOffset/2 + (c - c1) + uint16(i) - uint16(len(st.idRangeOffset)))

				if index >= len(st.glyphIDArray) {
					logrus.Debugf("c1=%d c=%d c2=%d", c1, c, c2)
					logrus.Debugf("ERROR: index outside bounds (%d/%d)", index, len(st.glyphIDArray))
					return nil, errors.New("outside bounds")
				}
				if st.glyphIDArray[index] != 0 {
					gid = (st.glyphIDArray[index] + d) & 0xFFFF
				} else {
					gid = 0
				}
			}

			logrus.Tracef("Charcode:GID - %d:%d", c, gid)

			if gid > 0 {
				b := runeDecoder.ToBytes(uint32(c))
				r := runeDecoder.DecodeRune(b)
				if int(gid) >= int(f.maxp.numGlyphs) {
					logrus.Debugf("ERROR: gid > numGlyphs (%d > %d)", gid, f.maxp.numGlyphs)
					return nil, errors.New("gid out of range")
				}
				runes[int(gid)] = r
				charcodes[int(gid)] = CharCode(c)
				charcodeMap[CharCode(c)] = GlyphIndex(gid)

				if _, has := cmap[r]; !has {
					// Avoid overwrite, if get same twice, use the earlier entry.
					cmap[r] = GlyphIndex(gid)
				}
			}
		}
	}

	return &cmapSubtable{
		format:        4,
		platformID:    platformID,
		encodingID:    encodingID,
		cmap:          cmap,
		charcodes:     charcodes,
		charcodeToGID: charcodeMap,
		runes:         runes,
		ctx:           st,
	}, nil
}

func writeCmapSubtableFormat4(subtable *cmapSubtable, w *byteWriter) error {
	subt := subtable.ctx.(cmapSubtableFormat4)
	var (
		format uint16
	)
	format = 4
	// TODO(gunnsth): Not the place to generate this?  Somewhere else should have ability to generate
	//       based on character codes.
	subt.length = 7*2 + subt.segCountX2 + 2 + 3*subt.segCountX2 + 2*uint16(len(subt.glyphIDArray))
	err := w.write(format, subt.length, subt.language)
	if err != nil {
		return err
	}
	err = w.write(subt.segCountX2, subt.searchRange, subt.entrySelector, subt.rangeShift)
	if err != nil {
		return err
	}
	err = w.writeSlice(subt.endCode)
	if err != nil {
		return err
	}
	err = w.write(subt.reservedPad)
	if err != nil {
		return nil
	}
	err = w.writeSlice(subt.startCode)
	if err != nil {
		return nil
	}
	err = w.writeSlice(subt.idDelta)
	if err != nil {
		return nil
	}
	err = w.writeSlice(subt.idRangeOffset)
	if err != nil {
		return nil
	}
	// TODO: Problem: the following slice is not populated.
	return w.writeSlice(subt.glyphIDArray)
}

// cmapSubtableFormat6 represents cmap data format 6: Trimmed table mapping.
type cmapSubtableFormat6 struct {
	length       uint16
	language     uint16
	firstCode    uint16
	entryCount   uint16
	glyphIDArray []uint16 // len = entryCount
}

func (f *font) parseCmapSubtableFormat6(r *byteReader, platformID, encodingID int) (*cmapSubtable, error) {
	st := cmapSubtableFormat6{}
	err := r.read(&st.length, &st.language, &st.firstCode, &st.entryCount)
	if err != nil {
		return nil, err
	}

	err = r.readSlice(&st.glyphIDArray, int(st.entryCount))
	if err != nil {
		return nil, err
	}

	encoding := getCmapEncoding(platformID, encodingID)
	runeDecoder := encoding.GetRuneDecoder()

	cmap := map[rune]GlyphIndex{}
	runes := make([]rune, st.entryCount)
	charcodes := make([]CharCode, st.entryCount)
	charcodeMap := make(map[CharCode]GlyphIndex, st.entryCount)
	for i := 0; i < int(st.entryCount); i++ {
		gid := GlyphIndex(st.glyphIDArray[i])
		code := st.firstCode + uint16(i)
		b := runeDecoder.ToBytes(uint32(code))
		r := runeDecoder.DecodeRune(b)
		runes[i] = r
		charcodes[i] = CharCode(code)
		charcodeMap[CharCode(code)] = gid
		if _, has := cmap[r]; !has {
			// Avoid ovewriting (stick to first gid).
			cmap[r] = gid
		}
	}

	return &cmapSubtable{
		format:        6,
		platformID:    platformID,
		encodingID:    encodingID,
		cmap:          cmap,
		runes:         runes,
		charcodes:     charcodes,
		charcodeToGID: charcodeMap,
		ctx:           st,
	}, nil
}

func writeCmapSubtableFormat6(subtable *cmapSubtable, w *byteWriter) error {
	subt := subtable.ctx.(cmapSubtableFormat6)
	var (
		format uint16
	)
	format = 6
	subt.length = 5*2 + 2*uint16(len(subt.glyphIDArray))
	err := w.write(format, subt.length, subt.language, subt.firstCode, subt.entryCount)
	if err != nil {
		return err
	}

	return w.writeSlice(subt.glyphIDArray)
}

// cmapSubtableFormat12 represents cmap data format 12: Segmented coverage.
// Format 12 is similar to format 4 in that it defines segments for sparse representation.
// It differs, however, in that it uses 32-bit character codes.
type cmapSubtableFormat12 struct {
	reserved  uint16
	length    uint32
	language  uint32
	numGroups uint32
	groups    []sequentialMapGroup // length = numGroups.
}

type sequentialMapGroup struct {
	startCharCode uint32 // First character code in this group.
	endCharCode   uint32 // Last character code in this group.
	startGlyphID  uint32 // Glyph index corresponding to the starting character code.
}

func (f *font) parseCmapSubtableFormat12(r *byteReader, platformID, encodingID int) (*cmapSubtable, error) {
	st := cmapSubtableFormat12{}
	err := r.read(&st.reserved, &st.length, &st.language, &st.numGroups)
	if err != nil {
		logrus.Debugf("Error: %v", err)
		return nil, err
	}

	for i := 0; i < int(st.numGroups); i++ {
		var group sequentialMapGroup
		err = r.read(&group.startCharCode, &group.endCharCode, &group.startGlyphID)
		if err != nil {
			logrus.Debugf("Error: %v", err)
			return nil, err
		}
		st.groups = append(st.groups, group)
	}

	encoding := getCmapEncoding(platformID, encodingID)
	runeDecoder := encoding.GetRuneDecoder()

	cmap := map[rune]GlyphIndex{}
	runes := make([]rune, f.maxp.numGlyphs)
	charcodes := make([]CharCode, f.maxp.numGlyphs)
	charcodeMap := make(map[CharCode]GlyphIndex, f.maxp.numGlyphs)
	for _, group := range st.groups {
		gid := GlyphIndex(group.startGlyphID)
		if int(gid) >= int(f.maxp.numGlyphs) {
			logrus.Debugf("gid >= numGlyphs (%d > %d)", gid, f.maxp.numGlyphs)
			logrus.Debugf("Error: %v", errRangeCheck)
			return nil, errRangeCheck
		}
		for charcode := group.startCharCode; charcode <= group.endCharCode; charcode++ {
			if int(gid) >= int(f.maxp.numGlyphs) {
				break
			}
			b := runeDecoder.ToBytes(charcode)
			r := runeDecoder.DecodeRune(b)
			runes[gid] = r
			charcodes[gid] = CharCode(charcode)
			charcodeMap[CharCode(charcode)] = gid
			if _, has := cmap[r]; !has {
				// Avoid overwrite, if get same twice, use the earlier entry.
				cmap[r] = gid
			}
			gid++
		}
	}

	return &cmapSubtable{
		format:        12,
		ctx:           st,
		platformID:    platformID,
		encodingID:    encodingID,
		cmap:          cmap,
		runes:         runes,
		charcodes:     charcodes,
		charcodeToGID: charcodeMap,
	}, nil
}

func writeCmapSubtableFormat12(subtable *cmapSubtable, w *byteWriter) error {
	subt := subtable.ctx.(cmapSubtableFormat12)
	var (
		format uint16
	)
	format = 12
	subt.length = 2*2 + 3*4 + uint32(len(subt.groups))*3*4
	err := w.write(format, subt.reserved, subt.length, subt.language, subt.numGroups)
	if err != nil {
		return err
	}

	for _, group := range subt.groups {
		logrus.Tracef("XXX Write, startCharcode: %d, endCharCode: %d, startGlyphID: %d", group.startCharCode, group.endCharCode, group.startGlyphID)
		err = w.write(group.startCharCode, group.endCharCode, group.startGlyphID)
		if err != nil {
			return err
		}
	}

	return nil
}

func (f *font) writeCmap(w *byteWriter) error {
	if f.cmap == nil {
		return nil
	}
	t := f.cmap

	err := w.write(t.version, t.numTables)

	// Write the cmap subtables to an in-memory mock buffer to calculate offsets.
	var mockBuffer bytes.Buffer
	mockWriter := newByteWriter(&mockBuffer)

	var encodingRecords []encodingRecord
	for _, subtkey := range t.subtableKeys {
		subt := t.subtables[subtkey]
		rec := encodingRecord{
			platformID: uint16(subt.platformID),
			encodingID: uint16(subt.encodingID),
			offset:     offset32(mockWriter.bufferedLen()),
		}

		supported := true
		switch subt.format {
		case 0:
			err := writeCmapSubtableFormat0(subt, mockWriter)
			if err != nil {
				return err
			}
		case 4:
			err := writeCmapSubtableFormat4(subt, mockWriter)
			if err != nil {
				return err
			}
		case 6:
			err := writeCmapSubtableFormat6(subt, mockWriter)
			if err != nil {
				return err
			}
		case 12:
			err := writeCmapSubtableFormat12(subt, mockWriter)
			if err != nil {
				return err
			}
		default:
			supported = false
		}

		if supported {
			encodingRecords = append(encodingRecords, rec)
		}
	}
	err = mockWriter.flush()
	if err != nil {
		return err
	}

	// Output the encoding records and the mock buffer.
	for _, rec := range encodingRecords {
		rec.offset += offset32(4 + 8*len(encodingRecords)) // Add static part.
		err := w.write(rec.platformID, rec.encodingID, rec.offset)
		if err != nil {
			return err
		}
	}
	return w.writeBytes(mockBuffer.Bytes())
}
