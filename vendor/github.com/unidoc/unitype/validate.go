/*
 * This file is subject to the terms and conditions defined in
 * file 'LICENSE.md', which is part of this source code package.
 */

package unitype

import (
	"bytes"
	"errors"
	"io"

	"github.com/sirupsen/logrus"
)

// validate font data model `f` in `r`. Checks if required tables are present and whether
// table checksums are correct.
func (f *font) validate(r *byteReader) error {
	if f.trec == nil {
		logrus.Debug("Table records missing")
		return errRequiredField
	}
	if f.ot == nil {
		logrus.Debug("Offsets table missing")
		return errRequiredField
	}
	if f.head == nil {
		logrus.Debug("head table missing")
		return errRequiredField
	}

	// Validate the font.
	logrus.Debug("Validating entire font")
	{
		err := r.SeekTo(0)
		if err != nil {
			return err
		}

		var buf bytes.Buffer
		_, err = io.Copy(&buf, r.reader)
		if err != nil {
			return err
		}

		data := buf.Bytes()

		headRec, ok := f.trec.trMap["head"]
		if !ok {
			logrus.Debug("head not set")
			return errRequiredField
		}
		hoff := headRec.offset

		// set checksumAdjustment data to 0 in the head table.
		data[hoff+8] = 0
		data[hoff+9] = 0
		data[hoff+10] = 0
		data[hoff+11] = 0

		bw := newByteWriter(&bytes.Buffer{})
		bw.buffer.Write(data)

		checksum := bw.checksum()
		adjustment := 0xB1B0AFBA - checksum
		if f.head.checksumAdjustment != adjustment {
			return errors.New("file checksum mismatch")
		}
	}

	// Validate each table.
	logrus.Debug("Validating font tables")
	for _, tr := range f.trec.list {
		logrus.Debugf("Validating %s", tr.tableTag.String())
		logrus.Debugf("%+v", tr)

		bw := newByteWriter(&bytes.Buffer{})

		if tr.offset < 0 || tr.length < 0 {
			logrus.Debug("Range check error")
			return errRangeCheck
		}

		logrus.Debugf("Seeking to %d, to read %d bytes", tr.offset, tr.length)
		err := r.SeekTo(int64(tr.offset))
		if err != nil {
			return err
		}
		logrus.Debugf("Offset: %d", r.Offset())

		b := make([]byte, tr.length)
		_, err = io.ReadFull(r.reader, b)
		if err != nil {
			return err
		}
		logrus.Debugf("Read (%d)", len(b))
		// TODO(gunnsth): Validate head.
		if tr.tableTag.String() == "head" {
			// Set the checksumAdjustment to 0 so that head checksum is valid.
			if len(b) < 12 {
				return errors.New("head too short")
			}
			b[8], b[9], b[10], b[11] = 0, 0, 0, 0
		}

		_, err = bw.buffer.Write(b)
		if err != nil {
			return err
		}

		checksum := bw.checksum()
		if tr.checksum != checksum {
			logrus.Debugf("Invalid checksum (%d != %d)", checksum, tr.checksum)
			return errors.New("checksum incorrect")
		}

		if int(tr.length) != bw.bufferedLen() {
			logrus.Debug("Length mismatch")
			return errRangeCheck
		}
	}

	return nil
}
