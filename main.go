package main

import (
	"fmt"
	"image"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/alecthomas/kong"
	"github.com/gen2brain/go-fitz"
	"github.com/golang/geo/r2"
	"github.com/mgmeyers/unipdf/v3/extractor"
	"github.com/mgmeyers/unipdf/v3/model"
	"golang.org/x/sync/errgroup"
)

const version = "v0.2.4"

var args struct {
	Version      kong.VersionFlag `short:"v" help:"Display the current version of pdf-annots2json"`
	IgnoreBefore time.Time        `short:"b" help:"Ignore annotations added before this date. Must be ISO 8601 formatted"`
	InputPDF     string           `arg:"" name:"input" help:"Path to input PDF" type:"path"`

	// Images
	NoWrite         bool   `short:"w" help:"Do not save images to disk"`
	ImageOutputPath string `short:"o" type:"path" help:"Output path of image annotations"`
	ImageBaseName   string `short:"n" default:"annot" help:"Base name of saved images"`
	ImageFormat     string `short:"f" enum:"jpg,png" default:"jpg" help:"Image format. Supports png and jpg"`
	ImageDPI        int    `short:"d" default:"120" help:"Image DPI"`
	ImageQuality    int    `short:"q" default:"90" help:"Image quality. Only applies to jpg images"`
	AttemptOCR      bool   `short:"e" help:"Attempt to extract text from images. tesseract-ocr must be installed on your system"`
	OCRLang         string `short:"l" default:"eng" help:"Set the OCR language. Supports multiple languages, eg. 'eng+deu'. The desired languages must be installed"`
	TesseractPath   string `default:"tesseract" help:"Absolute path to the tesseract executable"`
	TessDataDir     string `help:"Absolute path to the tesseract data folder"`
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
	AnnotatedText string  `json:"annotatedText,omitempty"`
	Color         string  `json:"color,omitempty"`
	ColorCategory string  `json:"colorCategory,omitempty"`
	Comment       string  `json:"comment,omitempty"`
	Date          string  `json:"date,omitempty"`
	ID            string  `json:"id"`
	ImagePath     string  `json:"imagePath,omitempty"`
	OCRText       string  `json:"ocrText,omitempty"`
	Page          int     `json:"page"`
	Type          string  `json:"type"`
	X             float64 `json:"x"`
	Y             float64 `json:"y"`
}

type ByX []*Annotation

func (a ByX) Len() int           { return len(a) }
func (a ByX) Less(i, j int) bool { return a[i].X < a[j].X }
func (a ByX) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

type ByY []*Annotation

func (a ByY) Len() int           { return len(a) }
func (a ByY) Less(i, j int) bool { return a[i].Y > a[j].Y }
func (a ByY) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

func main() {
	kong.Parse(&args, kong.Vars{
		"version": version,
	})

	if args.AttemptOCR {
		checkForTesseract()
		validateLang()
	}

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

	encryption := pdfReader.GetEncryptionMethod()
	if encryption != "" {
		success, err := pdfReader.Decrypt([]byte{})
		endIfErr(err)

		if !success {
			endIfErr(fmt.Errorf("Error: PDF is encrypted, unable to decrypt"))
		}
	}

	numPages, err := pdfReader.GetNumPages()
	endIfErr(err)

	collectedAnnotations := make([][]*Annotation, numPages)
	g := new(errgroup.Group)
	mu := sync.Mutex{}

	for i := 0; i < numPages; i++ {
		index := i
		g.Go(func() error {
			page, err := pdfReader.GetPage(index + 1)
			if err != nil {
				return err
			}

			mu.Lock()
			annotations, err := page.GetAnnotations()
			mu.Unlock()
			if err != nil {
				return err
			}

			if len(annotations) == 0 {
				return nil
			}

			haveRectangles := false
			filtered := []*model.PdfAnnotation{}

			for _, a := range annotations {
				annotType := getType(a.GetContext())

				if annotType == Unsupported {
					continue
				}

				if annotType == Rectangle {
					haveRectangles = true
				}

				filtered = append(filtered, a)
			}

			var pageImg image.Image
			var ocrImg image.Image

			if haveRectangles && !skipImages {
				if !args.NoWrite {
					pageImg, err = fitzDoc.ImageDPI(index, float64(args.ImageDPI))
					if err != nil {
						return err
					}
				}

				if args.AttemptOCR {
					ocrImg, err = fitzDoc.ImageDPI(index, 300.0)
					if err != nil {
						return err
					}
				}
			}

			annots := processAnnotations(
				fitzDoc,
				page,
				&pageImg,
				&ocrImg,
				index,
				filtered,
				skipImages,
			)
			collectedAnnotations[index] = annots

			return nil
		})
	}

	err = g.Wait()
	endIfErr(err)

	ordered := []*Annotation{}

	for _, annots := range collectedAnnotations {
		if annots != nil && len(annots) > 0 {
			ordered = append(ordered, annots...)
		}
	}

	logOutput(ordered)
}

