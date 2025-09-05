package main

import (
	"fmt"

	"github.com/KennethanCeyer/ptyx"
)

func main() {
	fmt.Println("--- 16 Colors (Foreground) ---")
	for i := 30; i <= 37; i++ {
		bright := i + 60
		fmt.Printf("%s%2d: Normal%s  ", ptyx.CSI(fmt.Sprintf("%dm", i)), i, ptyx.CSI("0m"))
		fmt.Printf("%s%2d: Bright%s\n", ptyx.CSI(fmt.Sprintf("%dm", bright)), bright, ptyx.CSI("0m"))
	}
	fmt.Println()

	fmt.Println("--- 16 Colors (Background) ---")
	for i := 40; i <= 47; i++ {
		bright := i + 60
		fmt.Printf("%s%3d: Normal%s  ", ptyx.CSI(fmt.Sprintf("%dm", i)), i, ptyx.CSI("0m"))
		fmt.Printf("%s%3d: Bright%s\n", ptyx.CSI(fmt.Sprintf("%dm", bright)), bright, ptyx.CSI("0m"))
	}
	fmt.Println()

	fmt.Println("--- 256-Color Palette ---")
	for i := 0; i < 256; i++ {
		fmt.Printf("%s %3d %s", ptyx.CSI(fmt.Sprintf("48;5;%dm", i)), i, ptyx.CSI("0m"))
		if (i+1)%16 == 0 {
			fmt.Println()
		}
	}
	fmt.Println()
}
