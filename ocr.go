package main

import (
	"fmt"
	"image"
	"image/png"
	"os/exec"
	"strings"
)

func checkForTesseract() {
	_, err := exec.LookPath("tesseract")
	endIfErr(err)
}

func ocrImage(img image.Image) (string, error) {
	tessArgs := []string{"stdin", "stdout", "--dpi", "300", "-l", args.OCRLang}

	cmd := exec.Command("tesseract", tessArgs...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", err
	}

	go func() {
		defer stdin.Close()
		png.Encode(stdin, img)
	}()

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	return condenseSpaces(string(out)), nil
}

func validateLang(langStr string) {
	split := strings.Split(langStr, "+")

	cmd := exec.Command("tesseract", "--list-langs")
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
