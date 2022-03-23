/*
 * This file is subject to the terms and conditions defined in
 * file 'LICENSE.md', which is part of this source code package.
 */

package unitype

import (
	"encoding/binary"
	"unicode/utf8"

	"github.com/sirupsen/logrus"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/encoding/unicode/utf32"
)

// The 258 standard mac glyph names used in 'post' format 1 and 2.
// https://developer.apple.com/fonts/TrueType-Reference-Manual/RM06/Chap6post.html
var macGlyphNames = []GlyphName{
	".notdef", ".null", "nonmarkingreturn", "space", "exclam", "quotedbl",
	"numbersign", "dollar", "percent", "ampersand", "quotesingle",
	"parenleft", "parenright", "asterisk", "plus", "comma", "hyphen",
	"period", "slash", "zero", "one", "two", "three", "four", "five",
	"six", "seven", "eight", "nine", "colon", "semicolon", "less",
	"equal", "greater", "question", "at", "A", "B", "C", "D", "E", "F",
	"G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q", "R", "S",
	"T", "U", "V", "W", "X", "Y", "Z", "bracketleft", "backslash",
	"bracketright", "asciicircum", "underscore", "grave", "a", "b",
	"c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o",
	"p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z", "braceleft",
	"bar", "braceright", "asciitilde", "Adieresis", "Aring",
	"Ccedilla", "Eacute", "Ntilde", "Odieresis", "Udieresis", "aacute",
	"agrave", "acircumflex", "adieresis", "atilde", "aring",
	"ccedilla", "eacute", "egrave", "ecircumflex", "edieresis",
	"iacute", "igrave", "icircumflex", "idieresis", "ntilde", "oacute",
	"ograve", "ocircumflex", "odieresis", "otilde", "uacute", "ugrave",
	"ucircumflex", "udieresis", "dagger", "degree", "cent", "sterling",
	"section", "bullet", "paragraph", "germandbls", "registered",
	"copyright", "trademark", "acute", "dieresis", "notequal", "AE",
	"Oslash", "infinity", "plusminus", "lessequal", "greaterequal",
	"yen", "mu", "partialdiff", "summation", "product", "pi",
	"integral", "ordfeminine", "ordmasculine", "Omega", "ae", "oslash",
	"questiondown", "exclamdown", "logicalnot", "radical", "florin",
	"approxequal", "Delta", "guillemotleft", "guillemotright",
	"ellipsis", "nonbreakingspace", "Agrave", "Atilde", "Otilde", "OE",
	"oe", "endash", "emdash", "quotedblleft", "quotedblright",
	"quoteleft", "quoteright", "divide", "lozenge", "ydieresis",
	"Ydieresis", "fraction", "currency", "guilsinglleft",
	"guilsinglright", "fi", "fl", "daggerdbl", "periodcentered",
	"quotesinglbase", "quotedblbase", "perthousand", "Acircumflex",
	"Ecircumflex", "Aacute", "Edieresis", "Egrave", "Iacute",
	"Icircumflex", "Idieresis", "Igrave", "Oacute", "Ocircumflex",
	"apple", "Ograve", "Uacute", "Ucircumflex", "Ugrave", "dotlessi",
	"circumflex", "tilde", "macron", "breve", "dotaccent", "ring",
	"cedilla", "hungarumlaut", "ogonek", "caron", "Lslash", "lslash",
	"Scaron", "scaron", "Zcaron", "zcaron", "brokenbar", "Eth", "eth",
	"Yacute", "yacute", "Thorn", "thorn", "minus", "multiply",
	"onesuperior", "twosuperior", "threesuperior", "onehalf",
	"onequarter", "threequarters", "franc", "Gbreve", "gbreve",
	"Idotaccent", "Scedilla", "scedilla", "Cacute", "cacute", "Ccaron",
	"ccaron", "dcroat",
}

const (
	platformIDUnicode   int = 0
	platformIDMacintosh     = 1
	platformIDWindows       = 3
)

