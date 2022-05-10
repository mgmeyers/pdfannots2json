package pdfutils

import (
	"math"

	"github.com/golang/geo/r2"
	"github.com/mgmeyers/unipdf/v3/core"
	"github.com/mgmeyers/unipdf/v3/extractor"
	"github.com/mgmeyers/unipdf/v3/model"
)

func ApplyPageRotation(page *model.PdfPage, rect []float64) []float64 {
	if page.Rotate == nil {
		return rect
	}

	angle := *page.Rotate
	if angle == 0 {
		return rect
	}

	width := page.MediaBox.Width()
	height := page.MediaBox.Height()

	if angle == 90 {
		return []float64{rect[1], width - rect[2], rect[3], width - rect[0]}
	}

	if angle == 270 {
		return []float64{height - rect[3], rect[0], height - rect[1], rect[2]}
	}

	// 180
	return []float64{width - rect[2], height - rect[3], width - rect[0], height - rect[1]}
}

func IsWithinOverlapThresh(annot r2.Rect, mark r2.Rect) bool {
	markSize := getArea(mark)
	intersect := getArea(annot.Intersection(mark))

	return intersect/markSize >= 0.5
}

func getArea(r r2.Rect) float64 {
	s := r.Size()
	return s.X * s.Y
}

func GetMarkRect(mark extractor.TextMark) r2.Rect {
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

func GetAnnotationRects(page *model.PdfPage, annotation *model.PdfAnnotation) []r2.Rect {
	qp := GetQuadPoint(annotation)

	if qp == nil {
		return nil
	}

	coords, err := qp.GetAsFloat64Slice()
	if err != nil {
		return nil
	}

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

func GetQuadPoint(annotation *model.PdfAnnotation) *core.PdfObjectArray {
	ctx := annotation.GetContext()
	annotType := GetAnnotationType(ctx)

	switch annotType {
	case Highlight:
		if qp, ok := ctx.(*model.PdfAnnotationHighlight).QuadPoints.(*core.PdfObjectArray); ok {
			return qp
		}
		break
	case Strike:
		if qp, ok := ctx.(*model.PdfAnnotationStrikeOut).QuadPoints.(*core.PdfObjectArray); ok {
			return qp
		}
		break
	case Underline:
		if qp, ok := ctx.(*model.PdfAnnotationUnderline).QuadPoints.(*core.PdfObjectArray); ok {
			return qp
		}
		break
	}

	return nil
}

func GetCoordinates(annotation *model.PdfAnnotation) (float64, float64) {
	objArr := annotation.Rect.(*core.PdfObjectArray)
	annotRect, err := objArr.ToFloat64Array()
	if err != nil {
		return 0.0, 0.0
	}

	x := math.Round(math.Min(annotRect[0], annotRect[2])*100) / 100
	y := math.Round(math.Min(annotRect[1], annotRect[3])*100) / 100

	return x, y
}

func GetBoundsFromAnnotMarks(annotRect r2.Rect, markRects []r2.Rect) r2.Rect {
	bound := r2.EmptyRect()
	boundSet := false

	for _, mark := range markRects {
		if !mark.IsValid() || mark.IsEmpty() {
			continue
		}

		if annotRect.Intersects(mark) && IsWithinOverlapThresh(annotRect, mark) {
			if !boundSet {
				bound = mark
				boundSet = true
			} else {
				bound.X.Lo = math.Min(bound.X.Lo, mark.X.Lo)
				bound.Y.Lo = math.Min(bound.Y.Lo, mark.Y.Lo)
				bound.X.Hi = math.Max(bound.X.Hi, mark.X.Hi)
				bound.Y.Hi = math.Max(bound.Y.Hi, mark.Y.Hi)
			}
		}
	}

	return bound
}
