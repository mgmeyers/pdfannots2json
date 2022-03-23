# pdf-annots2json

Extracts annotations from PDF and converts them to a JSON list. Rectangle annotations are exported as images.

```
Usage: pdf-annots2json <input>

Arguments:
  <input>    Path to input PDF

Flags:
  -h, --help                        Show context-sensitive help.
  -o, --image-output-path=STRING    Output path of image annotations
  -n, --image-base-name=STRING      Base name of saved images
  -f, --image-format="jpg"          Image format. Supports png and jpg
  -d, --image-dpi=120               Image DPI
  -q, --image-quality=90            Image quality. Only applies to jpg images
  -b, --ignore-before=TIME          Ignore annotations added before this date. Must be ISO
                                    8601 formatted
```