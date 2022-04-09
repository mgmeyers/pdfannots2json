package cpd

import "bytes"

// Boms - byte oder mark - special bytes for
var Boms = []struct {
	Bom []byte
	id  IDCodePage
}{
	{[]byte{0xef, 0xbb, 0xbf}, UTF8},
	{[]byte{0x00, 0x00, 0xfe, 0xff}, UTF32BE},
	{[]byte{0xff, 0xfe, 0x00, 0x00}, UTF32LE},
	{[]byte{0xfe, 0xff}, UTF16BE},
	{[]byte{0xff, 0xfe}, UTF16LE},
}

//CheckBOM - check buffer for match to utf-8, utf-16le or utf-16be BOM
func CheckBOM(buf []byte) (id IDCodePage, res bool) {
	for _, b := range Boms {
		if bytes.HasPrefix(buf, b.Bom) {
			return b.id, true
		}
	}
	return ASCII, false
}
