# pdf-annots2json

Extracts annotations from PDF and converts them to a JSON list.

Supported annotations:
- highlight
- strike
- underline
- text (also called notes)
- rectangle
  - *Note: rectangle annotations are exported as images*

`pdf-annots2json` uses [UniPDF](https://github.com/unidoc/unipdf/tree/v3.9.0/) to extract annotations and [MuPDF](https://mupdf.com/) to extract images from PDFs.

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

## Supported platforms

- Mac (intel): `pdf-annots2json.darwin.amd64.tar.gz`
- Mac (M1): `pdf-annots2json.darwin.arm64.tar.gz`
- Linux (x64): `pdf-annots2json.linux.amd64.tar.gz`
- Windows (x64): `pdf-annots2json.windows.amd64.zip`