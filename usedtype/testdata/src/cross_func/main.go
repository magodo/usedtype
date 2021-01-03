package main

import (
	"sdk"
)

func main() {
	model := sdk.ModelA{String: "x"}
	_ = model
}

func buildProp() sdk.Property {
	return sdk.Property{Int: 1}
}
