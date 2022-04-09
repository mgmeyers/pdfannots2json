# code page detect #

(c) softlandia@gmail.com

>download: go get -u github.com/softlandia/cpd  
>install: go install

библиотека для golang

предназначена для автоматического определения кодовой страницы текстовых файлов или потоков байт  
поддерживает следующие кодовые страницы:

| no | ID               | Name           | uint16  |
| -- | ---------------- | -------------- | ------- |
| 1. | ASCII            | "ASCII"        |      3  |
| 2. | ISOLatinCyrillic | "ISO-8859-5"   |      8  |
| 3. | CP866            | "CP866"        |   2086  |
| 4. | Windows1251      | "Windows-1251" |   2251  |
| 5. | UTF8             | "UTF-8"        |    106  |
| 6. | UTF16LE          | "UTF-16LE"     |   1014  |
| 7. | UTF16BE          | "UTF-16BE"     |   1013  |
| 8. | KOI8R            | "KOI8-R"       |   2084  |
| 9. | UTF32LE          | "UTF-32LE"     |   1019  |
| 10.| UTF32BE:         | "UTF-32BE"     |   1018  |

## особенности ##

определение делается как по наличию признака BOM в начале файла так и по эвристическому алгоритму  
если данные содержат только латинские символы (первая половина ASCII таблицы) будет определена кодировка UTF-8  
это не является ошибкой, поскольку такой файл или данные действительно можно корректно интерпретировать как UTF-8  
возможно некорректное определение файлов в кодировке UTF-32 не содержащих русских символов

>__ВНИМАНИЕ!__
>библиотека __поддерживает__ многопоточный режим

## зависимости ##

>"golang.org/x/text/encoding/charmap"  
>"golang.org/x/text/transform"  

## типы ##

**IDCodePage uint16** - индекс кодовой страницы, значения взяты из файла поставки golang golang.org\x\text\encoding\internal\identifier\mib.go
поддерживается interface String(), и можно выводить так
```
    cp := cpd.UTF8
    fmt.Printf("code page index: %d, name: %s\n", cp, cp)
    >> code page index: 106, name: UTF-8
```
## глобальные переменные ##

**ReadBufSize int = 1024** // количество байт считываемых из ридера (буфера) для определения кодировки

## функции ##

1. CodepageAutoDetect(b []byte) IDCodePage
2. CodepageDetect(r io.Reader, stopStr ...string) (IDCodePage, error)
3. FileCodepageDetect(fn string, stopStr ...string) (IDCodePage, error)
4. NewReader(r io.Reader, cpn ...string) (io.Reader, error)
5. NewReaderTo((r io.Reader, cpn string) (io.Reader, error)
6. SupportedEncoder(cpn string) bool

## описание ##

    CodepageAutoDetect(content []byte) (result IDCodePage) 
      автоматическое определеие кодировки по входному слайсу байт
      использовать вместо golang.org/x/net/html/charset.DetermineEncoding()

    CodepageDetect(r io.Reader, stopStr ...string) (IDCodePage, error)
      определяет кодовую страницу считывая поток байтов из 'r' 
      используется 'reflect.ValueOf(r).IsValid()' для проверки 'r' на существование
      считывает из 'r' первые ReadBufSize байтов
      параметр stopStr пока не используется

    FileCodepageDetect(fn string, stopStr ...string) (IDCodePage, error)
      определяет кодовую страницу считывая файл 'fn', считывает из файла первые ReadBufSize байтов
      ошибку возвращает если проблемы с открытием файла 'fn'
      возвращает cpd.ASCII если колировка не определена
    
    NewReader(r io.Reader, cpn ...string) (io.Reader, error)
      конвертация из указанной кодировки в UTF-8
      r - ридер из которого читаем
      cpn - имя кодировки в которой представлены входные данные, необязательный параметр
      создаёт новый io.Reader, чтение из которого будет в формате UTF-8,
      входная кодировка определяется автоматически, либо можно задать имя в параметре cpn
      если имя входной кодировки неверное (отсутствует в словаре) то выполняется автоопределение
      может вернуть ошибку чтения из входного ридера, либо ошибку неизвестной кодировки (кодировка из которой невозможно преобразовать в UTF-8)

    NewReaderTo(r io.Reader, cpn string) (io.Reader, error)
      конвертация из UTF-8 в целевую кодировку 
      r - ридер из которого читаем, обязательно в UTF-8
      cpn - имя кодировки в которую преобразуем данные
      создаёт новый io.Reader, чтение из которого будет в кодировке cpn,
      может вернуть ошибку чтения из входного ридера, либо ошибку неизвестной выходной кодировки

    SupportedEncoder(cpn string) bool
      проверка кодировки на возможность преобразования

## tests & static analiz ##

coverage: 89.8%  
в папке "test_files" лежат файлы для тестов, соответственно не править и не удалять
в папке "sample" примеры

1. tohex -- подаём строку и желаемую кодировку, получаем шестнадцатеричные коды символов строки в указанной кодировке. пример боевой, полученную строку можно забрать и вставить в код golang
2. detect-all-files -- выводит кодировку всех файлов найденных в текущем каталоге с указанным расширением
3. cpname -- пример работы с именами кодировок и прохода по всем кодировкам

linter.md отчёт статического анализатора golangci-lint
