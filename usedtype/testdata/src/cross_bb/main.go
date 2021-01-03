package main

import (
	"sdk"
)

func main() {
	req := sdk.ModelA{}

	props := make([]sdk.Property, 3)
	for i := 0; i < 3; i++ {
		props[i] = sdk.Property{Int: i}
	}
	req.ArrayOfProperty = props
}
