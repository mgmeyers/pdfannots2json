package main

import (
	"encoding/json"
	"image"
	"io"
	"log"
	"os"
	"time"

	"github.com/alecthomas/kong"
	"github.com/gen2brain/go-fitz"
	"github.com/golang/geo/r2"
	"github.com/mgmeyers/unipdf/v3/extractor"
	"github.com/mgmeyers/unipdf/v3/model"
)

var args struct {
	NoWrite         bool   `short:"w" help:"Do not save images to disk"`
	ImageOutputPath string `short:"o" type:"path" help:"Output path of image annotations"`
	ImageBaseName   string `short:"n" help:"Base name of saved images"`
	ImageFormat     string `short:"f" enum:"jpg,png" default:"jpg" help:"Image format. Supports png and jpg"`
	ImageDPI        int    `short:"d" default:"120" help:"Image DPI"`
	ImageQuality    int    `short:"q" default:"90" help:"Image quality. Only applies to jpg images"`

	IgnoreBefore time.Time `short:"b" help:"Ignore annotations added before this date. Must be ISO 8601 formatted"`

	InputPDF string `arg:"" name:"input" help:"Path to input PDF" type:"path"`
}

const (
	Highlight   string = "highlight"
	Strike             = "strike"
	Underline          = "underline"
	Text               = "text"
	Rectangle          = "rectangle"
	Image              = "image"
	Unsupported        = "unsupported"
)

type Annotation struct {
	AnnotatedText string `json:"annotatedText,omitempty"`
	Color         string `json:"color,omitempty"`
	Comment       string `json:"comment,omitempty"`
	Date          string `json:"date,omitempty"`
	ImagePath     string `json:"imagePath,omitempty"`
	Type          string `json:"type"`
	Page          int    `json:"page"`
}

func logOutput(annots []*Annotation) {
	jsonAnnots, err := json.Marshal(annots)

	endIfErr(err)

	oLog := log.New(os.Stdout, "", 0)
	oLog.Println(string(jsonAnnots))
}

func main() {
	kong.Parse(&args)

	skipImages := args.ImageBaseName == "" || args.ImageOutputPath == ""

	f, err := os.Open(args.InputPDF)
	endIfErr(err)

	defer f.Close()

	seeker := io.ReadSeeker(f)

	pdfReader, err := model.NewPdfReader(seeker)
	endIfErr(err)

	imgDoc, err := fitz.New(args.InputPDF)
	endIfErr(err)

	defer imgDoc.Close()

	numPages, err := pdfReader.GetNumPages()
	endIfErr(err)

	collectedAnnotations := []*Annotation{}

	for i := 0; i < numPages; i++ {
		page, err := pdfReader.GetPage(i + 1)
		endIfErr(err)

		var pageImg image.Image

		if !skipImages {
			pageImg, err = imgDoc.ImageDPI(i, float64(args.ImageDPI))
			endIfErr(err)
		}

		annotations, err := page.GetAnnotations()
		endIfErr(err)

		annots := processAnnotations(i, page, pageImg, annotations, skipImages)
		collectedAnnotations = append(collectedAnnotations, annots...)
	}

	logOutput(collectedAnnotations)
}

func processAnnotations(
	pageIndex int,
	page *model.PdfPage,
	pageImg image.Image,
	annotations []*model.PdfAnnotation,
	skipImages bool,
) []*Annotation {
	annots := []*Annotation{}

	ext, err := extractor.New(page)
	endIfErr(err)

	txt, _, _, err := ext.ExtractPageText()
	endIfErr(err)

	text := txt.Text()
	marks := txt.Marks().Elements()
	markRects := []r2.Rect{}

	for _, mark := range marks {
		markRects = append(markRects, getMarkRect(mark))
	}

	for _, annotation := range annotations {
		ctx := annotation.GetContext()
		annotType := getType(ctx)

		if annotType == Unsupported {
			continue
		}

		date := getDate(annotation)

		if date.Before(args.IgnoreBefore) {
			continue
		}

		if !skipImages && annotType == Rectangle {
			annots = append(annots, handleImageAnnot(pageIndex, page, pageImg, annotation))
			continue
		}

		annoRects := getAnnotationRects(annotation)

		if annoRects == nil {
			continue
		}

		str := ""

		for _, anno := range annoRects {
			if !anno.IsValid() || anno.IsEmpty() {
				continue
			}

			for i, mark := range markRects {
				if !mark.IsValid() || mark.IsEmpty() {
					continue
				}

				if anno.Intersects(mark) && isWithinOverlapThresh(anno, mark) {
					if len(marks[i].Text) > 0 && marks[i].Offset > 0 && len(str) > 0 {
						prevChar := string(text[marks[i].Offset-1])

						if prevChar == " " || prevChar == "\n" {
							str += " " + marks[i].Text
							continue
						}

					}

					str += marks[i].Text
					continue
				}
			}
		}

		comment := ""

		if annotation.Contents != nil {
			comment = removeNul(annotation.Contents.String())
		}

		annots = append(annots, &Annotation{
			AnnotatedText: str,
			Color:         getColor(annotation),
			Comment:       comment,
			Date:          date.Format(time.RFC3339),
			Type:          annotType,
			Page:          pageIndex + 1,
		})
	}

	return annots
}
