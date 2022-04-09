package cpd

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
)

// IDCodePage - index of code page
// implements interface String()
type IDCodePage uint16

func (i IDCodePage) String() string {
	//return codePageName[i]
	return codepageDic[i].name
}

//StringHasBom - return true if input string has BOM prefix
func (i IDCodePage) StringHasBom(s string) bool {
	if len(codepageDic[i].Boms) == 0 {
		return false
	}
	return bytes.HasPrefix([]byte(s), codepageDic[i].Boms)
}

//DeleteBom - return string without prefix bom bytes
func (i IDCodePage) DeleteBom(s string) (res string) {
	res = s
	if i.StringHasBom(s) {
		res = res[len(codepageDic[i].Boms):]
	}
	return res
}

// BomLen - return lenght in bytes of BOM for this
// for codepage no have Bom, return 0
func (i IDCodePage) BomLen() int {
	for _, b := range Boms {
		if b.id == i {
			return len(b.Bom)
		}
	}
	return 0
}

// ReaderHasBom - check reader to BOM prefix
func (i IDCodePage) ReaderHasBom(r io.Reader) bool {
	buf, err := bufio.NewReader(r).Peek(i.BomLen())
	if err != nil {
		return false
	}
	return bytes.HasPrefix(buf, codepageDic[i].Boms)
}

// DeleteBomFromReader - return reader after removing BOM from it
func (i IDCodePage) DeleteBomFromReader(r io.Reader) io.Reader {
	if i.ReaderHasBom(r) {
		//ошибку не обрабатываем, если мы здесь, то эти байты мы уже читали
		r.Read(make([]byte, UTF8.BomLen())) // считываем в никуда количество байт занимаемых BOM этой кодировки
	}
	return r
}

// codepageByName - search and return codepage id by name
func codepageByName(name string) IDCodePage {
	id, ok := nameMap[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		return ASCII
	}
	return id
}

// matcher - return struct MatchRes - two criterion
// this function must be realised in each codepage
type matcher func(data []byte, tbl *cpTable) MatchRes

// container - return true if b contain in
type container func(b byte) bool

type tableElement struct {
	code  rune //rune (letter) of the alphabet that interests us
	count int  //the number of these runes found in the text
}

// cpTable - stores 9 letters, we will look for them in the text
// element with index 0 for the case of non-location
// first 9 elements lowercase, second 9 elements uppercase
type cpTable [19]tableElement

// MatchRes - result criteria
// countMatch - the number of letters founded in text
// countCvPairs - then number of pairs consonans+vowels
type MatchRes struct {
	countMatch   int
	countCvPairs int
}

func (m MatchRes) String() string {
	return fmt.Sprintf("%d, %d", m.countMatch, m.countCvPairs)
}

// CodePage - realize code page
type CodePage struct {
	id       IDCodePage //id of code page
	name     string     //name of code page
	NumByte  byte       //number of byte using in codepage
	MatchRes            //count of matching
	match    matcher    //method for calculating the criteria for the proximity of input data to this code page
	contain  container  //method return true if this codepage contain byte
	Boms     []byte     //default BOM for this codepage
	table    cpTable    //table of main alphabet rune of this code page, contain [code, count]
}

func (o CodePage) String() string {
	return fmt.Sprintf("id: %s, countMatch: %d", o.id, o.countMatch)
}

// MatchingRunes - return string with rune/counts
func (o CodePage) MatchingRunes() string {
	var sb strings.Builder
	fmt.Fprint(&sb, "rune/counts: ")
	for i, e := range o.table {
		if i != 0 {
			fmt.Fprintf(&sb, "%x/%d, ", e.code, e.count)
		}
	}
	return sb.String()
}

// FirstAlphabetPos - return position of first alphabet
// возвращает позицию первого алфавитного символа данной кодировки встреченную в отсортированном массиве
func (o CodePage) FirstAlphabetPos(d []byte) int {
	d = sortBytes(d)
	for i, b := range d {
		if o.contain(b) {
			return i
		}
	}
	return 0
}

// TCodepagesDic - type to store all supported code page
type TCodepagesDic map[IDCodePage]CodePage

