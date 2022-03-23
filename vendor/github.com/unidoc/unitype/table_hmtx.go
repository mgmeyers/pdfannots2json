/*
 * This file is subject to the terms and conditions defined in
 * file 'LICENSE.md', which is part of this source code package.
 */

package unitype

import (
	"github.com/sirupsen/logrus"
)

type hmtxTable struct {
	hMetrics         []longHorMetric // length is numberOfHMetrics from hhea table.
	leftSideBearings []int16         // length is (numGlyphs - numberOfHmetrics) from maxp and hhea tables.
}

type longHorMetric struct {
	advanceWidth uint16
	lsb          int16
}

func (f *font) parseHmtx(r *byteReader) (*hmtxTable, error) {
	if f.maxp == nil || f.hhea == nil {
		logrus.Debug("maxp or hhea table missing")
		return nil, errRequiredField
	}

	_, has, err := f.seekToTable(r, "hmtx")
	if err != nil {
		return nil, err
	}
	if !has {
		logrus.Debug("hmtx table absent")
		return nil, nil
	}

	t := &hmtxTable{}

	numberOfHMetrics := int(f.hhea.numberOfHMetrics)
	for i := 0; i < numberOfHMetrics; i++ {
		var lhm longHorMetric
		err := r.read(&lhm.advanceWidth, &lhm.lsb)
		if err != nil {
			return nil, err
		}

		t.hMetrics = append(t.hMetrics, lhm)
	}

	lsbLen := int(f.maxp.numGlyphs) - numberOfHMetrics
	if lsbLen > 0 {
		err = r.readSlice(&t.leftSideBearings, lsbLen)
		if err != nil {
			return nil, err
		}
	}

	return t, nil
}

// optimizeHmtx optimizes the htmx table.
func (f *font) optimizeHmtx() {
	i := len(f.hmtx.hMetrics) - 1
	if i <= 0 {
		return
	}
	lastWidth := f.hmtx.hMetrics[i].advanceWidth
	j := i - 1
	for j >= 0 && f.hmtx.hMetrics[j].advanceWidth == lastWidth {
		j--
	}
	numStrip := i - j - 1
	if numStrip == 0 {
		return
	}

	f.hhea.numberOfHMetrics = uint16(j + 2)
	var lsbPrepend []int16
	for k := j + 2; k <= i; k++ {
		lsbPrepend = append(lsbPrepend, f.hmtx.hMetrics[k].lsb)
	}
	f.hmtx.leftSideBearings = append(lsbPrepend, f.hmtx.leftSideBearings...)
	f.hmtx.hMetrics = f.hmtx.hMetrics[0 : j+2]
}

// writeHmtx writes the font's hmtx table  to `w`.
func (f *font) writeHmtx(w *byteWriter) error {
	if f.hmtx == nil || f.hhea == nil {
		return nil
	}

	for _, lhm := range f.hmtx.hMetrics {
		err := w.write(lhm.advanceWidth, lhm.lsb)
		if err != nil {
			return err
		}
	}

	return w.writeSlice(f.hmtx.leftSideBearings)
}
