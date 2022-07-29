package pdfutils

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"time"

	"github.com/mgmeyers/unipdf/v3/core"
	"github.com/mgmeyers/unipdf/v3/model"
)

type ImageAnnotArgs struct {
	Page            *model.PdfPage
	PageImg         *image.Image
	OCRImg          *image.Image
	PageIndex       int
	Annotation      *model.PdfAnnotation
	X               float64
	Y               float64
	ID              string
	Write           bool
	AttemptOCR      bool
	ImageOutputPath string
	ImageBaseName   string
	ImageFormat     string
	ImageQuality    int
	TessPath        string
	TessLang        string
	TessDataDir     string
}

func HandleImageAnnot(args ImageAnnotArgs) (*Annotation, error) {
	page := args.Page
	width := page.CropBox.Width()
	height := page.CropBox.Height()

	ctx := args.Annotation.GetContext()

	objArr, ok := ctx.(*model.PdfAnnotationSquare).Rect.(*core.PdfObjectArray)
	if !ok {
		return nil, nil
	}

	annotRect, err := objArr.ToFloat64Array()
	if err != nil {
		return nil, err
	}

	xAdjust := page.MediaBox.Llx - page.CropBox.Llx
	yAdjust := page.MediaBox.Lly - page.CropBox.Lly

	if *page.Rotate == 90 {
		width = page.CropBox.Height()
		height = page.CropBox.Width()

		xAdjust = page.MediaBox.Lly - page.CropBox.Lly
		yAdjust = page.MediaBox.Llx - page.CropBox.Llx
	}

	if *page.Rotate == 180 {
		xAdjust = page.CropBox.Urx - page.MediaBox.Urx
		yAdjust = page.CropBox.Ury - page.MediaBox.Ury
	}

	if *page.Rotate == 270 {
		width = page.CropBox.Height()
		height = page.CropBox.Width()

		xAdjust = page.CropBox.Ury - page.MediaBox.Ury
		yAdjust = page.CropBox.Urx - page.MediaBox.Urx
	}

	annotRect = ApplyPageRotation(args.Page, annotRect)

	annotRect[0] = annotRect[0] + xAdjust
	annotRect[1] = height - (annotRect[1] + yAdjust)
	annotRect[2] = annotRect[2] + xAdjust
	annotRect[3] = height - (annotRect[3] + yAdjust)

	if args.Write {
		if _, err := os.Stat(args.ImageOutputPath); os.IsNotExist(err) {
			if err = os.MkdirAll(args.ImageOutputPath, os.ModePerm); err != nil {
				return nil, err
			}
		} else if err != nil {
			return nil, err
		}
	}

	imagePath := fmt.Sprintf(
		"%s/%s-%d-x%d-y%d.%s",
		args.ImageOutputPath,
		args.ImageBaseName,
		args.PageIndex+1,
		int(annotRect[0]),
		int(annotRect[2]),
		args.ImageFormat,
	)

	if args.Write {
		scale := float64((*args.PageImg).Bounds().Max.X) / width

		crop := image.Rect(
			int(math.Round(annotRect[0]*scale)),
			int(math.Round(annotRect[1]*scale)),
			int(math.Round(annotRect[2]*scale)),
			int(math.Round(annotRect[3]*scale)),
		)

		cropped, err := CropImage(args.PageImg, crop)
		if err != nil {
			return nil, err
		}

		if err := WriteImage(
			&cropped,
			imagePath,
			args.ImageFormat,
			args.ImageQuality,
		); err != nil {
			return nil, err
		}
	}

	comment := ""

	if args.Annotation.Contents != nil {
		comment = RemoveNul(args.Annotation.Contents.String())
	}

	builtAnnot := &Annotation{
		Color:         GetAnnotationColor(args.Annotation),
		ColorCategory: GetAnnotationColorCategory(args.Annotation),
		Comment:       comment,
		ImagePath:     imagePath,
		Type:          Image,
		Page:          args.PageIndex + 1,
		X:             args.X,
		Y:             args.Y,
		ID:            args.ID,
	}

	date := GetAnnotationDate(args.Annotation)

	if date != nil {
		builtAnnot.Date = date.Format(time.RFC3339)
	}

	if args.AttemptOCR {
		builtAnnot.OCRText = HandleImageOCR(args.Page, args.OCRImg, annotRect, args.TessPath, args.TessLang, args.TessDataDir)
	}

	return builtAnnot, nil
}

func HandleImageOCR(
	page *model.PdfPage,
	ocrImg *image.Image,
	annotRect []float64,
	tessPath string,
	lang string,
	dataDir string,
) string {
	width := page.CropBox.Width()

	if page.Rotate != nil && (*page.Rotate == 90 || *page.Rotate == 270) {
		width = page.CropBox.Height()
	}

	ocrScale := float64((*ocrImg).Bounds().Max.X) / width

	ocrCrop := image.Rect(
		int(math.Round(annotRect[0]*ocrScale)),
		int(math.Round(annotRect[1]*ocrScale)),
		int(math.Round(annotRect[2]*ocrScale)),
		int(math.Round(annotRect[3]*ocrScale)),
	)

	ocrCropped, err := CropImage(ocrImg, ocrCrop)
	if err != nil {
		return ""
	}

	str, err := OCRImage(ocrCropped, tessPath, lang, dataDir)
	if err != nil {
		return ""
	}

	return str
}

type subImager interface {
	SubImage(r image.Rectangle) image.Image
}

func CropImage(img *image.Image, crop image.Rectangle) (image.Image, error) {
	simg, ok := (*img).(subImager)
	if !ok {
		return nil, fmt.Errorf("image does not support cropping")
	}

	return simg.SubImage(crop), nil
}

func WriteImage(img *image.Image, name string, format string, quality int) error {
	if format == "jpg" {
		return writeJPGImage(img, name, quality)
	}

	return writePNGImage(img, name)
}

func writeJPGImage(img *image.Image, name string, quality int) error {
	fd, err := os.Create(name)
	if err != nil {
		return err
	}

	defer fd.Close()
	return jpeg.Encode(fd, *img, &jpeg.Options{Quality: quality})
}

func writePNGImage(img *image.Image, name string) error {
	fd, err := os.Create(name)
	if err != nil {
		return err
	}

	defer fd.Close()
	return png.Encode(fd, *img)
}
