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
	pageIndex int,
	page *model.PdfPage,
	pageImg image.Image,
	annotation *model.PdfAnnotation,
) *Annotation {
	ctx := annotation.GetContext()
	scale := float64(pageImg.Bounds().Max.X) / page.MediaBox.Width()
	annotRect, err := ctx.(*model.PdfAnnotationSquare).Rect.(*core.PdfObjectArray).ToFloat64Array()
	endIfErr(err)

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
		int(math.Round((page.MediaBox.Height()-annotRect[1])*scale)),
		int(math.Round(annotRect[2]*scale)),
		int(math.Round((page.MediaBox.Height()-annotRect[3])*scale)),
	)

	cropped, err := cropImage(pageImg, crop)
	endIfErr(err)

	if args.NoWrite != true {
		writeImage(
			args.ImageFormat,
			cropped,
			imagePath,
			args.ImageQuality,
		)
	}

	comment := ""

	if annotation.Contents != nil {
		comment = removeNul(annotation.Contents.String())
	}

	return &Annotation{
		Color:     getColor(annotation),
		Comment:   comment,
		Date:      getDate(annotation).Format(time.RFC3339),
		ImagePath: imagePath,
		Type:      Image,
		Page:      pageIndex + 1,
	}
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

func writeImage(format string, img image.Image, name string, quality int) error {
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
