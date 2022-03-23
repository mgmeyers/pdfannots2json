/*
 * This file is subject to the terms and conditions defined in
 * file 'LICENSE.md', which is part of this source code package.
 */

package unitype

import (
	"github.com/sirupsen/logrus"
)

// prepTable represents a Control Value Program table (prep).
// Consists of a set of TrueType instructions that will be executed whenever the font or point size
// or transformation matrix change and before each glyph is interpreted.
// Used for preparation (hence the name "prep").
type prepTable struct {
	// number of instructions - the number of uint8 that fit the size of the table.
	instructions []uint8
}

func (f *font) parsePrep(r *byteReader) (*prepTable, error) {
	tr, has, err := f.seekToTable(r, "prep")
	if err != nil {
		return nil, err
	}
	if !has || tr == nil {
		logrus.Debug("prep table absent")
		return nil, nil
	}

	t := &prepTable{}
	numInstructions := int(tr.length)
	err = r.readSlice(&t.instructions, numInstructions)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (f *font) writePrep(w *byteWriter) error {
	if f.prep == nil {
		return nil
	}

	return w.writeSlice(f.prep.instructions)
}
