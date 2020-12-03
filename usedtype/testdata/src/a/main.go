package main

import (
	"sdk"
)

func main() {
	req := sdk.ModelA{
		String: "foo",
	}

	prop := buildProp()
	req.Property = prop
	req.PointerOfProperty = &prop

	sarr := []string{"a"}
	req.ArrayOfString = sarr
	req.PointerOfArrayOfString = &sarr

	parr := []sdk.Property{prop}
	req.ArrayOfProperty = parr
	req.PointerOfArrayOfProperty = &parr

	pparr := []*sdk.Property{&prop}
	req.ArrayOfPointerOfProperty = pparr
	_ = req
}

func buildProp() sdk.Property {
	return sdk.Property{Int: 1}
}
