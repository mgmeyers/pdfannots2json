/*
 * This file is subject to the terms and conditions defined in
 * file 'LICENSE.md', which is part of this source code package.
 */

package unitype

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/sirupsen/logrus"
)

// Export what UniPDF needs.
// font flags:
//		IsFixedPitch, Serif, etc (Table 123 PDF32000_2008 - font flags)
//		FixedPitch() bool
//		Serif() bool
//		Symbolic() bool
//		Script() bool
//		Nonsymbolic() bool
//		Italic() bool
//		AllCap() bool
//		SmallCap() bool
//		ForceBold() bool
//      Need to be able to derive the font flags from the font to build a font descriptor
//
// Required table according to PDF32000_2008 (9.9 Embedded font programs - p. 299):
// “head”, “hhea”, “loca”, “maxp”, “cvt”, “prep”, “glyf”, “hmtx”, and “fpgm”. If used with a simple
// font dictionary, the font program shall additionally contain a cmap table defining one or more
// encodings, as discussed in 9.6.6.4, "Encodings for TrueType Fonts". If used with a CIDFont
// dictionary, the cmap table is not needed and shall not be present, since the mapping from
// character codes to glyph descriptions is provided separately.
//

// font is a data model for truetype fonts with basic access methods.
type font struct {
	strict            bool
	incompatibilities []string

	ot   *offsetTable
	trec *tableRecords // table records (references other tables).
	head *headTable
	hhea *hheaTable
	loca *locaTable
	maxp *maxpTable
	cvt  *cvtTable
	fpgm *fpgmTable
	prep *prepTable
	glyf *glyfTable
	hmtx *hmtxTable
	name *nameTable
	os2  *os2Table
	post *postTable
	cmap *cmapTable
}

// Returns an error in strict mode, otherwise adds the incompatibility to a list of noted incompatibilities.
func (f *font) recordIncompatibilityf(fmtstr string, a ...interface{}) error {
	str := fmt.Sprintf(fmtstr, a...)
	if f.strict {
		return fmt.Errorf("incompatibility: %s", str)
	}
	f.incompatibilities = append(f.incompatibilities, str)
	return nil
}

func (f font) numTables() int {
	return int(f.ot.numTables)
}

func parseFont(r *byteReader) (*font, error) {
	f := &font{}

	var err error

	// Load table offsets and records.
	f.ot, err = f.parseOffsetTable(r)
	if err != nil {
		return nil, err
	}

	f.trec, err = f.parseTableRecords(r)
	if err != nil {
		return nil, err
	}

	f.head, err = f.parseHead(r)
	if err != nil {
		return nil, err
	}

	f.maxp, err = f.parseMaxp(r)
	if err != nil {
		return nil, err
	}

	f.hhea, err = f.parseHhea(r)
	if err != nil {
		return nil, err
	}

	f.hmtx, err = f.parseHmtx(r)
	if err != nil {
		return nil, err
	}

	f.loca, err = f.parseLoca(r)
	if err != nil {
		return nil, err
	}

	f.glyf, err = f.parseGlyf(r)
	if err != nil {
		return nil, err
	}

	f.prep, err = f.parsePrep(r)
	if err != nil {
		return nil, err
	}

	f.name, err = f.parseNameTable(r)
	if err != nil {
		return nil, err
	}

	f.os2, err = f.parseOS2Table(r)
	if err != nil {
		return nil, err
	}

	f.post, err = f.parsePost(r)
	if err != nil {
		return nil, err
	}

	f.cmap, err = f.parseCmap(r)
	if err != nil {
		return nil, err
	}

	f.cvt, err = f.parseCvt(r)
	if err != nil {
		return nil, err
	}

	f.fpgm, err = f.parseFpgm(r)
	if err != nil {
		return nil, err
	}

	return f, nil
}

