package main

import (
	"a/bar"
	"a/foo"
)

func main() {
	var foo foo.Foo
	var b bar.Bar
	b.Init("name")
	foo.Bar = b
}
