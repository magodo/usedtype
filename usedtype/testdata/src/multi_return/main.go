package main

import (
	"sdk"
)

func main() {
	req := sdk.Req{}
	req.Properties, _ = buildProp()
	client := sdk.BuildClient()
	client.CreateOrUpdate(req)
}

func buildProp() (*sdk.Properties, bool) {
	prop := &sdk.Properties{}
	prop.Prop1 = 1
	return prop, true
}