// numTablesToWrite returns the number of tables in `f`.
// Calculates based on the number of tables will be written out.
// NOTE that not all tables that are loaded are written out.
func (f *font) numTablesToWrite() int {
	var num int

	if f.head != nil {
		num++
	}
	if f.maxp != nil {
		num++
	}
	if f.hhea != nil {
		num++
	}
	if f.hmtx != nil {
		num++
	}
	if f.loca != nil {
		num++
	}
	if f.glyf != nil {
		num++
	}
	if f.cvt != nil {
		num++
	}
	if f.fpgm != nil {
		num++
	}
	if f.prep != nil {
		num++
	}
	if f.name != nil {
		num++
	}
	if f.os2 != nil {
		num++
	}
	if f.post != nil {
		num++
	}
	if f.cmap != nil {
		num++
	}
	return num
}

func (f *font) write(w *byteWriter) error {
	logrus.Debug("Writing font")
	numTables := f.numTablesToWrite()
	otTable := &offsetTable{
		sfntVersion:   f.ot.sfntVersion,
		numTables:     uint16(numTables),
		searchRange:   f.ot.searchRange,
		entrySelector: f.ot.entrySelector,
		rangeShift:    f.ot.rangeShift,
	}
	trec := &tableRecords{}

	f.ot.numTables = uint16(numTables)

	// Starting offset after offset table and table records.
	startOffset := int64(12 + numTables*16)

	logrus.Tracef("==== write\nnumTables: %d\nstartOffset: %d", numTables, startOffset)
	logrus.Trace("Write 2")
	// Writing is two phases and is done in a few steps:
	// 1. Write the content tables: head, hhea, etc in the expected order and keep track of the length, checksum for each.
	// 2. Generate the table records based on the information.
	// 3. Write out in final order: offset table, table records, head, ...
	// 4. Set checkAdjustment of head table based on checksumof entire file
	// 5. Write the final output

	// Write to buffer to get offsets.
	var buf bytes.Buffer
	var headChecksum uint32
	{
		bufw := newByteWriter(&buf)

		// head.
		f.head.checksumAdjustment = 0
		offset := startOffset
		err := f.writeHead(bufw)
		if err != nil {
			return err
		}
		headChecksum = bufw.checksum()
		trec.Set("head", offset, bufw.bufferedLen(), headChecksum)
		err = bufw.flush()
		if err != nil {
			return err
		}

		// maxp.
		offset = startOffset + bufw.flushedLen
		err = f.writeMaxp(bufw)
		if err != nil {
			return err
		}
		trec.Set("maxp", offset, bufw.bufferedLen(), bufw.checksum())
		err = bufw.flush()
		if err != nil {
			return err
		}

		// hhea.
		if f.hhea != nil {
			offset = startOffset + bufw.flushedLen
			err = f.writeHhea(bufw)
			if err != nil {
				return err
			}
			trec.Set("hhea", offset, bufw.bufferedLen(), bufw.checksum())
			err = bufw.flush()
			if err != nil {
				return err
			}
		}

		// hmtx.
		if f.hmtx != nil {
			offset = startOffset + bufw.flushedLen
			err = f.writeHmtx(bufw)
			if err != nil {
				return err
			}
			trec.Set("hmtx", offset, bufw.bufferedLen(), bufw.checksum())
			err = bufw.flush()
			if err != nil {
				return err
			}
		}

		// loca.
		if f.loca != nil {
			offset = startOffset + bufw.flushedLen
			err = f.writeLoca(bufw)
			if err != nil {
				return err
			}
			trec.Set("loca", offset, bufw.bufferedLen(), bufw.checksum())
			err = bufw.flush()
			if err != nil {
				return err
			}
		}

		// glyf.
		if f.glyf != nil {
			offset = startOffset + bufw.flushedLen
			err = f.writeGlyf(bufw)
			if err != nil {
				return err
			}
			trec.Set("glyf", offset, bufw.bufferedLen(), bufw.checksum())
			err = bufw.flush()
			if err != nil {
				return err
			}
		}

		// prep.
		if f.prep != nil {
			offset = startOffset + bufw.flushedLen
			err = f.writePrep(bufw)
			if err != nil {
				return err
			}
			trec.Set("prep", offset, bufw.bufferedLen(), bufw.checksum())
			err = bufw.flush()
			if err != nil {
				return err
			}
		}

		// cvt.
		if f.cvt != nil {
			offset = startOffset + bufw.flushedLen
			err = f.writeCvt(bufw)
			if err != nil {
				return err
			}
			trec.Set("cvt", offset, bufw.bufferedLen(), bufw.checksum())
			err = bufw.flush()
			if err != nil {
				return err
			}
		}

		// fpgm.
		if f.fpgm != nil {
			offset = startOffset + bufw.flushedLen
			err = f.writeFpgm(bufw)
			if err != nil {
				return err
			}
			trec.Set("fpgm", offset, bufw.bufferedLen(), bufw.checksum())
			err = bufw.flush()
			if err != nil {
				return err
			}
		}

		// name.
		if f.name != nil {
			offset = startOffset + bufw.flushedLen
			err = f.writeNameTable(bufw)
			if err != nil {
				return err
			}
			trec.Set("name", offset, bufw.bufferedLen(), bufw.checksum())
			err = bufw.flush()
			if err != nil {
				return err
			}
		}

		// os2.
		if f.os2 != nil {
			offset = startOffset + bufw.flushedLen
			err = f.writeOS2(bufw)
			if err != nil {
				return err
			}
			trec.Set("OS/2", offset, bufw.bufferedLen(), bufw.checksum())
			err = bufw.flush()
			if err != nil {
				return err
			}
		}

		// post
		if f.post != nil {
			offset = startOffset + bufw.flushedLen
			err = f.writePost(bufw)
			if err != nil {
				return err
			}
			trec.Set("post", offset, bufw.bufferedLen(), bufw.checksum())
			err = bufw.flush()
			if err != nil {
				return err
			}
		}

		// cmap
		if f.cmap != nil {
			offset = startOffset + bufw.flushedLen
			err = f.writeCmap(bufw)
			if err != nil {
				return err
			}
			trec.Set("cmap", offset, bufw.bufferedLen(), bufw.checksum())
			err = bufw.flush()
			if err != nil {
				return err
			}
		}
	}
	logrus.Trace("Write 3")

	// Write the offset and table records to another mock buffer.
	var bufh bytes.Buffer
	{
		bufw := newByteWriter(&bufh)
		// Create a mock font for writing without modifying the original entries of `f`.
		mockf := &font{
			ot:   otTable,
			trec: trec,
		}

		err := mockf.writeOffsetTable(bufw)
		if err != nil {
			return err
		}

		err = mockf.writeTableRecords(bufw)
		if err != nil {
			return err
		}
		err = bufw.flush()
		if err != nil {
			return err
		}
	}

	// Write everything to bufh.
	_, err := buf.WriteTo(&bufh)
	if err != nil {
		return err
	}

	// Calculate total checksum for the entire font.
	checksummer := byteWriter{
		buffer: bufh,
	}
	fontChecksum := checksummer.checksum()
	checksumAdjustment := 0xB1B0AFBA - fontChecksum

	// Set the checksumAdjustment of the head table.
	data := bufh.Bytes()
	hoff := startOffset
	binary.BigEndian.PutUint32(data[hoff+8:hoff+12], checksumAdjustment)

	buffer := bytes.NewBuffer(data)
	_, err = io.Copy(&w.buffer, buffer)
	return err
}

