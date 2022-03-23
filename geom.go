package main

import (
	"github.com/golang/geo/r2"
	"github.com/mgmeyers/unipdf/v3/core"
	"github.com/mgmeyers/unipdf/v3/extractor"
	"github.com/mgmeyers/unipdf/v3/model"
)

func isWithinOverlapThresh(annot r2.Rect, mark r2.Rect) bool {
	markSize := getArea(mark)
	intersect := getArea(annot.Intersection(mark))

	return intersect/markSize >= 0.5
}

func getArea(r r2.Rect) float64 {
	s := r.Size()
	return s.X * s.Y
}

func getMarkRect(mark extractor.TextMark) r2.Rect {
	return r2.RectFromPoints(
		r2.Point{
			X: mark.BBox.Llx,
			Y: mark.BBox.Lly,
		},
		r2.Point{
			X: mark.BBox.Llx,
			Y: mark.BBox.Ury,
		},
		r2.Point{
			X: mark.BBox.Urx,
			Y: mark.BBox.Lly,
		},
		r2.Point{
			X: mark.BBox.Urx,
			Y: mark.BBox.Ury,
		},
	)
}

func getAnnotationRects(annotation *model.PdfAnnotation) []r2.Rect {
	qp := getQuadPoint(annotation)

	if qp == nil {
		return nil
	}

	coords, err := qp.GetAsFloat64Slice()
	endIfErr(err)

	coordHolder := []float64{}
	ptHolder := []r2.Point{}
	rects := []r2.Rect{}

	for _, coord := range coords {
		coordHolder = append(coordHolder, coord)

		if len(coordHolder) == 2 {
			pt := r2.Point{X: coordHolder[0], Y: coordHolder[1]}
			ptHolder = append(ptHolder, pt)

			coordHolder = []float64{}

			if len(ptHolder) == 4 {
				r := r2.RectFromPoints(ptHolder[0], ptHolder[1], ptHolder[2], ptHolder[3])
				rects = append(rects, r)
				ptHolder = []r2.Point{}
			}
		}
	}

	return rects
}

func getQuadPoint(annotation *model.PdfAnnotation) *core.PdfObjectArray {
	ctx := annotation.GetContext()
	annotType := getType(ctx)

	switch annotType {
	case Highlight:
		return ctx.(*model.PdfAnnotationHighlight).QuadPoints.(*core.PdfObjectArray)
	case Strike:
		return ctx.(*model.PdfAnnotationStrikeOut).QuadPoints.(*core.PdfObjectArray)
	case Underline:
		return ctx.(*model.PdfAnnotationUnderline).QuadPoints.(*core.PdfObjectArray)
	}

	return nil
}
