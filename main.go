package main

import (
	"encoding/json"
	"fmt"
	"image"
	"io"
	"log"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/alecthomas/kong"
	"github.com/golang/geo/r2"
	"github.com/mgmeyers/go-fitz"
	"github.com/mgmeyers/pdfannots2json/pdfutils"
	"github.com/mgmeyers/unipdf/v3/extractor"
	"github.com/mgmeyers/unipdf/v3/model"
	"golang.org/x/sync/errgroup"
)

const version = "v1.0.4"

var args struct {
	Version      kong.VersionFlag `short:"v" help:"Display the current version of pdfannots2json"`
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

func logOutput(annots []*pdfutils.Annotation) {
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

func main() {
	kong.Parse(&args, kong.Vars{
		"version": version,
	})

	if args.AttemptOCR {
		haveTess := pdfutils.CheckForTesseract(args.TesseractPath)
		if !haveTess {
			endIfErr(fmt.Errorf("Error: %s not found", args.TesseractPath))
		}

		valid := pdfutils.ValidateLang(args.TesseractPath, args.OCRLang)
		if !valid {
			endIfErr(fmt.Errorf("Error: %s not a valid tesseract language string", args.OCRLang))
		}
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

	collectedAnnotations := make([][]*pdfutils.Annotation, numPages)
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
				annotType := pdfutils.GetAnnotationType(a.GetContext())

				if annotType == pdfutils.Unsupported {
					continue
				}

				if annotType == pdfutils.Rectangle {
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

	filtered := []*pdfutils.Annotation{}

	for _, annots := range collectedAnnotations {
		if annots != nil && len(annots) > 0 {
			filtered = append(filtered, annots...)
		}
	}

	logOutput(filtered)
}

func processAnnotations(
	fitzDoc *fitz.Document,
	page *model.PdfPage,
	pageImg *image.Image,
	ocrImg *image.Image,
	pageIndex int,
	annotations []*model.PdfAnnotation,
	skipImages bool,
) []*pdfutils.Annotation {
	annots := make([]*pdfutils.Annotation, len(annotations))
	seenIDs := map[string]bool{}

	ext, err := extractor.New(page)
	endIfErr(err)

	txt, _, _, err := ext.ExtractPageText()
	endIfErr(err)

	text := txt.Text()
	marks := txt.Marks().Elements()
	markRects := []r2.Rect{}

	for _, mark := range marks {
		markRects = append(markRects, pdfutils.GetMarkRect(mark))
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
			annotType := pdfutils.GetAnnotationType(annotation.GetContext())
			if annotType == pdfutils.Unsupported {
				return nil
			}

			date := pdfutils.GetAnnotationDate(annotation)
			if date != nil && date.Before(args.IgnoreBefore) {
				return nil
			}

			x, y := pdfutils.GetCoordinates(annotation)

			mu.Lock()
			id := pdfutils.GetAnnotationID(seenIDs, pageIndex, x, y, annotType)
			mu.Unlock()

			if !skipImages && annotType == pdfutils.Rectangle {
				imgAnnot, err := pdfutils.HandleImageAnnot(pdfutils.ImageAnnotArgs{
					Page:            page,
					PageImg:         pageImg,
					PageIndex:       pageIndex,
					OCRImg:          ocrImg,
					AttemptOCR:      args.AttemptOCR,
					Annotation:      annotation,
					X:               x,
					Y:               y,
					ID:              id,
					Write:           !args.NoWrite,
					ImageOutputPath: args.ImageOutputPath,
					ImageBaseName:   args.ImageBaseName,
					ImageFormat:     args.ImageFormat,
					ImageQuality:    args.ImageQuality,
					TessPath:        args.TesseractPath,
					TessLang:        args.OCRLang,
					TessDataDir:     args.TessDataDir,
				})

				if err != nil {
					return err
				}

				annots[index] = imgAnnot
				return nil
			}

			str := ""
			fallbackStr := ""

			if annotType != pdfutils.Text {
				annoRects := pdfutils.GetAnnotationRects(page, annotation)

				if annoRects == nil {
					return nil
				}

				for _, anno := range annoRects {
					if !anno.IsValid() || anno.IsEmpty() {
						return nil
					}

					bounds := pdfutils.GetBoundsFromAnnotMarks(anno, markRects)
					annotText, err := pdfutils.GetTextByAnnotBounds(fitzDoc, pageIndex, page, bounds)
					endIfErr(err)

					if str == "" {
						str = annotText
					} else if strings.HasSuffix(str, " ") {
						str += annotText
					} else {
						str += " " + annotText
					}

					fallback := pdfutils.GetFallbackText(text, anno, markRects, marks)

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
				comment = pdfutils.RemoveNul(annotation.Contents.String())
			}

			annotatedText := str

			if pdfutils.ShouldUseFallback(str) {
				annotatedText = fallbackStr
			}

			builtAnnot := &pdfutils.Annotation{
				AnnotatedText: pdfutils.CondenseSpaces(annotatedText),
				Color:         pdfutils.GetAnnotationColor(annotation),
				ColorCategory: pdfutils.GetAnnotationColorCategory(annotation),
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

	filtered := []*pdfutils.Annotation{}

	for _, annot := range annots {
		if annot != nil {
			filtered = append(filtered, annot)
		}
	}

	sort.Sort(pdfutils.ByCoord(filtered))

	return filtered
}
