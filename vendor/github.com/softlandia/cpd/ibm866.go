package cpd

//unit for ibm866

// for CP866 calculate only count of letter from table 'tbl'
func match866(data []byte, tbl *cpTable) MatchRes {
	for i := range data {
		j := tbl.index(rune(data[i])) //return 0 if rune data[i] not found
		(*tbl)[j].count++
	}
	return MatchRes{tbl.founded(), 0}
}

const (
	cp866StartUpperChar  = 0x80
	cp866StopUpperChar   = 0x9F
	cp866BeginLowerChar1 = 0xA0
	cp866StopLowerChar1  = 0xAF
	cp866BeginLowerChar2 = 0xE0
	cp866StopLowerChar2  = 0xEF
)

func isUpper866(r byte) bool {
	return (r >= cp866StartUpperChar) && (r <= cp866StopUpperChar)
}

func isLower866(r byte) bool {
	return ((r >= cp866BeginLowerChar1) && (r <= cp866StopLowerChar1)) ||
		((r >= cp866BeginLowerChar2) && (r <= cp866StopLowerChar2))
}

func is866(r byte) bool {
	return isUpper866(r) || isLower866(r)
}
