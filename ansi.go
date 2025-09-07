package ptyx

import (
	"fmt"
	"strconv"
	"strings"
)

func CSI(seq string) string { return "\x1b[" + seq }
func CUP(row, col int) string { return CSI(fmt.Sprintf("%d;%dH", row, col)) }
func SGR(codes ...int) string {
	if len(codes) == 0 {
		return CSI("0m")
	}
	s := make([]string, len(codes))
	for i, c := range codes {
		s[i] = strconv.Itoa(c)
	}
	return CSI(strings.Join(s, ";") + "m")
}
