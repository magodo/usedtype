package main

import (
	"sdk"
)

func main() {
	req := sdk.Req{}
	prop := sdk.Properties{}
	prop.Prop1 = 1
	req.Properties = &prop
	prop.Prop2 = "still used"
	client := sdk.BuildClient()
	client.CreateOrUpdate(req)
}