// NewCodepageDic - create a new map by copying the global
func NewCodepageDic() TCodepagesDic {
	return TCodepagesDic{
		ASCII: {ASCII, "ASCII", 0, MatchRes{0, 0}, matchASCII, isASCII, []byte{},
			cpTable{{0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}}},

		CP866: {CP866, "CP866", 1, MatchRes{0, 0}, match866, is866, []byte{},
			cpTable{
				//first element serves as sign of absence
				{0, 0},
				//о          е		   а		  и			 н			т			с		  р			в
				{0xAE, 0}, {0xA5, 0}, {0xA0, 0}, {0xA8, 0}, {0xAD, 0}, {0xE2, 0}, {0xE1, 0}, {0xE0, 0}, {0xA2, 0},
				{0x8E, 0}, {0x85, 0}, {0x80, 0}, {0x88, 0}, {0x8D, 0}, {0x92, 0}, {0x91, 0}, {0x90, 0}, {0x82, 0}}},
		CP1251: {CP1251, "CP1251", 1, MatchRes{0, 0}, match1251, is1251, []byte{},
			cpTable{
				{0, 0},
				//а		    и		   н		  с			 р			в		   л		  к			 я
				{0xE0, 0}, {0xE8, 0}, {0xED, 0}, {0xF1, 0}, {0xF0, 0}, {0xE2, 0}, {0xEB, 0}, {0xEA, 0}, {0xFF, 0},
				{0xC0, 0}, {0xC8, 0}, {0xCD, 0}, {0xD1, 0}, {0xD0, 0}, {0xC2, 0}, {0xCB, 0}, {0xCA, 0}, {0xDF, 0}}},
		KOI8R: {KOI8R, "KOI8-R", 1, MatchRes{0, 0}, matchKOI8, isKOI8, []byte{},
			cpTable{
				//о		    а		   и		  т			 с			в		   л		  к			м
				{0, 0},
				{0xCF, 0}, {0xC1, 0}, {0xC9, 0}, {0xD4, 0}, {0xD3, 0}, {0xD7, 0}, {0xCC, 0}, {0xCB, 0}, {0xCD, 0},
				{0xEF, 0}, {0xE1, 0}, {0xE9, 0}, {0xF4, 0}, {0xF3, 0}, {0xF7, 0}, {0xEC, 0}, {0xEB, 0}, {0xED, 0}}},
		ISOLatinCyrillic: {ISOLatinCyrillic, "ISO-8859-5", 1, MatchRes{0, 0}, matchISO88595, isISO88595, []byte{},
			cpTable{
				//о		    а		   и		  т			 с			в		   л		  к			е
				{0, 0},
				{0xDE, 0}, {0xD0, 0}, {0xD8, 0}, {0xE2, 0}, {0xE1, 0}, {0xD2, 0}, {0xDB, 0}, {0xDA, 0}, {0xD5, 0},
				{0xBF, 0}, {0xB0, 0}, {0xB8, 0}, {0xC2, 0}, {0xC1, 0}, {0xB2, 0}, {0xBB, 0}, {0xBA, 0}, {0xB5, 0}}},
		UTF8: {UTF8, "UTF-8", 4, MatchRes{0, 0}, matchUTF8, isASCII, []byte{0xef, 0xbb, 0xbf},
			cpTable{
				{0, 0},
				//о           е				а		    и			 н			  т			   с			р			в
				{0xD0BE, 0}, {0xD0B5, 0}, {0xD0B0, 0}, {0xD0B8, 0}, {0xD0BD, 0}, {0xD182, 0}, {0xD181, 0}, {0xD180, 0}, {0xD0B2, 0},
				{0xD09E, 0}, {0xD095, 0}, {0xD090, 0}, {0xD098, 0}, {0xD0AD, 0}, {0xD0A2, 0}, {0xD0A1, 0}, {0xD0A0, 0}, {0xD092, 0}}},
		UTF16LE: {UTF16LE, "UTF-16LE", 2, MatchRes{0, 0}, matchUTF16le, isASCII, []byte{0xff, 0xfe},
			cpTable{
				{0, 0},
				//о           е				а		    и			 н			  т			   с			р			в
				{0x3E04, 0}, {0x3504, 0}, {0x1004, 0}, {0x3804, 0}, {0x3D04, 0}, {0x4204, 0}, {0x4104, 0}, {0x4004, 0}, {0x3204, 0},
				{0x1E04, 0}, {0x1504, 0}, {0x3004, 0}, {0x1804, 0}, {0x1D04, 0}, {0x2204, 0}, {0x2104, 0}, {0x2004, 0}, {0x1204, 0}}},
		UTF16BE: {UTF16BE, "UTF-16BE", 2, MatchRes{0, 0}, matchUTF16be, isASCII, []byte{0xfe, 0xff},
			cpTable{
				{0, 0},
				//о           е				а		    и			 н			  т			   с			р			в
				{0x043E, 0}, {0x0435, 0}, {0x0410, 0}, {0x0438, 0}, {0x043D, 0}, {0x0442, 0}, {0x0441, 0}, {0x0440, 0}, {0x0432, 0},
				{0x041E, 0}, {0x0415, 0}, {0x0430, 0}, {0x0418, 0}, {0x041D, 0}, {0x0422, 0}, {0x0421, 0}, {0x0420, 0}, {0x0412, 0}}},
		UTF32BE: {UTF32BE, "UTF-32BE", 4, MatchRes{0, 0}, matchUTF32be, isASCII, []byte{0x00, 0x00, 0xfe, 0xff},
			cpTable{
				{0, 0},
				{0x0, 0}, {0x0, 0}, {0x0, 0}, {0x0, 0}, {0x0, 0}, {0x0, 0}, {0x0, 0}, {0x0, 0}, {0x0, 0},
				{0x0, 0}, {0x0, 0}, {0x0, 0}, {0x0, 0}, {0x0, 0}, {0x0, 0}, {0x0, 0}, {0x0, 0}, {0x0, 0}}},
		UTF32LE: {UTF32LE, "UTF-32LE", 4, MatchRes{0, 0}, matchUTF32le, isASCII, []byte{0xff, 0xfe, 0x00, 0x00},
			cpTable{
				{0, 0},
				{0x0, 0}, {0x0, 0}, {0x0, 0}, {0x0, 0}, {0x0, 0}, {0x0, 0}, {0x0, 0}, {0x0, 0}, {0x0, 0},
				{0x0, 0}, {0x0, 0}, {0x0, 0}, {0x0, 0}, {0x0, 0}, {0x0, 0}, {0x0, 0}, {0x0, 0}, {0x0, 0}}},
	}
}

