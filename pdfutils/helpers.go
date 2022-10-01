package pdfutils

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/golang/geo/r2"
	"github.com/mgmeyers/go-fitz"
	"github.com/mgmeyers/unipdf/v3/core"
	"github.com/mgmeyers/unipdf/v3/extractor"
	"github.com/mgmeyers/unipdf/v3/model"
)

func max6(i int) string {
	str := strconv.Itoa(i)

	if len(str) > 6 {
		return str[0:6]
	}

	return str
}

func GetAnnotationSortKey(page int, offset int, top int) string {
	return fmt.Sprintf("%06s|%06s|%06s", max6(page), max6(offset), max6(top))
}

const dateFormat = "D:20060102150405+0700"
const dateFormatZ = "D:20060102150405Z0700"
const dateFormatNoZ = "D:20060102150405"

func GetAnnotationDate(annot *model.PdfAnnotation) *time.Time {
	dateObj := annot.M

	if dateObj == nil {
		return nil
	}

	dateStr := strings.ReplaceAll(dateObj.String(), "'", "")
	date, err := time.Parse(dateFormat, dateStr)

	if err != nil {
		date, err = time.Parse(dateFormatZ, dateStr)
	}

	if err != nil {
		split := strings.Split(dateStr, "Z")
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

var ff = regexp.MustCompile(`\x{FB00} *`)
var fi = regexp.MustCompile(`\x{FB01} *`)
var fl = regexp.MustCompile(`\x{FB02} *`)
var ffi = regexp.MustCompile(`\x{FB03} *`)
var ffl = regexp.MustCompile(`\x{FB04} *`)
var ft = regexp.MustCompile(`\x{FB05} *`)
var st = regexp.MustCompile(`\x{FB06} *`)

func ExpandLigatures(str string) string {
	str = ff.ReplaceAllString(str, "ff")
	str = fi.ReplaceAllString(str, "fi")
	str = fl.ReplaceAllString(str, "fl")
	str = ffi.ReplaceAllString(str, "ffi")
	str = ffl.ReplaceAllString(str, "ffl")
	str = ft.ReplaceAllString(str, "ft")
	str = st.ReplaceAllString(str, "st")

	return str
}

var hyphen = regexp.MustCompile(`([a-zA-Z])- +([a-zA-Z])`)

func DeHyphen(str string) string {
	return hyphen.ReplaceAllString(str, "$1$2")
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

	xAdjust := page.MediaBox.Llx - page.CropBox.Llx
	yAdjust := page.MediaBox.Ury - page.CropBox.Ury

	if *page.Rotate == 90 {
		height = page.MediaBox.Width()

		xAdjust = page.MediaBox.Lly - page.CropBox.Lly
		yAdjust = page.MediaBox.Urx - page.CropBox.Urx
	}

	if *page.Rotate == 180 {
		xAdjust = page.CropBox.Urx - page.MediaBox.Urx
		yAdjust = page.CropBox.Lly - page.MediaBox.Lly
	}

	if *page.Rotate == 270 {
		height = page.MediaBox.Width()

		xAdjust = page.CropBox.Ury - page.MediaBox.Ury
		yAdjust = page.CropBox.Llx - page.MediaBox.Llx
	}

	scaled := scaleY(bounds, 0.6)
	rotated := ApplyPageRotation(page, []float64{scaled.X.Lo, scaled.Y.Lo, scaled.X.Hi, scaled.Y.Hi})

	x1 := rotated[0] + xAdjust
	x2 := rotated[2] + xAdjust

	// fitz's y-axis is oriented at the top
	y1 := height - (rotated[1] + yAdjust)
	y2 := height - (rotated[3] + yAdjust)

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

		scaled := scaleY(annotRect, 0.6)

		if scaled.Intersects(mark) {
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

func ShouldUseFallback(str string, fallback string) bool {
	length := len(str)
	missingChars := strings.Count(str, "�")
	spaceChars := strings.Count(str, " ")
	spaceFallbackChars := strings.Count(fallback, " ")

	if missingChars == 0 {
		return float64(spaceFallbackChars) > float64(spaceChars)*1.2
	}

	ratio := float64(missingChars) / float64(length)

	return ratio > 0.2 || float64(spaceFallbackChars) > float64(spaceChars)*1.2
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
	return nlAndSpace.ReplaceAllString(strings.Trim(str, " "), " ")
}

func intToRoman(number int) string {
	maxRomanNumber := 3999
	if number > maxRomanNumber {
		return strconv.Itoa(number)
	}

	conversions := []struct {
		value int
		digit string
	}{
		{1000, "M"},
		{900, "CM"},
		{500, "D"},
		{400, "CD"},
		{100, "C"},
		{90, "XC"},
		{50, "L"},
		{40, "XL"},
		{10, "X"},
		{9, "IX"},
		{5, "V"},
		{4, "IV"},
		{1, "I"},
	}

	var roman strings.Builder
	for _, conversion := range conversions {
		for number >= conversion.value {
			roman.WriteString(conversion.digit)
			number -= conversion.value
		}
	}

	return roman.String()
}

func intToAZ(number int) string {
	quot := (number - 1) / 26
	rem := number % 26

	if rem == 0 {
		rem = 26
	}

	alpha := "abcdefghijklmnopqrstuvwxyz"

	return strings.Repeat(string(alpha[rem-1]), quot+1)
}

// https://www.w3.org/TR/WCAG20-TECHS/PDF17.html#PDF17-ex2
func GetPageLabelMap(numPages int, labels core.PdfObject) map[int]string {
	labelMap := map[int]string{}

	asIO, ok := labels.(*core.PdfIndirectObject)
	if !ok {
		return nil
	}

	asOD, ok := asIO.PdfObject.(*core.PdfObjectDictionary)
	if !ok {
		return nil
	}

	nums := asOD.Get("Nums")

	arr, ok := nums.(*core.PdfObjectArray)
	if !ok {
		return nil
	}

	indexMap := map[int]*core.PdfObjectDictionary{}

	for i := 0; i < arr.Len(); i += 2 {
		idx, ok := arr.Get(i).(*core.PdfObjectInteger)
		if !ok {
			continue
		}

		obj, ok := arr.Get(i + 1).(*core.PdfIndirectObject)
		if !ok {
			continue
		}

		dict, ok := obj.PdfObject.(*core.PdfObjectDictionary)
		if !ok {
			continue
		}

		indexMap[int(*idx)] = dict
	}

	var curr *core.PdfObjectDictionary
	curPage := 0

	for i := 0; i < numPages; i++ {
		dict, ok := indexMap[i]
		if ok {
			curr = dict
			curPage = 0
		}

		if curr != nil {
			s, ok := curr.Get("S").(*core.PdfObjectName)

			if ok {
				st, _ := curr.Get("St").(*core.PdfObjectInteger)
				p, _ := curr.Get("P").(*core.PdfObjectString)
				page := curPage

				if st != nil {
					page = int(*st) + curPage
				} else {
					page = curPage + 1
				}

				pageStr := strconv.Itoa(page)

				switch s.String() {
				case "r":
					pageStr = strings.ToLower(intToRoman(page))
				case "R":
					pageStr = strings.ToUpper(intToRoman(page))
				case "a":
					pageStr = strings.ToLower(intToAZ(page))
				case "A":
					pageStr = strings.ToUpper(intToAZ(page))
				}

				if p != nil {
					pageStr = p.Str() + pageStr
				}

				labelMap[i] = pageStr
			}
		}

		curPage++
	}

	return labelMap
}

func GetMediaBox(page *model.PdfPage) *model.PdfRectangle {
	if page.MediaBox != nil {
		return page.MediaBox
	}

	parent, ok := page.Parent.(*core.PdfIndirectObject)

	for ok {
		dict, dOk := parent.PdfObject.(*core.PdfObjectDictionary)
		if !dOk {
			return nil
		}

		mb, mbOk := dict.Get("MediaBox").(*core.PdfObjectArray)
		if !mbOk {
			parent, ok = dict.Get("Parent").(*core.PdfIndirectObject)
			continue
		}

		rect, err := mb.ToFloat64Array()
		if err != nil {
			return nil
		}

		if len(rect) < 4 {
			return nil
		}

		return &model.PdfRectangle{
			Llx: rect[0],
			Lly: rect[1],
			Urx: rect[2],
			Ury: rect[3],
		}
	}

	return nil
}
