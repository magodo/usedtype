package main

import (
	"sdk"
)

func main() {
	req := sdk.Req{}
	req.Properties = buildProp()
	_ = req
}

func buildProp() *sdk.Properties {
	return &sdk.Properties{}
}
