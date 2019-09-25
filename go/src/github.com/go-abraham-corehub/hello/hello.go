package main

import (
	"fmt"

	"github.com/abraham-corehub/go/stringutil"
)

func main() {
	fmt.Println(stringutil.Reverse("!dlroW ,olleH"))
	x := byte(0b00001011)
	y := x>>4&1 == 1
	fmt.Println(y)
}
