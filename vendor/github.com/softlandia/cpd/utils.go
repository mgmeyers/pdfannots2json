package cpd

import "sort"

// Max - return max of two int
func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

// SortBytes - return sorted slice of bytes
func sortBytes(b []byte) []byte {
	sort.Slice(b, func(i, j int) bool { return b[i] < b[j] })
	return b
}
