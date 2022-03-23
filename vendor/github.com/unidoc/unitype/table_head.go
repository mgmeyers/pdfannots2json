/*
 * This file is subject to the terms and conditions defined in
 * file 'LICENSE.md', which is part of this source code package.
 */

package unitype

import (
	"errors"

	"github.com/sirupsen/logrus"
)

// Font header.
// https://docs.microsoft.com/en-us/typography/opentype/spec/head
type headTable struct {
	majorVersion       uint16 // 00 01
	minorVersion       uint16 // 00 00
	fontRevision       fixed  // 00 01 CA 3D
	checksumAdjustment uint32 // 00 00 00 00
	magicNumber        uint32 // 5F 0F 3C F5
	flags              uint16
	unitsPerEm         uint16
	created            longdatetime
	modified           longdatetime
	xMin               int16
	yMin               int16
	xMax               int16
	yMax               int16
	macStyle           uint16
	lowestRecPPEM      uint16
	fontDirectionHint  int16
	indexToLocFormat   int16
	glyphDataFormat    int16
}

// parse the font's *head* table from `r` in the context of `f`.
// TODO(gunnsth): Read the table as bytes first and then process? Probably easier in terms of checksumming etc.
func (f *font) parseHead(r *byteReader) (*headTable, error) {
	_, has, err := f.seekToTable(r, "head")
	if err != nil {
		return nil, err
	}
	if !has {
		// Does not have head.
		return nil, nil
	}

	t := &headTable{}
	err = r.read(&t.majorVersion, &t.minorVersion, &t.fontRevision)
	if err != nil {
		return nil, err
	}

	err = r.read(&t.checksumAdjustment, &t.magicNumber)
	if err != nil {
		return nil, err
	}
	if t.magicNumber != 0x5F0F3CF5 {
		logrus.Debugf("Error: got magic number 0x%X", t.magicNumber)
		return nil, errors.New("magic number mismatch")
	}

	err = r.read(&t.flags, &t.unitsPerEm, &t.created, &t.modified)
	if err != nil {
		return nil, err
	}

	err = r.read(&t.xMin, &t.yMin, &t.xMax, &t.yMax)
	if err != nil {
		return nil, err
	}

	return t, r.read(&t.macStyle, &t.lowestRecPPEM, &t.fontDirectionHint, &t.indexToLocFormat, &t.glyphDataFormat)
}

func (f *font) writeHead(w *byteWriter) error {
	if f.head == nil {
		return errRequiredField
	}
	t := f.head
	err := w.write(t.majorVersion, t.minorVersion, t.fontRevision, t.checksumAdjustment, t.magicNumber)
	if err != nil {
		return err
	}

	err = w.write(t.flags, t.unitsPerEm, t.created, t.modified, t.xMin, t.yMin, t.xMax, t.yMax)
	if err != nil {
		return err
	}

	return w.write(t.macStyle, t.lowestRecPPEM, t.fontDirectionHint, t.indexToLocFormat, t.glyphDataFormat)
}
