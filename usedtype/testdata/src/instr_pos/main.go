package main

type Foo struct {
	I int
	*Bar
}

type Bar struct {
	J int
}

func (f Foo) F() {}

type IFoo interface{ F() }

func main() {
	// FieldAddr regular
	f1 := Foo{}
	f1.I = 1

	// FieldAddr composite literal
	_ = Foo{I: 1}

	// FieldAddr composite literal
	_ = Foo{
		I: 1,
	}

	// MakeInterface
	var iF IFoo = Foo{}
	_ = iF

	// Field: implicitly access embedded member's (pointer type) structure
	f2 := Foo{}
	f2.J = 1

}
