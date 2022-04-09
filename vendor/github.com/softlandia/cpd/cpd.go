//Package cpd - code page detect
// (c) 2020 softlandia@gmail.com
package cpd
	
import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"unicode"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/htmlindex"
	"golang.org/x/text/transform"
)

// ReadBufSize - byte count for reading from file, func FileCodePageDetect()
var ReadBufSize int = 1024

// SupportedEncoder - check codepage name
func SupportedEncoder(cpn string) bool {
	return codepageByName(cpn) != ASCII
}

// FileCodepageDetect - detect codepage of text file
func FileCodepageDetect(fn string, stopStr ...string) (IDCodePage, error) {
	iFile, err := os.Open(fn)
	if err != nil {
		return ASCII, err
	}
	defer iFile.Close()
	return CodepageDetect(iFile)
}

// CodepageDetect - detect code page of ascii data from reader 'r'
func CodepageDetect(r io.Reader) (IDCodePage, error) {
	if r == nil {
		return ASCII, nil
	}
	buf, err := bufio.NewReader(r).Peek(ReadBufSize)
	if (err != nil) && (err != io.EOF) {
		return ASCII, err
	}
	//match code page from BOM, support: utf-8, utf-16le, utf-16be, utf-32le or utf-32be
	if idCodePage, ok := CheckBOM(buf); ok {
		return idCodePage, nil
	}
	if ValidUTF8(buf) {
		return UTF8, nil
	}
	return CodepageAutoDetect(buf), nil
}

// CodepageAutoDetect - auto detect code page of input content
func CodepageAutoDetect(b []byte) IDCodePage {
	return NewCodepageDic().Match(b)
}

// FileConvertCodepage - replace code page text file from one to another
// support convert only from/to Windows1251/IBM866
func FileConvertCodepage(fileName string, fromCP, toCP IDCodePage) error {
	switch {
	case fromCP == toCP:
		return nil
	case (fromCP != CP1251) && (fromCP != CP866):
		return nil
	case (toCP != CP1251) && (toCP != CP866):
		return nil
	}
	iFile, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer iFile.Close()

	//TODO need using system tmp folder
	tmpFileName := fileName + "~"
	oFile, err := os.Create(tmpFileName)
	if err != nil {
		return err
	}
	defer oFile.Close()

	s := ""
	iScanner := bufio.NewScanner(iFile)
	for i := 0; iScanner.Scan(); i++ {
		s = iScanner.Text()
		s, err = StrConvertCodepage(s, fromCP, toCP)
		if err != nil {
			oFile.Close()
			os.Remove(tmpFileName)
			return fmt.Errorf("code page convert error on file '%s': %v", fileName, err)
		}
		fmt.Fprintf(oFile, "%s\n", s)
	}
	oFile.Close()
	iFile.Close()
	return os.Rename(tmpFileName, fileName)
}

//IsSeparator - return true if input rune is SPACE or PUNCT
func IsSeparator(r rune) bool {
	return unicode.IsPunct(r) || unicode.IsSpace(r)
}

// CodepageAsString - return name of char set with id codepage
// if codepage not exist - return ""
func CodepageAsString(codepage IDCodePage) string {
	return codepageDic[codepage].name
}

// StrConvertCodepage - convert string from one code page to another
// function for future, at now support convert only from/to Windows1251/IBM866
func StrConvertCodepage(s string, fromCP, toCP IDCodePage) (string, error) {
	if len(s) == 0 {
		return "", nil
	}
	if fromCP == toCP {
		return s, nil
	}

	var err error

	switch fromCP {
	case CP866:
		s, _, err = transform.String(charmap.CodePage866.NewDecoder(), s)
	case CP1251:
		s, _, err = transform.String(charmap.Windows1251.NewDecoder(), s)
	}
	switch toCP {
	case CP866:
		s, _, err = transform.String(charmap.CodePage866.NewEncoder(), s)
	case CP1251:
		s, _, err = transform.String(charmap.Windows1251.NewEncoder(), s)
	}
	return s, err
}

func checkBomExist(r io.Reader) bool {
	buf, _ := bufio.NewReader(r).Peek(4)
	_, res := CheckBOM(buf)
	return res
}

var (
	errUnknown                   = errors.New("htmlindex: unknown Encoding")
	errInputIsNil                = errors.New("cpd: input reader is nil")
	errUnsupportedCodepage       = errors.New("cpd: codepage not support encode/decode")
	errUnsupportedOutputCodepage = errors.New("cpd: output codepage not support encode")
)

// NewReader - conversion to UTF-8
// return input reader if input contain less 4 bytes
// return input reader if input contain ASCII data
// if cpn[0] exist, then using it as input codepage name
func NewReader(r io.Reader, cpn ...string) (io.Reader, error) {
	if r == nil {
		return r, errInputIsNil
	}
	tmpReader := bufio.NewReader(r)
	var err error
	cp := ASCII
	if len(cpn) > 0 {
		cp = codepageByName(cpn[0])
	}
	if cp == ASCII {
		cp, err = CodepageDetect(tmpReader)
	}
	//TODO внимательно нужно посмотреть что может вернуть CodepageDetect()
	//эти случаи обработать, например через func unsupportedCodepageToDecode(cp)
	switch {
	case (cp == UTF32) || (cp == UTF32BE) || (cp == UTF32LE):
		return r, errUnsupportedCodepage
	case cp == ASCII: // кодировку определить не удалось, неизвестную кодировку возвращаем как есть
		return r, errUnknown
	case err != nil: // и если ошибка при чтении, то возвращаем как есть
		return r, err
	}

	if checkBomExist(tmpReader) {
		//ошибку не обрабатываем, если мы здесь, то эти байты мы уже читали
		tmpReader.Read(make([]byte, cp.BomLen())) // считываем в никуда количество байт занимаемых BOM этой кодировки
	}
	if cp == UTF8 {
		return tmpReader, nil // когда удалили BOM тогда можно вернуть UTF-8, ведь его конвертировать не нужно
	}
	//ошибку не обрабатываем, htmlindex.Get() возвращает ошибку только если не найдена кодировка, здесь это уже невозможно
	//здесь cp может содержать только кодировки имеющиеся в htmlindex
	e, _ := htmlindex.Get(cp.String())
	r = transform.NewReader(tmpReader, e.NewDecoder())
	return r, nil
}

// NewReaderTo - creates a new reader encoding from UTF-8 to the specified codepage
// return input reader and error if output codepage not found, or unsupport encoding
// if input str contains the BOM char, then BOM be deleted
func NewReaderTo(r io.Reader, cpn string) (io.Reader, error) {
	cpTo := codepageByName(cpn)
	if cpTo == ASCII {
		return r, errUnsupportedOutputCodepage
	}
	tmpReader := UTF8.DeleteBomFromReader(bufio.NewReader(r))
	if cpTo == UTF8 {
		return tmpReader, nil
	}
	e, _ := htmlindex.Get(cpTo.String())
	r = transform.NewReader(tmpReader, e.NewEncoder())
	return r, nil
}
