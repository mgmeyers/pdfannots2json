package cpd

//UTF-32BE

//первые 2 байта практически всегда равны 0
func matchUTF32be(d []byte, tbl *cpTable) MatchRes {
	zerroCounts := 0
	for i := 0; i < len(d)-4; i += 4 {
		if (int(d[i]) + int(d[i+1])) == 0 {
			zerroCounts++
		}
	}
	if zerroCounts*2 < len(d)/4 { //количество байтов в файле UTF-32 со значением 0 должно быть больше половины
		return MatchRes{0, 0}
	}
	return MatchRes{zerroCounts, 0}

}
