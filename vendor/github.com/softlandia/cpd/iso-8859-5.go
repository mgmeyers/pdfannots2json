package cpd

//unit for ISO-8859-5

func matchISO88595(d []byte, tbl *cpTable) MatchRes {
	for i := 0; i < len(d); i++ {
		if isISO88595(d[i]) {
			upper := lu88595(d[i])
			j := tbl.index(rune(d[i]))
			(*tbl)[j].count++
			for i++; (i < len(d)) && isISO88595(d[i]); i++ {
				if upper >= lu88595(d[i]) {
					j = tbl.index(rune(d[i]))
					(*tbl)[j].count++
				}
			}
		}
	}
	return MatchRes{tbl.founded(), 0}
}

const (
	cpISO88595BeginUpperChar = 0xB0
	cpISO88595StopUpperChar  = 0xCF
	cpISO88595BeginLowerChar = 0xD0
	cpISO88595StopLowerChar  = 0xEF
)

func lu88595(r byte) (res int) {
	if isUpperISO88595(r) {
		res = 1
	}
	return
}

func isUpperISO88595(r byte) bool {
	return (r >= cpISO88595BeginUpperChar) && (r <= cpISO88595StopUpperChar)
}

func isLowerISO88595(r byte) bool {
	return (r >= cpISO88595BeginLowerChar) && (r <= cpISO88595StopLowerChar)
}

func isISO88595(r byte) bool {
	return isUpperISO88595(r) || isLowerISO88595(r)
}
