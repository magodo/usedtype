package main

import (
	"sdk"
)

func main() {
	req := sdk.Req{
		Properties: &sdk.Properties{},
	}
	mutateProp(req.Properties)
	client := sdk.BuildClient()
	client.CreateOrUpdate(req)
}

func mutateProp(prop *sdk.Properties) {
	prop.Prop1 = 1
}
