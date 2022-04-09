# VERTION DESCRIPTION #

## ver 0.3.3 // 2020.01.17 ##

* add to type CodePage field BOM,  
* refactoring IDCodePage.String() - get name from global var CodepageDic  
* add IDCodePage.StringHasBom(s string) bool - checks that the string s contains characters BOM matching this given codepage  
* add IDCodePage.DeleteBom(s string) string - return string without prefix bom bytes

### todo ###

   - UTF16LE & UTF16BE not recognized correctly if file no contains russian characters
_____________________________

## ver 0.3.4 // 2020.01.17 ##

* change recognition of UTF16LE and UTF16BE
* add test for UTF16LE and UTF16BE without russian

_____________________________

## ver 0.3.5 // 2020.01.27 ##

* minor updates

### todo ###

   - test with multithreading __not__ pass, 

_____________________________

## ver 0.4.0 // 2020.01.29 ##

* multithreading support updates
* add multithreading tests
* rename exported functions
* hide global var CodepageDic from export, rename to codepageDic

### todo ###

   - string UTF32 w/o bom and w/o russian char detect as UTF16

_____________________________

## ver 0.4.1 // 2020.02.05 ##

* add function NewReader() - convertion to UTF8 with automatic detection
* add function NewReaderCP() - convertion from UTF8 to the specified codepage

### todo ###

   - string UTF32 w/o bom and w/o russian char detect as UTF16

_____________________________

## ver 0.5.1 // 2020.02.13 ##

* rename function NewReaderCP() to NewReaderTo()
* add tests
* add samples

_____________________________

## ver 0.5.2 // 2020.07.09 ##

* add license
* add to code_pages_id.go new alias for KOI-8

### todo ###

   - string UTF32 w/o bom and w/o russian char detect as UTF16
