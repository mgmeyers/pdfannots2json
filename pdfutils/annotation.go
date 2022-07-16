package pdfutils

const (
	Highlight   string = "highlight"
	Strike             = "strike"
	Underline          = "underline"
	Text               = "text"
	Rectangle          = "rectangle"
	Image              = "image"
	Unsupported        = "unsupported"
)

type Annotation struct {
	AnnotatedText string  `json:"annotatedText,omitempty"`
	Color         string  `json:"color,omitempty"`
	ColorCategory string  `json:"colorCategory,omitempty"`
	Comment       string  `json:"comment,omitempty"`
	Date          string  `json:"date,omitempty"`
	ID            string  `json:"id"`
	ImagePath     string  `json:"imagePath,omitempty"`
	OCRText       string  `json:"ocrText,omitempty"`
	Page          int     `json:"page"`
	Type          string  `json:"type"`
	X             float64 `json:"x"`
	Y             float64 `json:"y"`
}

type ByCoord []*Annotation

func (a ByCoord) Len() int { return len(a) }
func (a ByCoord) Less(i, j int) bool {
	if a[i].X == a[j].X {
		return a[i].Y > a[j].Y
	} else {
		return a[i].X < a[j].X
	}
}
func (a ByCoord) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