// getCmapEncoding returns the cmapEncoding for the specified `platformID` and platform-specific `encodingID`.
func getCmapEncoding(platformID, encodingID int) cmapEncoding {
	switch platformID {
	case platformIDUnicode:
		return cmapEncodingUCS2
	case platformIDMacintosh:
		return cmapEncodingMacRoman
	case platformIDWindows:
		switch encodingID {
		case 0: // Symbol
			// TODO(gunnsth): Is this correct for symbol?
			return cmapEncodingUCS2
		case 1: // Unicode BMP-only (UCS-2)
			return cmapEncodingUCS2
		case 2: // Shift-JIS
			return cmapEncodingShiftJIS
		case 3: // PRC
			return cmapEncodingPRC
		case 4: // BigFive
			return cmapEncodingBig5
		case 5: // Johab.
			return cmapEncodingJohab
		case 10: // Unicode UCS-4.
			return cmapEncodingUCS4
		}
	}
	logrus.Debugf("Unsupported: PlatformID=%d, EncodingID=%d", platformID, encodingID)

	return cmapEncodingUnsupported
}

type cmapEncoding int

const (
	cmapEncodingUCS2 cmapEncoding = iota
	cmapEncodingUCS4
	cmapEncodingMacRoman
	cmapEncodingShiftJIS
	cmapEncodingPRC
	cmapEncodingBig5
	cmapEncodingJohab
	cmapEncodingUnsupported
)

// GetRuneDecoder returns a rune decoder for the given cmapEncoding.
// TODO(gunnsth): Combine this and getCmapEncoding into a single function?
func (e cmapEncoding) GetRuneDecoder() runeDecoder {
	var d *encoding.Decoder
	var charcodeBytes int

	switch e {
	case cmapEncodingUCS2:
		// UCS2 is a subset of UTF16.
		// Typically is big endian, although it is not specified explicitly in the specifications.
		d = unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM).NewDecoder()
		charcodeBytes = 2
	case cmapEncodingUCS4:
		// UCS4 is a subset of UTF32.
		d = utf32.UTF32(utf32.BigEndian, utf32.IgnoreBOM).NewDecoder()
		charcodeBytes = 4
	case cmapEncodingMacRoman:
		d = charmap.Macintosh.NewDecoder()
		charcodeBytes = 1
	case cmapEncodingShiftJIS:
		d = japanese.ShiftJIS.NewDecoder()
		charcodeBytes = 2
	case cmapEncodingPRC:
		d = simplifiedchinese.GBK.NewDecoder()
		charcodeBytes = 2
	case cmapEncodingBig5:
		d = traditionalchinese.Big5.NewDecoder()
		charcodeBytes = 2
	}

	if d == nil {
		logrus.Debugf("ERROR: Unsupported encoding (%d) - returning charcodes as runes", e)
		d = unicode.UTF8.NewDecoder()
		charcodeBytes = 1
	}

	return runeDecoder{
		Decoder:       d,
		charcodeBytes: charcodeBytes,
	}
}

// runeDecoder decodes runes from encoded byte data.
type runeDecoder struct {
	*encoding.Decoder
	charcodeBytes int // number of bytes per charcode in TTF data.
}

// ToBytes encodes `charcode` into bytes as represented in TTF data.
func (d runeDecoder) ToBytes(charcode uint32) []byte {
	b := make([]byte, d.charcodeBytes)

	switch d.charcodeBytes {
	case 1:
		b[0] = byte(charcode)
	case 2:
		binary.BigEndian.PutUint16(b, uint16(charcode))
	case 4:
		binary.BigEndian.PutUint32(b, charcode)
	default:
		logrus.Debugf("ERROR: Unsupported number of bytes per charcode: %d", d.charcodeBytes)
		return []byte{0}
	}

	return b
}

// DecodeRune decodes character codes in `b` and returns the decode rune.
func (d runeDecoder) DecodeRune(b []byte) rune {
	// Get decoded bytes (the decoder decodes to UTF8 byte format).
	decoded, err := d.Bytes(b)
	if err != nil {
		logrus.Debugf("Decoding error: %v", err)
	}

	// TODO(gunnsth): Benchmark utf8.DecodeRune vs string().
	r, _ := utf8.DecodeRune(decoded)
	return r
}
