/*
 * This file is subject to the terms and conditions defined in
 * file 'LICENSE.md', which is part of this source code package.
 */

package unitype

import (
	"bytes"
	"strconv"
	"unicode"
	"unicode/utf8"

	"github.com/sirupsen/logrus"

	"golang.org/x/text/encoding/charmap"

	"github.com/unidoc/unitype/internal/strutils"
)

// nameTable represents the Naming table (name).
// The naming table allows multilingual strings to be associated with the font.
// These strings can represent copyright notices, font names, family names, style names, and so on.
type nameTable struct {
	// format >= 0
	format       uint16
	count        uint16
	stringOffset offset16
	nameRecords  []*nameRecord // len = count.

	// format = 1 adds
	langTagCount   uint16
	langTagRecords []*langTagRecord // len = langTagCount
}

type langTagRecord struct {
	length uint16
	offset offset16
	data   []byte // actual string data (UTF-16BE format).
}

// Each string in the string storage is referenced by a name record.
type nameRecord struct {
	platformID uint16
	encodingID uint16
	languageID uint16
	nameID     uint16
	length     uint16
	offset     offset16
	data       []byte // actual string data.
}

// GetNameByID returns the first entry according to the name table with `nameID`.
// An empty string is returned otherwise (nothing found).
func (f *font) GetNameByID(nameID int) string {
	if f == nil || f.name == nil {
		logrus.Debug("ERROR: Font or name not set")
		return ""
	}
	for _, nr := range f.name.nameRecords {
		if int(nr.nameID) == nameID {
			return nr.Decoded()
		}
	}
	return ""
}

// numPrintables returns the number of printable runes in `str`
func numPrintables(str string) int {
	printables := 0
	for _, r := range str {
		if unicode.IsPrint(r) || r == '\n' {
			printables++
		}
	}
	return printables
}

// makePrintable replaces unprintable runes with quotes runes, returning printable string.
func makePrintable(str string) string {
	var buf bytes.Buffer
	for _, r := range str {
		if unicode.IsPrint(r) || r == '\n' {
			buf.WriteRune(r)
		} else {
			buf.WriteString(strconv.QuoteRune(r))
		}
	}
	return buf.String()
}

// Decoded attempts to decode the underlying data and convert to a string.
// NOTE: Works in many cases but often has some -garbage- around texts.
func (nr nameRecord) Decoded() string {
	switch nr.platformID {
	case 0: // unicode
		// TODO(gunnsth): Untested as have not encountered this yet.
		dup := make([]byte, len(nr.data))
		copy(dup, nr.data)
		var decoded bytes.Buffer

		for len(dup) > 0 {
			r, size := utf8.DecodeRune(dup)
			dup = dup[size:]
			decoded.WriteRune(r)
		}

		return makePrintable(decoded.String())
	case 1: // macintosh
		var decoded bytes.Buffer
		for _, val := range nr.data {
			decoded.WriteRune(charmap.Macintosh.DecodeByte(val))
		}
		macs := decoded.String()

		// Following may be needed in rare cases:
		/*
			utf16s := strutils.UTF16ToString([]byte(macs))
			if numPrintables(utf16s) > numPrintables(macs) {
				return makePrintable(utf16s)
			}
		*/
		return makePrintable(macs)

	case 3: // windows
		// When building a Unicode font for Windows, the platform ID should be 3 and the encoding ID should be 1,
		// and the referenced string data must be encoded in UTF-16BE. When building a symbol font for Windows,
		// the platform ID should be 3 and the encoding ID should be 0, and the referenced string data must be
		// encoded in UTF-16BE. (https://docs.microsoft.com/en-us/typography/opentype/spec/name).
		if nr.encodingID == 0 || nr.encodingID == 1 {
			if len(nr.data) > 0 {
				decoded := strutils.UTF16ToString(nr.data)
				return makePrintable(decoded)
			}
		}
	}

	return makePrintable(string(nr.data))
}

