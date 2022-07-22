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
	PageLabel     string  `json:"pageLabel"`
	Type          string  `json:"type"`
	X             float64 `json:"x"`
	Y             float64 `json:"y"`
	SortIndex     string  `json:"-"`
}

type BySortIndex []*Annotation

func (a BySortIndex) Len() int { return len(a) }
func (a BySortIndex) Less(i, j int) bool {
	return a[i].SortIndex < a[j].SortIndex
}
func (a BySortIndex) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
