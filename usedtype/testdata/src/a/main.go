package main

import (
	"sdk"
)

func main() {
	req := sdk.Req{
		Name: "name",
	}
	req.Properties.Prop1 = 1
	req.Properties = buildProp(true)
	_ = req
}

func buildProp(b bool) *sdk.Properties {
	var prop *sdk.Properties
	if b {
		prop = &sdk.Properties{}
		prop.Prop1 = 1
	} else {
		prop = &sdk.Properties{}
		prop.Prop2 = "foo"
	}
	return prop
}
