package customfields

import (
	"strconv"
	"strings"
	"unicode"
)

func separateStr(str string) (uint64, string, error) {
	str = strings.ToLower(str)
	pos := 0
	for ; pos < len(str); pos++ {
		if !unicode.IsDigit(rune(str[pos])) {
			break
		}
	}
	a, err := strconv.ParseUint(str[:pos], 10, 64)
	if err != nil {
		return 0, "", err
	}
	return a, str[pos:], nil
}