// TableInfo provides readable information regarding a table.
func (f *font) TableInfo(table string) string {
	var b bytes.Buffer

	switch table {
	case "trec":
		if f.trec == nil {
			b.WriteString(fmt.Sprintf("trec: missing\n"))
			break
		}
		b.WriteString(fmt.Sprintf("trec: present with %d table records\n", len(f.trec.list)))
		for _, tr := range f.trec.list {
			if tr.length > 1024*1024 {
				b.WriteString(fmt.Sprintf("%s: %.2f MB\n", tr.tableTag.String(), float64(tr.length)/1024/1024))
			} else if tr.length > 1024 {
				b.WriteString(fmt.Sprintf("%s: %.2f kB\n", tr.tableTag.String(), float64(tr.length)/1024))
			} else {
				b.WriteString(fmt.Sprintf("%s: %d B\n", tr.tableTag.String(), tr.length))
			}
		}
		b.WriteString("--\n")
	case "head":
		if f.head == nil {
			b.WriteString("head: missing\n")
			break
		}
		b.WriteString(fmt.Sprintf("head table: %#v\n", f.head))
	case "os2":
		if f.os2 == nil {
			b.WriteString("os2: missing\n")
			break
		}
		b.WriteString(fmt.Sprintf("os/2 table: %#v\n", f.os2))
	case "hhea":
		if f.hhea == nil {
			b.WriteString("hhea: missing\n")
			break
		}
		b.WriteString(fmt.Sprintf("hhea table: numHMetrics: %d\n", f.hhea.numberOfHMetrics))
	case "hmtx":
		if f.hmtx == nil {
			b.WriteString("hmtx: missing\n")
			break
		}
		b.WriteString(fmt.Sprintf("hmtx: hmetrics: %d, leftSideBearings: %d\n",
			len(f.hmtx.hMetrics), len(f.hmtx.leftSideBearings)))
	case "cmap":
		if f.cmap == nil {
			b.WriteString("cmap: missing\n")
			break
		}
		b.WriteString(fmt.Sprintf("cmap version: %d\n",
			f.cmap.version))
		b.WriteString(fmt.Sprintf("cmap: encoding records: %d subtables: %d\n",
			len(f.cmap.encodingRecords), len(f.cmap.subtables)))
		b.WriteString(fmt.Sprintf("cmap: subtables: %+v\n", f.cmap.subtableKeys))
		for _, k := range f.cmap.subtableKeys {
			subt := f.cmap.subtables[k]
			b.WriteString(fmt.Sprintf("cmap subtable: %s: runes: %d\n", k, len(subt.runes)))
			for i := range subt.charcodes {
				b.WriteString(fmt.Sprintf("\t%d - Charcode %d (0x%X) - rune % X\n", i, subt.charcodes[i], subt.charcodes[i], subt.runes[i]))
			}
		}
	case "loca":
		if f.loca == nil {
			b.WriteString("loca: missing\n")
			break
		}
		b.WriteString(fmt.Sprintf("Loca table\n"))
		b.WriteString(fmt.Sprintf("- Short offsets: %d\n", len(f.loca.offsetsShort)))
		b.WriteString(fmt.Sprintf("- Long offsets: %d\n", len(f.loca.offsetsLong)))
	case "name":
		if f.name == nil {
			b.WriteString("name: missing\n")
			break
		}
		b.WriteString(fmt.Sprintf("name table\n"))
		b.WriteString(fmt.Sprintf("%#v\n", f.name))
	case "glyf":
		if f.glyf == nil {
			b.WriteString("glyf: missing\n")
			break
		}
		rawTotal := 0.0
		for _, desc := range f.glyf.descs {
			rawTotal += float64(len(desc.raw))
		}
		b.WriteString(fmt.Sprintf("glyf table present: %d descriptions (%.2f kB)\n", len(f.glyf.descs), rawTotal/1024))
	case "post":
		if f.post == nil {
			b.WriteString("post: missing\n")
			break
		}
		b.WriteString(fmt.Sprintf("post table present: %d numGlyphs\n", f.post.numGlyphs))
		b.WriteString(fmt.Sprintf("- post glyphNameIndex: %d\n", len(f.post.glyphNameIndex)))
		b.WriteString(fmt.Sprintf("- post glyphNames: %d\n", len(f.post.glyphNames)))
		for i, gn := range f.post.glyphNames {
			if i > 10 {
				break
			}
			b.WriteString(fmt.Sprintf("- post: %d: %s\n", i+1, gn))
		}
		b.WriteString(fmt.Sprintf("%#v\n", f.post))
	default:
		b.WriteString(fmt.Sprintf("%s: unsupported table for info\n", table))
	}

	return b.String()
}

// String outputs some readable information about the font (table record stats).
func (f *font) String() string {
	return f.TableInfo("trec")
}
