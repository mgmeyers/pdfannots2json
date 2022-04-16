# pdf-annots2json

Extracts annotations from PDF and converts them to a JSON list.

Supported annotations:
- highlight
- strike
- underline
- text (also called notes)
- rectangle
  - *Note: rectangle annotations are exported as images*

`pdf-annots2json` uses [UniPDF](https://github.com/unidoc/unipdf/tree/v3.9.0/) to extract annotations and [MuPDF (Fitz)](https://mupdf.com/) to extract images from PDFs.

```
Usage: pdf-annots2json <input>

Arguments:
  <input>    Path to input PDF

Flags:
  -h, --help                          Show context-sensitive help.
  -v, --version                       Display the current version of pdf-annots2json
  -b, --ignore-before=TIME            Ignore annotations added before this date. Must be ISO 8601 formatted
  -w, --no-write                      Do not save images to disk
  -o, --image-output-path=STRING      Output path of image annotations
  -n, --image-base-name="annot"       Base name of saved images
  -f, --image-format="jpg"            Image format. Supports png and jpg
  -d, --image-dpi=120                 Image DPI
  -q, --image-quality=90              Image quality. Only applies to jpg images
  -e, --attempt-ocr                   Attempt to extract text from images. tesseract-ocr must be installed on your system
  -l, --ocr-lang="eng"                Set the OCR language. Supports multiple languages, eg. 'eng+deu'. The desired languages must be installed
      --tesseract-path="tesseract"    Absolute path to the tesseract executable
      --tess-data-dir=STRING          Absolute path to the tesseract data folder
```


## Supported platforms (see releases)

- Mac (intel, M1)
- Linux (x64)
- Windows (x64)

## OCR

Using the `--attempt-ocr` flag instructs `pdf-annots2json` to extract text from the images created by rectangle annotations. This requires that `tesseract` is installed on your system, including the appropriate language data (by default tesseract only support english). Tesseract can be installed from homebrew on mac, various linux package managers, and from here on windows: https://github.com/UB-Mannheim/tesseract/wiki Additional language files can be downloaded here: https://github.com/tesseract-ocr/tessdata

## Sample output

```json
[
  {
    "color": "#7fff7f",
    "colorCategory": "Green",
    "date": "2022-03-14T19:57:56Z",
    "id": "rectangle-p1x26y636",
    "imagePath": "/some/path/annot-1-x26-y636.jpg",
    "ocrText": "Fabio D’Antoni !-23-*®, Alessio Matiz 23, Franco Fabbro 2“ and Cristiano Crescentini 24© ",
    "page": 1,
    "type": "image",
    "x": 26.43,
    "y": 636.51
  },
  {
    "annotatedText": "Objectives: We explored the effects of a single 40-min session ",
    "color": "#ff7f7f",
    "colorCategory": "Red",
    "date": "2022-03-14T19:56:25Z",
    "id": "highlight-p1x205y514",
    "page": 1,
    "type": "highlight",
    "x": 205.42,
    "y": 514.86
  },
  {
    "annotatedText": "processing of distressing memories reported by a non-clinical sample of adult participants. Design: A within-subject design was used. Methods: Participants (n = 40 Psychologists/MDs) reported four distressing memories",
    "color": "#ffff7f",
    "colorCategory": "Yellow",
    "comment": "This is a highlight",
    "date": "2022-03-14T19:57:17Z",
    "id": "highlight-p1x166y462",
    "page": 1,
    "type": "highlight",
    "x": 166.02,
    "y": 462.99
  },
  {
    "annotatedText": "Post-Intervention, Follow-up. Results: SUD scores associated with EMDR, BSP, and BSM signifcantly decreased from Pre- to Post-Intervention (p \u003c 0.001). At Post-Intervention and Follow-up, EMDR and BSP SUD scores were signifcantly lower than BSM and BR scores (p \u003c 0.02). At both Post-Intervention a",
    "color": "#ffff7f",
    "colorCategory": "Yellow",
    "comment": "This is an underline",
    "date": "2022-03-14T19:57:11Z",
    "id": "underline-p1x166y385",
    "page": 1,
    "type": "underline",
    "x": 166.02,
    "y": 385.17
  },
  {
    "annotatedText": "Keywords: psychotherapy; distressing memories; EMDR; Brainspotting; body scan meditation; mindfulness; bottom-up therapy; body-oriented intervention; trauma; stress",
    "color": "#ff7fff",
    "colorCategory": "Magenta",
    "comment": "This is a strike",
    "date": "2022-03-14T19:57:07Z",
    "id": "strike-p1x166y295",
    "page": 1,
    "type": "strike",
    "x": 166.39,
    "y": 295.38
  },
  {
    "color": "#ff7fff",
    "colorCategory": "Magenta",
    "comment": "Hello",
    "date": "2022-03-14T19:57:00Z",
    "id": "text-p1x351y197",
    "page": 1,
    "type": "text",
    "x": 351.45,
    "y": 197.36
  },
  {
    "annotatedText": "In clinical contexts, however, distressing ",
    "color": "#7fff7f",
    "colorCategory": "Green",
    "date": "2022-03-14T19:57:23Z",
    "id": "highlight-p1x187y166",
    "page": 1,
    "type": "highlight",
    "x": 187.65,
    "y": 166.72
  },
  {
    "annotatedText": "and may originate from both “big trauma” (“T”), such as life-threatening experiences and sexual violence, o",
    "color": "#7fff7f",
    "colorCategory": "Green",
    "comment": "A green highlight",
    "date": "2022-03-14T19:57:32Z",
    "id": "highlight-p1x166y129",
    "page": 1,
    "type": "highlight",
    "x": 166.07,
    "y": 129.02
  }
]
```
