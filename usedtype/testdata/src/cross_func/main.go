package main

import (
	"sdk"
)

func main() {
	model := sdk.ModelA{String: "x", Property: sdk.Property{}}
	_ = model
}

func buildProp() sdk.Property {
	return sdk.Property{Int: 1}
}
