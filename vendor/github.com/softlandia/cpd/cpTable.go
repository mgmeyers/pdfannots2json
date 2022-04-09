package cpd

import "sort"

//codePageTable

// return index of rune in code page table
// return 0 if rune not in code page table
func (t *cpTable) index(r rune) int {
	for j, e := range *t {
		if r == e.code {
			return j
		}
	}
	return 0
}

// founded - calculates total number of matching
func (t *cpTable) founded() (res int) {
	//0 элемент исключён, он не содержит количество найденных букв
	for i := 1; i < len(t); i++ {
		res += t[i].count
	}
	return
}

func (t *cpTable) clear() {
	for i := 0; i < len(t); i++ {
		t[i].count = 0
	}
}

func (t *cpTable) sort() *cpTable {
	sort.Slice(&t, func(i, j int) bool { return i < j })
	return t
}
