/*
 * This file is subject to the terms and conditions defined in
 * file 'LICENSE.md', which is part of this source code package.
 */

package unitype

import (
	"github.com/sirupsen/logrus"
)

// cvtTable represents the Control Value Table (cvt).
// This table contains a list of values that can be referenced by instructions.
// TODO: For subsetting/optimization it would be good to know what glyphs need each value, so non-used values can be removed.
//       Probably part of optimization in the locations where these values are referenced.
type cvtTable struct {
	controlValues []int16 //fword
}

func (f *font) parseCvt(r *byteReader) (*cvtTable, error) {
	tr, has, err := f.seekToTable(r, "cvt")
	if err != nil {
		return nil, err
	}
	if !has || tr == nil {
		logrus.Debug("cvt table absent")
		return nil, nil
	}

	t := &cvtTable{}
	numVals := int(tr.length / 2)
	err = r.readSlice(&t.controlValues, numVals)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (f *font) writeCvt(w *byteWriter) error {
	if f.cvt == nil {
		return nil
	}

	return w.writeSlice(f.cvt.controlValues)
}
