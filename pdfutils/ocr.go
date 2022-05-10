package pdfutils

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"os/exec"
	"strings"
)

func CheckForTesseract(path string) bool {
	if path == "tesseract" {
		if _, err := exec.LookPath("tesseract"); err != nil {
			return false
		}
	} else {
		if _, err := os.Stat(path); err != nil {
			return false
		}
	}

	return true
}

func OCRImage(img image.Image, tessPath, lang, dataDir string) (string, error) {
	tessArgs := []string{"stdin", "stdout", "--dpi", "300", "-l", lang}

	if dataDir != "" {
		tessArgs = append(tessArgs, "--tessdata-dir", dataDir)
	}

	cmd := exec.Command(tessPath, tessArgs...)
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

	return CondenseSpaces(out.String()), nil
}

func ValidateLang(tessPath, code string) bool {
	split := strings.Split(code, "+")

	cmd := exec.Command(tessPath, "--list-langs")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}

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
			return false
		}
	}

	return true
}
