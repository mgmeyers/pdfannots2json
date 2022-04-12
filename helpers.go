package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
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
	bHeight := bounds.Y.Hi - bounds.Y.Lo
	diff := (bHeight * 0.6) / 2

	x1 := bounds.X.Lo
	y1 := (page.MediaBox.Height() - (bounds.Y.Lo + diff))
	x2 := bounds.X.Hi
	y2 := (page.MediaBox.Height() - (bounds.Y.Hi - diff))

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
