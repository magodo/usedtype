package main

import (
	"sdk"
)

func main() {
	req := sdk.Req{}
	useMeta(req.Metadata)
	_ = req
}

func useMeta(meta sdk.Metadata) {
	_ = meta.Scope
}
