package pdfutils

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/golang/geo/r2"
	"github.com/mgmeyers/go-fitz"
	"github.com/mgmeyers/unipdf/v3/extractor"
	"github.com/mgmeyers/unipdf/v3/model"
)

const dateFormat = "D:20060102150405+07'00'"
const dateFormatZ = "D:20060102150405Z07'00'"
const dateFormatNoZ = "D:20060102150405"

func GetAnnotationDate(annot *model.PdfAnnotation) *time.Time {
	dateStr := annot.M

	if dateStr == nil {
		return nil
	}

	date, err := time.Parse(dateFormat, dateStr.String())

	if err != nil {
		date, err = time.Parse(dateFormatZ, dateStr.String())
	}

	if err != nil {
		split := strings.Split(dateStr.String(), "Z")
		date, err = time.Parse(dateFormatNoZ, split[0])
	}

	if err != nil {
		return nil
	}

	return &date
}

func GetAnnotationType(t interface{}) string {
	switch t.(type) {
	case *model.PdfAnnotationHighlight:
		return Highlight
	case *model.PdfAnnotationStrikeOut:
		return Strike
	case *model.PdfAnnotationUnderline:
		return Underline
	case *model.PdfAnnotationSquare:
		return Rectangle
	case *model.PdfAnnotationText:
		return Text
	default:
		return Unsupported
	}
}

func RemoveNul(str string) string {
	return strings.Map(func(r rune) rune {
		if r == unicode.ReplacementChar {
			return -1
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, str)
}

func GetTextByAnnotBounds(fitzDoc *fitz.Document, pageIndex int, page *model.PdfPage, bounds r2.Rect) (string, error) {
	height := page.MediaBox.Height()

	if page.Rotate != nil && (*page.Rotate == 90 || *page.Rotate == 270) {
		height = page.MediaBox.Width()
	}

	rotated := ApplyPageRotation(page, []float64{bounds.X.Lo, bounds.Y.Lo, bounds.X.Hi, bounds.Y.Hi})

	x1 := rotated[0]
	y1 := rotated[1]
	x2 := rotated[2]
	y2 := rotated[3]

	if page.Rotate != nil && (*page.Rotate == 0 || *page.Rotate == 180) {
		bHeight := rotated[3] - rotated[1]
		yDiff := (bHeight * 0.6) / 2
		y1 += yDiff
		y2 -= yDiff
	} else {
		bWidth := rotated[2] - rotated[0]
		xDiff := (bWidth * 0.6) / 2
		x1 += xDiff
		x2 -= xDiff
	}

	// fitz's y-axis is oriented at the top
	y1 = height - y1
	y2 = height - y2

	return fitzDoc.TextByBounds(
		pageIndex,
		72.0,
		float32(math.Min(x1, x2)),
		float32(math.Min(y1, y2)),
		float32(math.Max(x1, x2)),
		float32(math.Max(y1, y2)),
	)
}

func GetFallbackText(text string, annotRect r2.Rect, markRects []r2.Rect, marks []extractor.TextMark) string {
	segment := ""

	for i, mark := range markRects {
		if !mark.IsValid() || mark.IsEmpty() {
			continue
		}

		if annotRect.Intersects(mark) && IsWithinOverlapThresh(annotRect, mark) {
			if len(marks[i].Text) > 0 && marks[i].Offset > 0 && len(segment) > 0 {
				prevChar := string(text[marks[i].Offset-1])

				if prevChar == " " || prevChar == "\n" {
					segment += " " + marks[i].Text
					continue
				}

			}

			segment += marks[i].Text
			continue
		}
	}

	return segment
}

func ShouldUseFallback(str string) bool {
	length := len(str)
	missingChars := strings.Count(str, "�")

	if missingChars == 0 {
		return false
	}

	ratio := float64(missingChars) / float64(length)

	return ratio > 0.2
}

func GetAnnotationID(ids map[string]bool, pageIndex int, x float64, y float64, annotType string) string {
	xInt := int(x)
	yInt := int(y)
	id := fmt.Sprintf("%s-p%dx%dy%d", annotType, pageIndex+1, xInt, yInt)
	_, ok := ids[id]

	for i := 1; ok; i++ {
		id = fmt.Sprintf("%s-p%dx%dy%d-%d", annotType, pageIndex+1, xInt, yInt, i)
		_, ok = ids[id]
	}

	ids[id] = true

	return id
}

var nlAndSpace = regexp.MustCompile(`[\n\s]+`)

func CondenseSpaces(str string) string {
	return nlAndSpace.ReplaceAllString(str, " ")
}