func processAnnotations(
	fitzDoc *fitz.Document,
	page *model.PdfPage,
	pageImg *image.Image,
	ocrImg *image.Image,
	pageIndex int,
	annotations []*model.PdfAnnotation,
	skipImages bool,
) []*Annotation {
	annots := make([]*Annotation, len(annotations))
	seenIDs := map[string]bool{}

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

	g := new(errgroup.Group)
	mu := sync.Mutex{}

	for index, annotation := range annotations {
		annotation := annotation
		index := index

		if annotation == nil {
			continue
		}

		g.Go(func() error {
			annotType := getType(annotation.GetContext())
			if annotType == Unsupported {
				return nil
			}

			date := getDate(annotation)
			if date != nil && date.Before(args.IgnoreBefore) {
				return nil
			}

			x, y := getCoordinates(annotation)

			mu.Lock()
			id := getID(seenIDs, pageIndex, x, y, annotType)
			mu.Unlock()

			if !skipImages && annotType == Rectangle {
				annots[index] = handleImageAnnot(page, pageImg, ocrImg, pageIndex, annotation, x, y, id)
				return nil
			}

			str := ""
			fallbackStr := ""

			if annotType != Text {
				annoRects := getAnnotationRects(page, annotation)

				if annoRects == nil {
					return nil
				}

				for _, anno := range annoRects {
					if !anno.IsValid() || anno.IsEmpty() {
						return nil
					}

					bounds := getBoundsFromAnnotMarks(anno, markRects)
					annotText := getTextByAnnotBounds(fitzDoc, pageIndex, page, bounds)

					if str == "" {
						str = annotText
					} else if strings.HasSuffix(str, " ") {
						str += annotText
					} else {
						str += " " + annotText
					}

					fallback := getFallbackText(text, anno, markRects, marks)

					if fallbackStr == "" {
						fallbackStr = fallback
					} else if strings.HasSuffix(fallbackStr, " ") {
						fallbackStr += fallback
					} else {
						fallbackStr += " " + fallback
					}
				}
			}

			comment := ""

			if annotation.Contents != nil {
				comment = removeNul(annotation.Contents.String())
			}

			annotatedText := str

			if shouldUseFallback(str) {
				annotatedText = fallbackStr
			}

			builtAnnot := &Annotation{
				AnnotatedText: condenseSpaces(annotatedText),
				Color:         getColor(annotation),
				ColorCategory: getColorCategory(annotation),
				Comment:       comment,
				Type:          annotType,
				Page:          pageIndex + 1,
				X:             x,
				Y:             y,
				ID:            id,
			}

			if date != nil {
				builtAnnot.Date = date.Format(time.RFC3339)
			}

			annots[index] = builtAnnot
			return nil
		})
	}

	err = g.Wait()
	endIfErr(err)

	filtered := []*Annotation{}

	for _, annot := range annots {
		if annot != nil {
			filtered = append(filtered, annot)
		}
	}

	sort.Sort(ByX(filtered))
	sort.Sort(ByY(filtered))

	return filtered
}
