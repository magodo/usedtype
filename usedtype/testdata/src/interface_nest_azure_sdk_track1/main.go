package main

import (
	"os"
	"sdk"
)

func main() {
	var mid sdk.BasicMiddle
	a := sdk.A{Name: "A"}
	_ = a
	kind := os.Args[1]
	switch kind {
	case "b":
		mid = sdk.B{Name: "B"}
	case "c":
		mid = sdk.C{Name: "C"}
	}
	_ = mid
}
