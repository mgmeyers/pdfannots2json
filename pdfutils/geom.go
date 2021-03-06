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

func IsWithinOverlapThresh(annot r2.Rect, mark r2.Rect, thresh float64) bool {
	markSize := getArea(mark)
	intersect := getArea(annot.Intersection(mark))

	return intersect/markSize >= thresh
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

func distanceBetween(x1, y1, x2, y2 float64) float64 {
	return math.Sqrt(math.Pow(x1-x2, 2.0) + math.Pow(y1-y2, 2.0))
}

func GetClosestMark(x float64, y float64, markRects []r2.Rect) int {
	min := 999999999.0
	closest := -1

	for i, mark := range markRects {
		dist := distanceBetween(x, y, mark.X.Lo, mark.Y.Lo)

		if dist < min {
			min = dist
			closest = i
		}
	}

	return closest
}

func scaleY(rect r2.Rect, by float64) r2.Rect {
	clone := r2.EmptyRect()

	clone.X.Hi = rect.X.Hi
	clone.X.Lo = rect.X.Lo
	clone.Y.Hi = rect.Y.Hi
	clone.Y.Lo = rect.Y.Lo

	height := clone.Y.Hi - clone.Y.Lo
	yDiff := (height * by) / 2
	clone.Y.Hi -= yDiff
	clone.Y.Lo += yDiff

	return clone
}

func GetBoundsFromAnnotMarks(annotRect r2.Rect, markRects []r2.Rect) (r2.Rect, int) {
	bound := r2.EmptyRect()
	boundSet := false
	offset := -1

	for i, mark := range markRects {
		if !mark.IsValid() || mark.IsEmpty() {
			continue
		}

		scaled := scaleY(annotRect, 0.6)

		if scaled.Intersects(mark) {
			if offset == -1 {
				offset = i
			}
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

	return bound, offset
}
