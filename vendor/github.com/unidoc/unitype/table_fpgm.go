/*
 * This file is subject to the terms and conditions defined in
 * file 'LICENSE.md', which is part of this source code package.
 */

package unitype

import (
	"github.com/sirupsen/logrus"
)

// fpgmTable represents font program instructions and is needed by fonts that are instructed.
type fpgmTable struct {
	instructions []uint8
}

func (f *font) parseFpgm(r *byteReader) (*fpgmTable, error) {
	tr, has, err := f.seekToTable(r, "fpgm")
	if err != nil {
		return nil, err
	}
	if !has || tr == nil {
		logrus.Debug("fpgm table absent")
		return nil, nil
	}

	t := &fpgmTable{}
	numVals := int(tr.length)
	err = r.readSlice(&t.instructions, numVals)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (f *font) writeFpgm(w *byteWriter) error {
	if f.fpgm == nil {
		return nil
	}

	return w.writeSlice(f.fpgm.instructions)
}
