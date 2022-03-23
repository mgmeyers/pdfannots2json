# TODO

## Discussion
1. Define exactly the purpose of the package. Its harder to make a fully generic package.  For example its one thing
to load the outer data structures (tables etc) and another one to fully parse them down to detail.

- If subsetting is the goal then its enough to know the glyph indices and where the data is kept for each index.
Then if subsetting is simply selecting a subset of glyph indices (and mapping to a new smaller set), then its
enough to know where the data for each glyph is kept and how its references and then simply dropping the data
for certain glyphs is easier than reconstructing the whole thing from memory (where we know what each value means).

- Potential use cases:
  - Subsetting - reducing font size
  - Validation - handy for developers and debugging
  - Extraction - Identifying glyph index to rune mapping and for use in PDF text extraction
  - Rendering - Rendering glyphs to image

Validation requires loading and analyzing the font tables and both their individual content and how it
fits together.  Subsetting requires locating content for input glyph indices and regenerating font with
only data for the desires glyphs (and updated tables) with the output being a new valid font file and 
an updated glyph index map mapping new desired glyph indices to their new index.
Extraction requires mapping unicode runes to glyph index (understanding the cmap tables mostly, and potentially
OCR in cases where the cmaps are unreliable).
Rendering requires quickly going from glyph index to rendering data for glyph (and caching for performance),
it does require having a detailed representation of some of the inner maps.

A performant true type package would have two levels, one quick loading where the data for a table is loaded
and then a detailed.  Lazy loading is ideal, i.e. only loading what is needed for a given application.
When regenerating a font, if tables have not been changed it makes sense that their original binary data
can be written out almost instantaneously.  No need to parse the details into data models and regenerate,
that only slows down and is not necessary, although it is a great way to test the parsing and regenerating
detail (so its good to have it as an option for development).

Regeneration: When loading data from tables and storing in a model, the way that the data is stored is not
necessarily the most practical way for using it.  For example cmaps have quite many types, some for compatibility
so its not necessarily the best way to output the data.  Cmap data is basically map: rune -> glyph index, and
there are a few preferred tables, so it might make sense to use the most common tables when outputing.
For speed, it might make sense to use the same tables as in the original and only take the data needed
for the new glyph set.

It makes sense to only export functions for the needed use cases. This allows more flexibility in changing
the internals going forward.

Currently the package is somewhere in between, i.e. it loads a bunch of stuff, probably more than is needed.

The key should be to get the needed use cases to work, then it can be refactored for better performance
later.

## List

- Lazy loading of tables.  See font.go.  Only parse tables when they are called.  Example:
GetHead() function that returns the head table: parses it if not already loaded, returns cached version if loaded.

- glyf table: Code for parsing the glyf table contents is commented out.  Need to parsing it on demand?
Related to the above point.
Key is we dont want to do this parsing except when using this table.
Also ideally we only want to parse the ones we need to use, not every single one, unless
it can be avoided or using for full validation/checking.

- glyf table extended parsing needs more test cases too looks like.

- CLI utility - would be nice to have a cli to play around with for checking and working with fonts
truecli tables fnt.ttf should output:
number of tables: 5
name: offset / size MB

trucli fnt.ttf should output some summary about the font
like number of glyphs
and some basic infos (version, creator, basic dimensions etc)

trucli tablename fnt.ttf should output serialized info for that specific table.


- pdf side:


+simple fonts:
subsetting ttf font that is using only simple fonts...  we can probably
generate a font with 255 glyphs that are only those needed, and maybe empty ones in between,
if needed
subset name: EOODIA+Poetica : arbitrary 6 uppercase letters + postscript name
or just take the 0-255 entries... which should be pretty easy, order unchanged.

+composite fonts:
create a CIDToGIDMap: stream/name
default is identity (1/1),
If the value is a stream, the bytes in the stream
shall contain the mapping from CIDs to glyph indices: the glyph index
for a particular CID value c shall be a 2-byte value stored in bytes
2 × c and 2 × c + 1, where the first byte shall be the high-order byte.
If the value of CIDToGIDMap is a name, it shall be Identity,
indicating that the mapping between CIDs and glyph indices is the
identity mapping. Default value: Identity.

The generation of the CIDToGIDMap assumes that the original CID:GID mapping was 1:1
and the deviation is due to changes in the glyph indices, so each entry will
be CID:newGID value
0:1:2:3:4:5:6...





