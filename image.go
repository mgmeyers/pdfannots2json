package main

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

func handleImageAnnot(
	page *model.PdfPage,
	pageImg image.Image,
	pageIndex int,
	annotation *model.PdfAnnotation,
	x float64,
	y float64,
	id string,
) *Annotation {
	ctx := annotation.GetContext()
	width := page.MediaBox.Width()
	height := page.MediaBox.Height()

	if *page.Rotate == 90 || *page.Rotate == 270 {
		width = page.MediaBox.Height()
		height = page.MediaBox.Width()
	}

	scale := float64(pageImg.Bounds().Max.X) / width

	objArr, ok := ctx.(*model.PdfAnnotationSquare).Rect.(*core.PdfObjectArray)
	if !ok {
		return nil
	}

	annotRect, err := objArr.ToFloat64Array()
	endIfErr(err)

	annotRect = applyPageRotation(page, annotRect)

	if args.NoWrite != true {
		if _, err := os.Stat(args.ImageOutputPath); os.IsNotExist(err) {
			os.MkdirAll(args.ImageOutputPath, os.ModePerm)
		}
	}

	imagePath := fmt.Sprintf(
		"%s/%s-%d-x%d-y%d.%s",
		args.ImageOutputPath,
		args.ImageBaseName,
		pageIndex+1,
		int(annotRect[0]),
		int(annotRect[1]),
		args.ImageFormat,
	)

	crop := image.Rect(
		int(math.Round(annotRect[0]*scale)),
		int(math.Round((height-annotRect[1])*scale)),
		int(math.Round(annotRect[2]*scale)),
		int(math.Round((height-annotRect[3])*scale)),
	)

	cropped, err := cropImage(pageImg, crop)
	endIfErr(err)

	if args.NoWrite != true {
		writeImage(
			cropped,
			imagePath,
			args.ImageFormat,
			args.ImageQuality,
		)
	}

	comment := ""

	if annotation.Contents != nil {
		comment = removeNul(annotation.Contents.String())
	}

	builtAnnot := &Annotation{
		Color:         getColor(annotation),
		ColorCategory: getColorCategory(annotation),
		Comment:       comment,
		ImagePath:     imagePath,
		Type:          Image,
		Page:          pageIndex + 1,
		X:             x,
		Y:             y,
		ID:            id,
	}

	date := getDate(annotation)

	if date != nil {
		builtAnnot.Date = date.Format(time.RFC3339)
	}

	return builtAnnot
}

type subImager interface {
	SubImage(r image.Rectangle) image.Image
}

func cropImage(img image.Image, crop image.Rectangle) (image.Image, error) {
	simg, ok := img.(subImager)
	if !ok {
		return nil, fmt.Errorf("image does not support cropping")
	}

	return simg.SubImage(crop), nil
}

func writeImage(img image.Image, name string, format string, quality int) error {
	if format == "jpg" {
		return writeJPGImage(img, name, quality)
	}

	return writePNGImage(img, name)
}

func writeJPGImage(img image.Image, name string, quality int) error {
	fd, err := os.Create(name)
	endIfErr(err)

	defer fd.Close()
	return jpeg.Encode(fd, img, &jpeg.Options{Quality: quality})
}

func writePNGImage(img image.Image, name string) error {
	fd, err := os.Create(name)
	endIfErr(err)

	defer fd.Close()
	return png.Encode(fd, img)
}
