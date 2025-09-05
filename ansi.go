package ptyx

import "fmt"

func CSI(seq string) string { return "\x1b[" + seq }
func CUP(row, col int) string { return CSI(fmt.Sprintf("%d;%dH", row, col)) }
func SGR(codes ...int) string {
	if len(codes) == 0 { return CSI("0m") }
	s := ""
	for i, c := range codes {
		if i > 0 { s += ";" }
		s += fmt.Sprintf("%d", c)
	}
	return CSI(s + "m")
}