//befor detecting of code page need clear all counts
//this not for correct run, this need only if we want get correct statistic
func (o TCodepagesDic) clear() {
	for id, cp := range o {
		cp.MatchRes = MatchRes{0, 0}
		cp.table.clear()
		o[id] = cp
	}
}

// Match - return the id of code page to which the data best matches
// call function match of each codepage
func (o TCodepagesDic) Match(data []byte) (result IDCodePage) {
	result = ASCII
	maxCount := 0
	m := 0
	for id, cp := range o {
		cp.MatchRes = cp.match(data, &cp.table)
		o[id] = cp
		m = cp.MatchRes.countMatch + cp.MatchRes.countCvPairs
		if m > maxCount {
			maxCount = m
			result = id
		}
	}
	return result
}

// CodepageDic - global map of all codepage
// used for support function
var codepageDic = TCodepagesDic{
	ASCII: {ASCII, "ASCII", 0, MatchRes{0, 0}, matchASCII, isASCII, []byte{},
		cpTable{{0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}}},

	CP866: {CP866, "CP866", 1, MatchRes{0, 0}, match866, is866, []byte{},
		cpTable{
			//first element serves as sign of absence
			{0, 0},
			//о          е		   а		  и			 н			т			с		  р			в
			{0xAE, 0}, {0xA5, 0}, {0xA0, 0}, {0xA8, 0}, {0xAD, 0}, {0xE2, 0}, {0xE1, 0}, {0xE0, 0}, {0xA2, 0},
			{0x8E, 0}, {0x85, 0}, {0x80, 0}, {0x88, 0}, {0x8D, 0}, {0x92, 0}, {0x91, 0}, {0x90, 0}, {0x82, 0}}},
	CP1251: {CP1251, "CP1251", 1, MatchRes{0, 0}, match1251, is1251, []byte{},
		cpTable{
			{0, 0},
			//а		    и		   н		  с			 р			в		   л		  к			 я
			{0xE0, 0}, {0xE8, 0}, {0xED, 0}, {0xF1, 0}, {0xF0, 0}, {0xE2, 0}, {0xEB, 0}, {0xEA, 0}, {0xFF, 0},
			{0xC0, 0}, {0xC8, 0}, {0xCD, 0}, {0xD1, 0}, {0xD0, 0}, {0xC2, 0}, {0xCB, 0}, {0xCA, 0}, {0xDF, 0}}},
	KOI8R: {KOI8R, "KOI8-R", 1, MatchRes{0, 0}, matchKOI8, isKOI8, []byte{},
		cpTable{
			//о		    а		   и		  т			 с			в		   л		  к			м
			{0, 0},
			{0xCF, 0}, {0xC1, 0}, {0xC9, 0}, {0xD4, 0}, {0xD3, 0}, {0xD7, 0}, {0xCC, 0}, {0xCB, 0}, {0xCD, 0},
			{0xEF, 0}, {0xE1, 0}, {0xE9, 0}, {0xF4, 0}, {0xF3, 0}, {0xF7, 0}, {0xEC, 0}, {0xEB, 0}, {0xED, 0}}},
	ISOLatinCyrillic: {ISOLatinCyrillic, "ISO-8859-5", 1, MatchRes{0, 0}, matchISO88595, isISO88595, []byte{},
		cpTable{
			//о		    а		   и		  т			 с			в		   л		  к			е
			{0, 0},
			{0xDE, 0}, {0xD0, 0}, {0xD8, 0}, {0xE2, 0}, {0xE1, 0}, {0xD2, 0}, {0xDB, 0}, {0xDA, 0}, {0xD5, 0},
			{0xBF, 0}, {0xB0, 0}, {0xB8, 0}, {0xC2, 0}, {0xC1, 0}, {0xB2, 0}, {0xBB, 0}, {0xBA, 0}, {0xB5, 0}}},
	UTF8: {UTF8, "UTF-8", 4, MatchRes{0, 0}, matchUTF8, isASCII, []byte{0xef, 0xbb, 0xbf},
		cpTable{
			{0, 0},
			//о           е				а		    и			 н			  т			   с			р			в
			{0xD0BE, 0}, {0xD0B5, 0}, {0xD0B0, 0}, {0xD0B8, 0}, {0xD0BD, 0}, {0xD182, 0}, {0xD181, 0}, {0xD180, 0}, {0xD0B2, 0},
			{0xD09E, 0}, {0xD095, 0}, {0xD090, 0}, {0xD098, 0}, {0xD0AD, 0}, {0xD0A2, 0}, {0xD0A1, 0}, {0xD0A0, 0}, {0xD092, 0}}},
	UTF16LE: {UTF16LE, "UTF-16LE", 2, MatchRes{0, 0}, matchUTF16le, isASCII, []byte{0xff, 0xfe},
		cpTable{
			{0, 0},
			//о           е				а		    и			 н			  т			   с			р			в
			{0x3E04, 0}, {0x3504, 0}, {0x1004, 0}, {0x3804, 0}, {0x3D04, 0}, {0x4204, 0}, {0x4104, 0}, {0x4004, 0}, {0x3204, 0},
			{0x1E04, 0}, {0x1504, 0}, {0x3004, 0}, {0x1804, 0}, {0x1D04, 0}, {0x2204, 0}, {0x2104, 0}, {0x2004, 0}, {0x1204, 0}}},
	UTF16BE: {UTF16BE, "UTF-16BE", 2, MatchRes{0, 0}, matchUTF16be, isASCII, []byte{0xfe, 0xff},
		cpTable{
			{0, 0},
			//о           е				а		    и			 н			  т			   с			р			в
			{0x043E, 0}, {0x0435, 0}, {0x0410, 0}, {0x0438, 0}, {0x043D, 0}, {0x0442, 0}, {0x0441, 0}, {0x0440, 0}, {0x0432, 0},
			{0x041E, 0}, {0x0415, 0}, {0x0430, 0}, {0x0418, 0}, {0x041D, 0}, {0x0422, 0}, {0x0421, 0}, {0x0420, 0}, {0x0412, 0}}},
	UTF32BE: {UTF32BE, "UTF-32BE", 4, MatchRes{0, 0}, matchUTF32be, isASCII, []byte{0x00, 0x00, 0xfe, 0xff},
		cpTable{
			{0, 0},
			{0x0, 0}, {0x0, 0}, {0x0, 0}, {0x0, 0}, {0x0, 0}, {0x0, 0}, {0x0, 0}, {0x0, 0}, {0x0, 0},
			{0x0, 0}, {0x0, 0}, {0x0, 0}, {0x0, 0}, {0x0, 0}, {0x0, 0}, {0x0, 0}, {0x0, 0}, {0x0, 0}}},
	UTF32LE: {UTF32LE, "UTF-32LE", 4, MatchRes{0, 0}, matchUTF32le, isASCII, []byte{0xff, 0xfe, 0x00, 0x00},
		cpTable{
			{0, 0},
			{0x0, 0}, {0x0, 0}, {0x0, 0}, {0x0, 0}, {0x0, 0}, {0x0, 0}, {0x0, 0}, {0x0, 0}, {0x0, 0},
			{0x0, 0}, {0x0, 0}, {0x0, 0}, {0x0, 0}, {0x0, 0}, {0x0, 0}, {0x0, 0}, {0x0, 0}, {0x0, 0}}},
}

//foo function for default codepage ASCII
func matchASCII(b []byte, tbl *cpTable) MatchRes {
	return MatchRes{0, 0}
}

func isASCII(b byte) bool {
	return true
}
