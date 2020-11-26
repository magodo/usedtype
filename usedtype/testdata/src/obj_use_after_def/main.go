package main

import (
	"sdk"
)

func main() {
	req := sdk.Req{}
	meta := sdk.Metadata{}
	meta.Scope = "scope"
	req.Metadata = meta
	meta.Version = "not used"
	client := sdk.BuildClient()
	client.CreateOrUpdate(req)
}
