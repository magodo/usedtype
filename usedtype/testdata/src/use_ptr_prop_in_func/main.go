package main

import (
	"sdk"
)

func main() {
	req := sdk.Req{}
	useProp(req.Properties)
	_ = req
}

func useProp(prop *sdk.Properties) {
	_ = prop.Prop1
}
