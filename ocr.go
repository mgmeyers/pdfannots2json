package main

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
	"strings"
)

func checkForTesseract() {
	if args.TesseractPath == "tesseract" {
		_, err := exec.LookPath("tesseract")
		endIfErr(err)
	} else {
		if _, err := os.Stat(args.TesseractPath); os.IsNotExist(err) {
			endIfErr(err)
		}
	}
}

func ocrImage(img image.Image) (string, error) {
	tessArgs := []string{"stdin", "stdout", "--dpi", "300", "-l", args.OCRLang}

	if args.TessDataDir != "" {
		tessArgs = append(tessArgs, "--tessdata-dir", args.TessDataDir)
	}

	cmd := exec.Command(args.TesseractPath, tessArgs...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", err
	}

	go func() {
		defer stdin.Close()
		png.Encode(stdin, img)
	}()

	var out bytes.Buffer
	cmd.Stdout = &out

	err = cmd.Run()
	if err != nil {
		return "", err
	}

	return condenseSpaces(out.String()), nil
}

func validateLang() {
	split := strings.Split(args.OCRLang, "+")

	cmd := exec.Command(args.TesseractPath, "--list-langs")
	out, err := cmd.CombinedOutput()
	endIfErr(err)

	outLines := strings.Split(string(out), "\n")

	for _, lang := range split {
		found := false
		for _, line := range outLines {
			if strings.Trim(line, "\n ") == lang {
				found = true
				break
			}
		}

		if !found {
			endIfErr(fmt.Errorf("Tesseract language `%s` not installed", lang))
		}
	}
}
