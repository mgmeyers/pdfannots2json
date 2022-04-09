package cpd

import (
	"bytes"
	"encoding/binary"
	"unicode/utf16"
	"unicode/utf8"
)

//unit for UTF16BE

// DecodeUTF16be - decode slice of byte from UTF16 to UTF8
func DecodeUTF16be(s string) string {
	if len(s) == 0 {
		return ""
	}
	s = UTF16BE.DeleteBom(s)
	b := []byte(s)
	u16s := make([]uint16, 1)
	ret := &bytes.Buffer{}
	b8buf := make([]byte, 4)
	for i := 0; i < len(b); i += 2 {
		u16s[0] = uint16(b[i+1]) + (uint16(b[i]) << 8)
		r := utf16.Decode(u16s)
		n := utf8.EncodeRune(b8buf, r[0])
		ret.Write(b8buf[:n])
	}
	return ret.String()
}

func matchUTF16be(b []byte, tbl *cpTable) MatchRes {
	n := len(b)/2 - 1
	if n <= 0 {
		return MatchRes{0, 0}
	}
	//два критерия используется
	//первый количество найденных русских букв
	//второй количество найденных 0x00
	//решающим является максимальный
	return MatchRes{max(matchUTF16beRu(b, tbl), matchUTF16beZerro(b)), 0}
}

// matchUTF16leZerro - вычисляет критерий по количеству нулевых байтов, текст набранный латинскими символами в колировке UTF16le будет вторым символом иметь 0x00
func matchUTF16beZerro(b []byte) int {
	zerroCount := 0
	n := len(b)/2 - 1
	for i := 0; i < n; i += 2 {
		if b[i] == 0x00 {
			zerroCount++
		}
	}
	return zerroCount
}

// matchUTF16beRu - вычисляет критерий по количеству русских букв
// tbl *codePageTable - передаётся не для нахождения кодировки, а для заполнения встречаемости популярных русских букв
func matchUTF16beRu(data []byte, tbl *cpTable) int {
	matches := 0
	n := len(data)/2 - 1
	if n <= 0 {
		return 0
	}
	count04 := 0
	for i := 0; i < n; i += 2 {
		if data[i] == 0x04 {
			count04++
		}
		t := data[i : i+2]
		d := binary.BigEndian.Uint16(t)
		j := tbl.index(rune(d))
		if j > 0 {
			(*tbl)[j].count++
		}
		if isUTF16BE(rune(d)) {
			matches++
		}
	}
	if count04 < matches {
		matches = count04
	}
	return matches
}

/*func matchUTF16beFirstLessSecond(b []byte) int {
	count := 0
	n := len(b)/2 - 1
	for i := 0; i < n; i += 2 {
		//second byte of UTF16BE usually greate than first
		if b[i] < b[i+1] {
			count++
		}
	}
	return count
}*/

const (
	cpUTF16beBeginUpperChar = 0x0410
	cpUTF16BEStopUpperChar  = 0x042F
	cpUTF16beBeginLowerChar = 0x0430
	cpUTF16BEStopLowerChar  = 0x044F
)

func isUpperUTF16BE(r rune) bool {
	return (r >= cpUTF16beBeginUpperChar) && (r <= cpUTF16BEStopUpperChar)
}

func isLowerUTF16BE(r rune) bool {
	return (r >= cpUTF16beBeginLowerChar) && (r <= cpUTF16BEStopLowerChar)
}

func isUTF16BE(r rune) bool {
	return isUpperUTF16BE(r) || isLowerUTF16BE(r)
}
