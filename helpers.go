package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/gen2brain/go-fitz"
	"github.com/golang/geo/r2"
	"github.com/mgmeyers/unipdf/v3/model"
)

func logOutput(annots []*Annotation) {
	jsonAnnots, err := json.Marshal(annots)

	endIfErr(err)

	oLog := log.New(os.Stdout, "", 0)
	oLog.Println(string(jsonAnnots))
}

func endIfErr(e error) {
	if e != nil {
		eLog := log.New(os.Stderr, "", 0)
		eLog.Fatalln(e)
	}
}

const dateFormat = "D:20060102150405+07'00'"
const dateFormatZ = "D:20060102150405Z07'00'"
const dateFormatNoZ = "D:20060102150405"

func getDate(annot *model.PdfAnnotation) *time.Time {
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

	endIfErr(err)

	return &date
}

func getType(t interface{}) string {
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

func removeNul(str string) string {
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

func getTextByAnnotBounds(fitzDoc *fitz.Document, pageIndex int, page *model.PdfPage, bounds r2.Rect) string {
	height := page.MediaBox.Height()

	if *page.Rotate == 90 || *page.Rotate == 270 {
		height = page.MediaBox.Width()
	}

	rotated := applyPageRotation(page, []float64{bounds.X.Lo, bounds.Y.Lo, bounds.X.Hi, bounds.Y.Hi})

	x1 := rotated[0]
	y1 := rotated[1]
	x2 := rotated[2]
	y2 := rotated[3]

	if *page.Rotate == 0 || *page.Rotate == 180 {
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

	annotText, err := fitzDoc.TextByBounds(
		pageIndex,
		72.0,
		float32(math.Min(x1, x2)),
		float32(math.Min(y1, y2)),
		float32(math.Max(x1, x2)),
		float32(math.Max(y1, y2)),
	)
	endIfErr(err)

	return annotText
}

func getID(ids map[string]bool, pageIndex int, x float64, y float64, annotType string) string {
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

func condenseSpaces(str string) string {
	return nlAndSpace.ReplaceAllString(str, " ")
}