func (f *font) parseNameTable(r *byteReader) (*nameTable, error) {
	tr, has, err := f.seekToTable(r, "name")
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, nil
	}
	logrus.Debugf("TR: %+v", tr)

	t := &nameTable{}
	err = r.read(&t.format, &t.count, &t.stringOffset)
	if err != nil {
		return nil, err
	}
	logrus.Debugf("format/count/stringOffset: %v/%v/%v", t.format, t.count, t.stringOffset)
	logrus.Debugf("-- name string offset: %d", t.stringOffset)

	if t.format > 1 {
		logrus.Debugf("ERROR: format > 1 (%d)", t.format)
		return nil, errRangeCheck
	}

	for i := 0; i < int(t.count); i++ {
		var nr nameRecord
		err = r.read(&nr.platformID, &nr.encodingID, &nr.languageID, &nr.nameID, &nr.length, &nr.offset)
		if err != nil {
			return nil, err
		}
		logrus.Debugf("name record %d: %v/%v/%v/%v/%v/%v", i, nr.platformID, nr.encodingID, nr.languageID, nr.nameID,
			nr.length, nr.offset)
		t.nameRecords = append(t.nameRecords, &nr)
	}

	if t.format == 1 {
		err = r.read(&t.langTagCount)
		if err != nil {
			return nil, err
		}
		for i := 0; i < int(t.langTagCount); i++ {
			var ltr langTagRecord
			err = r.read(&ltr.length, &ltr.offset)
			if err != nil {
				return nil, err
			}
			logrus.Debugf("ltr name record %d: %v/%v", i, ltr.offset, ltr.length)
			t.langTagRecords = append(t.langTagRecords, &ltr)
		}
	}

	// Get the actual string data.
	for _, nr := range t.nameRecords {
		if int(t.stringOffset)+int(nr.offset)+int(nr.length) > int(tr.length) {
			logrus.Debugf("%v> %v", int(t.stringOffset)+int(nr.offset)+int(nr.length), int(tr.length))
			logrus.Debug("name string offset outside table")
			return nil, errRangeCheck
		}

		err = r.SeekTo(int64(t.stringOffset) + int64(tr.offset) + int64(nr.offset))
		if err != nil {
			logrus.Debugf("Error: %v", err)
			return nil, err
		}

		err = r.readBytes(&nr.data, int(nr.length))
		if err != nil {
			logrus.Debugf("Error: %v", err)
			return nil, err
		}
	}

	for _, ltr := range t.langTagRecords {
		if int(t.stringOffset)+int(ltr.offset)+int(ltr.length) > int(tr.length) {
			logrus.Debug("lang tag string offset outside table")
			return nil, errRangeCheck
		}

		err = r.SeekTo(int64(t.stringOffset) + int64(tr.offset) + int64(ltr.offset))
		if err != nil {
			logrus.Debugf("Error: %v", err)
			return nil, err
		}
		err = r.readBytes(&ltr.data, int(ltr.length))
		if err != nil {
			logrus.Debugf("Error: %v", err)
			return nil, err
		}
	}

	logrus.Debugf("Name records: %d", len(t.nameRecords))
	for _, nr := range t.nameRecords {
		logrus.Debugf("%d %d %d - '%s' (%d)", nr.platformID, nr.encodingID, nr.nameID, nr.Decoded(), len(nr.data))
	}

	return t, nil
}

func (f *font) writeNameTable(w *byteWriter) error {
	if f.name == nil {
		logrus.Debug("name is nil")
		return nil
	}
	t := f.name

	// Preprocess: Write to buffer and update offsets.
	var buf bytes.Buffer
	{
		bufw := newByteWriter(&buf)
		for _, nr := range t.nameRecords {
			nr.offset = offset16(bufw.bufferedLen())
			nr.length = uint16(len(nr.data))
			err := bufw.writeSlice(nr.data)
			if err != nil {
				return err
			}
		}
		for _, ltr := range t.langTagRecords {
			ltr.offset = offset16(bufw.bufferedLen())
			ltr.length = uint16(len(ltr.data))
			err := bufw.writeSlice(ltr.data)
			if err != nil {
				return err
			}
		}
		err := bufw.flush()
		if err != nil {
			return err
		}
	}
	logrus.Debugf("Buffer length: %d", buf.Len())

	// Update count and stringOffsets (calculated).
	t.count = uint16(len(t.nameRecords))
	t.langTagCount = uint16(len(t.langTagRecords))

	// 2+2+2+count*(6*2) + (format=1) 2+langTagCount*2
	t.stringOffset = 6 + offset16(t.count)*12
	if t.format == 1 {
		t.stringOffset += 2 + offset16(t.langTagCount)*4
	}

	logrus.Debugf("w @ %d", w.bufferedLen())
	err := w.write(t.format, t.count, t.stringOffset)
	if err != nil {
		return err
	}

	for _, nr := range t.nameRecords {
		err = w.write(nr.platformID, nr.encodingID, nr.languageID, nr.nameID, nr.length, nr.offset)
		if err != nil {
			return err
		}
	}
	logrus.Debugf("w @ %d", w.bufferedLen())

	if t.format == 1 {
		err = w.write(t.langTagCount)
		if err != nil {
			return err
		}
		for _, ltr := range t.langTagRecords {
			err = w.write(ltr.length, ltr.offset)
			if err != nil {
				return err
			}
		}
	}

	logrus.Debugf("w @ %d", w.bufferedLen())
	// Write the buffered data.
	err = w.writeBytes(buf.Bytes())
	if err != nil {
		return err
	}
	logrus.Debugf("w @ %d", w.bufferedLen())

	return nil
}
