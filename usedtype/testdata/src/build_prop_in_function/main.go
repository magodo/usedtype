package main

import (
	"sdk"
)

func main() {
	req := sdk.Req{}
	req.Metadata = buildProp()
	_ = req
}

func buildProp() sdk.Metadata {
	return sdk.Metadata{}
}
