package utils

import "strings"

func PadRightToSameLength(strs ...*string) {
	if len(strs) < 2 {
		return
	}

	maxLen := 0
	for _, sptr := range strs {
		if len(*sptr) > maxLen {
			maxLen = len(*sptr)
		}
	}

	for _, sptr := range strs {
		if len(*sptr) < maxLen {
			*sptr = *sptr + strings.Repeat(" ", maxLen-len(*sptr))
		}
	}
}
