package main

import (
	"encoding/json"
	"image"
	"io"
	"log"
	"math"
	"os"
	"strings"
	"time"

	"github.com/alecthomas/kong"
	"github.com/gen2brain/go-fitz"
	"github.com/golang/geo/r2"
	"github.com/mgmeyers/unipdf/v3/extractor"
	"github.com/mgmeyers/unipdf/v3/model"
)

const version = "v0.1.0"

var args struct {
	Version         kong.VersionFlag `short:"v" help:"Display the current version of pdf-annots2json"`
	NoWrite         bool             `short:"w" help:"Do not save images to disk"`
	ImageOutputPath string           `short:"o" type:"path" help:"Output path of image annotations"`
	ImageBaseName   string           `short:"n" default:"annot" help:"Base name of saved images"`
	ImageFormat     string           `short:"f" enum:"jpg,png" default:"jpg" help:"Image format. Supports png and jpg"`
	ImageDPI        int              `short:"d" default:"120" help:"Image DPI"`
	ImageQuality    int              `short:"q" default:"90" help:"Image quality. Only applies to jpg images"`

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
	ColorCategory string `json:"colorCategory,omitempty"`
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
	kong.Parse(&args, kong.Vars{
		"version": version,
	})

	skipImages := args.ImageBaseName == "" || args.ImageOutputPath == ""

	f, err := os.Open(args.InputPDF)
	endIfErr(err)

	defer f.Close()

	seeker := io.ReadSeeker(f)

	pdfReader, err := model.NewPdfReader(seeker)
	endIfErr(err)

	fitzDoc, err := fitz.New(args.InputPDF)
	endIfErr(err)

	defer fitzDoc.Close()

	numPages, err := pdfReader.GetNumPages()
	endIfErr(err)

	collectedAnnotations := []*Annotation{}

	for i := 0; i < numPages; i++ {
		page, err := pdfReader.GetPage(i + 1)
		endIfErr(err)

		var pageImg image.Image

		if !skipImages {
			pageImg, err = fitzDoc.ImageDPI(i, float64(args.ImageDPI))
			endIfErr(err)
		}

		annotations, err := page.GetAnnotations()
		endIfErr(err)

		annots := processAnnotations(i, page, pageImg, fitzDoc, annotations, skipImages)
		collectedAnnotations = append(collectedAnnotations, annots...)
	}

	logOutput(collectedAnnotations)
}

func processAnnotations(
	pageIndex int,
	page *model.PdfPage,
	pageImg image.Image,
	fitzDoc *fitz.Document,
	annotations []*model.PdfAnnotation,
	skipImages bool,
) []*Annotation {
	annots := []*Annotation{}

	ext, err := extractor.New(page)
	endIfErr(err)

	txt, _, _, err := ext.ExtractPageText()
	endIfErr(err)

	marks := txt.Marks().Elements()
	markRects := []r2.Rect{}

	for _, mark := range marks {
		markRects = append(markRects, getMarkRect(mark))
	}

	for _, annotation := range annotations {
		if annotation == nil {
			continue
		}

		ctx := annotation.GetContext()
		annotType := getType(ctx)

		if annotType == Unsupported {
			continue
		}

		date := getDate(annotation)

		if date != nil && date.Before(args.IgnoreBefore) {
			continue
		}

		if !skipImages && annotType == Rectangle {
			annots = append(annots, handleImageAnnot(pageIndex, page, pageImg, annotation))
			continue
		}

		str := ""

		if annotType != Text {
			annoRects := getAnnotationRects(annotation)

			if annoRects == nil {
				continue
			}

			for _, anno := range annoRects {
				if !anno.IsValid() || anno.IsEmpty() {
					continue
				}

				bound := r2.EmptyRect()
				boundSet := false

				for _, mark := range markRects {
					if !mark.IsValid() || mark.IsEmpty() {
						continue
					}

					if anno.Intersects(mark) && isWithinOverlapThresh(anno, mark) {
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

				bHeight := bound.Y.Hi - bound.Y.Lo
				diff := (bHeight * 0.6) / 2

				x1 := bound.X.Lo
				y1 := (page.MediaBox.Height() - (bound.Y.Lo + diff))
				x2 := bound.X.Hi
				y2 := (page.MediaBox.Height() - (bound.Y.Hi - diff))

				annotText, err := fitzDoc.TextByBounds(
					pageIndex,
					72.0,
					float32(math.Min(x1, x2)),
					float32(math.Min(y1, y2)),
					float32(math.Max(x1, x2)),
					float32(math.Max(y1, y2)),
				)
				endIfErr(err)

				if str == "" {
					str = annotText
				} else if strings.HasSuffix(str, " ") {
					str += annotText
				} else {
					str += " " + annotText
				}
			}
		}

		comment := ""

		if annotation.Contents != nil {
			comment = removeNul(annotation.Contents.String())
		}

		builtAnnot := &Annotation{
			AnnotatedText: str,
			Color:         getColor(annotation),
			ColorCategory: getColorCategory(annotation),
			Comment:       comment,
			Type:          annotType,
			Page:          pageIndex + 1,
		}

		if date != nil {
			builtAnnot.Date = date.Format(time.RFC3339)
		}

		annots = append(annots, builtAnnot)
	}

	return annots
}
