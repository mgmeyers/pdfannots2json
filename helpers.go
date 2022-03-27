package main

import (
	"log"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/mgmeyers/unipdf/v3/model"
)

func endIfErr(e error) {
	if e != nil {
		eLog := log.New(os.Stderr, "", 0)
		eLog.Fatalln(e)
	}
}

const dateFormat = "D:20060102150405+07'00'"
const dateFormatZ = "D:20060102150405Z07'00'"
const dateFormatNoZ = "D:20060102150405"

func getDate(annot *model.PdfAnnotation) time.Time {
	dateStr := annot.M
	date, err := time.Parse(dateFormat, dateStr.String())

	if err != nil {
		date, err = time.Parse(dateFormatZ, dateStr.String())
	}

	if err != nil {
		split := strings.Split(dateStr.String(), "Z")
		date, err = time.Parse(dateFormatNoZ, split[0])
	}

	endIfErr(err)

	return date
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
