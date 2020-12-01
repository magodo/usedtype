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
	return sdk.Metadata{AdditionalInfo: buildNestedProp()}
}

func buildNestedProp() sdk.AdditionalInfo {
	meta := sdk.Metadata{}
	meta.AdditionalInfo.Foo = "abc"
	return meta.AdditionalInfo
}
