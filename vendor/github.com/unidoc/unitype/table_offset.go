/*
 * This file is subject to the terms and conditions defined in
 * file 'LICENSE.md', which is part of this source code package.
 */

package unitype

type offsetTable struct {
	sfntVersion   uint32
	numTables     uint16
	searchRange   uint16
	entrySelector uint16
	rangeShift    uint16
}

// Size returns size of `t` in bytes.
func (t *offsetTable) Size() int64 {
	return 4 + 4*2 // 4+8=12
}

func (f *font) parseOffsetTable(r *byteReader) (*offsetTable, error) {
	ot := &offsetTable{}

	err := r.read(&ot.sfntVersion, &ot.numTables, &ot.searchRange)
	if err != nil {
		return nil, err
	}

	err = r.read(&ot.entrySelector, &ot.rangeShift)
	if err != nil {
		return nil, err
	}

	return ot, nil
}

func (f *font) writeOffsetTable(w *byteWriter) error {
	if f.ot == nil {
		return errRequiredField
	}
	return w.write(f.ot.sfntVersion, f.ot.numTables, f.ot.searchRange, f.ot.entrySelector, f.ot.rangeShift)
}
