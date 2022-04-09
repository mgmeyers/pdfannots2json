package main

import (
	"fmt"

	colorful "github.com/lucasb-eyer/go-colorful"
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
	if annotation == nil {
		return ""
	}

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
		return pdfObjToHex(ctx.(*model.PdfAnnotationSquare).C)
	case Text:
		return pdfObjToHex(ctx.(*model.PdfAnnotationText).C)
	}

	return ""
}

func getColorCategory(annotation *model.PdfAnnotation) string {
	if annotation == nil {
		return ""
	}

	ctx := annotation.GetContext()
	annotType := getType(ctx)

	switch annotType {
	case Highlight:
		return pdfObjToColorCategory(ctx.(*model.PdfAnnotationHighlight).C)
	case Strike:
		return pdfObjToColorCategory(ctx.(*model.PdfAnnotationStrikeOut).C)
	case Underline:
		return pdfObjToColorCategory(ctx.(*model.PdfAnnotationUnderline).C)
	case Rectangle:
		return pdfObjToColorCategory(ctx.(*model.PdfAnnotationSquare).C)
	case Text:
		return pdfObjToColorCategory(ctx.(*model.PdfAnnotationText).C)
	}

	return ""
}

func pdfObjToColorCategory(c core.PdfObject) string {
	if c == nil {
		return ""
	}

	clr, err := c.(*core.PdfObjectArray).ToFloat64Array()
	endIfErr(err)

	if len(clr) < 3 {
		return ""
	}

	color := colorful.Color{
		R: clr[0],
		G: clr[1],
		B: clr[2],
	}
	h, s, l := color.Hsl()

	// define color category based on HSL
	if l < 0.12 {
		return "Black"
	}
	if l > 0.98 {
		return "White"
	}
	if s < 0.2 {
		return "Gray"
	}
	if h < 15 {
		return "Red"
	}
	if h < 45 {
		return "Orange"
	}
	if h < 65 {
		return "Yellow"
	}
	if h < 170 {
		return "Green"
	}
	if h < 190 {
		return "Cyan"
	}
	if h < 263 {
		return "Blue"
	}
	if h < 280 {
		return "Purple"
	}
	if h < 335 {
		return "Magenta"
	}
	return "Red"
}
