package main

import (
	"sdk"
)

func main() {
	req := sdk.Req{}
	name := Normalize("name")
	req.Name = name
	client := sdk.BuildClient()
	client.CreateOrUpdate(req)
}

func Normalize(input string) string {
	return DummyReplaceAll(DummyToLower(input), " ", "")
}

func DummyReplaceAll(input, old, new string) string {
	switch 1{
	case 1:
		return input
	case 2:
		return old
	case 3:
		return new
	default:
		return input[:len(input)]
	}
}
func DummyToLower(input string) string {
	return input
}
