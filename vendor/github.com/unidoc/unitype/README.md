# UniType - truetype font library for golang.
This library is designed for parsing and editing truetype fonts.
Useful along with UniPDF for subsetting fonts for use in PDF files.

Contains a CLI for useful operations:
```bash
$ ./truecli
truecli - TrueCLI

Usage:
  truecli [command]

Available Commands:
  help        Help about any command
  info        Get font file info
  readwrite   Read and write font file
  subset      Subset font
  validate    Validate font file

Flags:
  -h, --help              help for truecli
      --loglevel string   Log level 'debug' and 'trace' give debug information

Use "truecli [command] --help" for more information about a command.
```

for example:
```bash
$ $ ./truecli info --trec myfnt.ttf
  trec: present with 22 table records
  DSIG: 6.78 kB
  GSUB: 368 B
  LTSH: 28.53 kB
  OS/2: 96 B
  VDMX: 1.47 kB
  cmap: 124.93 kB
  cvt: 672 B
  fpgm: 2.33 kB
  gasp: 16 B
  glyf: 13.12 MB
  hdmx: 484.97 kB
  head: 54 B
  hhea: 36 B
  hmtx: 114.09 kB
  kern: 3.43 kB
  loca: 114.09 kB
  maxp: 32 B
  name: 2.54 kB
  post: 283.82 kB
  prep: 676 B
  vhea: 36 B
  vmtx: 114.09 kB 
```

