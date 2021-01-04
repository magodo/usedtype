package main

import (
	"sdk"
)

func main() {
	prop := buildProp()

	req := sdk.ModelA{
		String:   "foo",
		Property: prop,
	}

	req.ArrOfPropWrapper = buildPropWrapperArr(prop)

}

func buildProp() sdk.Property {
	return sdk.Property{Int: 1}
}

func buildPropWrapperArr(prop sdk.Property) []sdk.PropWrapper {
	out := []sdk.PropWrapper{}
	for i := 0; i < 3; i++ {
		pw := sdk.PropWrapper{Prop: prop}
		out = append(out, pw)
	}
	return out
}
