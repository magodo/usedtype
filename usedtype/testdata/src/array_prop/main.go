package main

import (
	"sdk"
)

func main() {
	req := sdk.Req{}
	regions := []sdk.Region{{State: "unknown"}}
	req.Regions = &regions
	_ = req
}
