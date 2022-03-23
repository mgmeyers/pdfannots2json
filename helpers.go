package main

import (
	"log"
	"os"
	"time"

	"github.com/mgmeyers/unipdf/v3/model"
)

func endIfErr(e error) {
	if e != nil {
		eLog := log.New(os.Stderr, "", 0)
		eLog.Fatalln(e)
	}
}

const dateFormat = "D:20060102150405+07'00'"

func getDate(annot *model.PdfAnnotation) time.Time {
	dateStr := annot.M
	date, err := time.Parse(dateFormat, dateStr.String())

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
		return Unsuported
	}
}
