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
  -h, --help                      Show context-sensitive help.
  -w, --no-write                  Do not save images to disk
  -o, --image-output-path=STRING
                                  Output path of image annotations
  -n, --image-base-name=STRING    Base name of saved images
  -f, --image-format="jpg"        Image format. Supports png and jpg
  -d, --image-dpi=120             Image DPI
  -q, --image-quality=90          Image quality. Only applies to jpg images
  -b, --ignore-before=TIME        Ignore annotations added before this date. Must
                                  be ISO 8601 formatted
```

Sample output:

```json
[
  {
    "annotatedText": "Objectives: We explored the effects of a single 40-min session",
    "color": "#ff7f7f",
    "date": "2022-03-14T19:56:25Z",
    "type": "highlight",
    "page": 1
  },
  {
    "annotatedText": "processing of distressing memories reported by a non-clinical sample of adult participants. Design: A within-subject design was used. Methods: Participants (n = 40 Psychologists/MDs) reported four distressing memories",
    "color": "#ffff7f",
    "comment": "This is a highlight",
    "date": "2022-03-14T19:57:17Z",
    "type": "highlight",
    "page": 1
  },
  {
    "annotatedText": "Post-Intervention, Follow-up. Results: SUD scores associated with EMDR, BSP, and BSM significantly decreased from Pre- to Post-Intervention (p \u003c 0.001). At Post-Intervention and Follow-up, EMDR and BSP SUD scores were significantly lower than BSM and BR scores (p \u003c 0.02). At both Post-Intervention a",
    "color": "#ffff7f",
    "comment": "This is an underline",
    "date": "2022-03-14T19:57:11Z",
    "type": "underline",
    "page": 1
  },
  {
    "annotatedText": "Keywords: psychotherapy; distressing memories; EMDR; Brainspotting; body scan meditation; mindfulness; bottom-up therapy; body-oriented intervention; trauma; stress",
    "color": "#ff7fff",
    "comment": "This is a strike",
    "date": "2022-03-14T19:57:07Z",
    "type": "strike",
    "page": 1
  },
  {
    "color": "#ff7fff",
    "comment": "Hello, wow",
    "date": "2022-03-14T19:57:00Z",
    "type": "text",
    "page": 1
  },
  {
    "annotatedText": "In clinical contexts, however, distressing",
    "color": "#7fff7f",
    "date": "2022-03-14T19:57:23Z",
    "type": "highlight",
    "page": 1
  },
  {
    "annotatedText": "and may originate from both “big trauma” (“T”), such as life-threatening experiences and sexual violence, o",
    "color": "#7fff7f",
    "comment": "+",
    "date": "2022-03-14T19:57:32Z",
    "type": "highlight",
    "page": 1
  },
  {
    "color": "#7fff7f",
    "date": "2022-03-14T19:57:56Z",
    "imagePath": "/Users/matt/Documents/Personal/pdf-annots2json/image/image-1-x26-y636.jpg",
    "type": "image",
    "page": 1
  }
]
```

## Supported platforms

- Mac (intel): `pdf-annots2json.darwin.amd64.tar.gz`
- Mac (M1): `pdf-annots2json.darwin.arm64.tar.gz`
- Linux (x64): `pdf-annots2json.linux.amd64.tar.gz`
- Windows (x64): `pdf-annots2json.windows.amd64.zip`