package main

import (
	"sdk"
	"strings"
)

func main() {
	req := sdk.Req{}
	name := Normalize("name")
	req.Name = name
	client := sdk.BuildClient()
	client.CreateOrUpdate(req)
}

func Normalize(input string) string {
	return strings.ReplaceAll(strings.ToLower(input), " ", "")
}
