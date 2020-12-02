package main

import (
	"sdk"
)

func main() {
	req := sdk.Req{}
	req.Properties = buildProp()
	_ = req
}

func foo() {
	prop := buildProp()
	prop.Prop1 = 1
	return
}

func buildProp() *sdk.Properties {
	return &sdk.Properties{}
}
