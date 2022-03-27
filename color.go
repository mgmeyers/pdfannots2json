package main

import (
	"fmt"

	"github.com/mgmeyers/unipdf/v3/core"
	"github.com/mgmeyers/unipdf/v3/model"
)

func toHEXStr(i int) string {
	s := fmt.Sprintf("%x", i)

	if len(s) == 1 {
		return "0" + s
	}

	return s
}

func pdfObjToHex(c core.PdfObject) string {
	if c == nil {
		return ""
	}

	clr, err := c.(*core.PdfObjectArray).ToFloat64Array()
	endIfErr(err)

	if len(clr) < 3 {
		return ""
	}

	return "#" + toHEXStr(int(clr[0]*255)) + toHEXStr(int(clr[1]*255)) + toHEXStr(int(clr[2]*255))
}

func getColor(annotation *model.PdfAnnotation) string {
	ctx := annotation.GetContext()
	annotType := getType(ctx)

	switch annotType {
	case Highlight:
		return pdfObjToHex(ctx.(*model.PdfAnnotationHighlight).C)
	case Strike:
		return pdfObjToHex(ctx.(*model.PdfAnnotationStrikeOut).C)
	case Underline:
		return pdfObjToHex(ctx.(*model.PdfAnnotationUnderline).C)
	case Rectangle:
		return pdfObjToHex(ctx.(*model.PdfAnnotationSquare).Border)
	case Text:
		return pdfObjToHex(ctx.(*model.PdfAnnotationText).C)
	}

	return ""
}
