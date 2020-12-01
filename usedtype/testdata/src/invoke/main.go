package main

type IFoo interface {
	Hello(name string)
}

type Foo struct{}

func (f Foo) Hello(name string) {
	return
}

func main() {
	var i IFoo = Foo{}
	i.Hello("world")
}
