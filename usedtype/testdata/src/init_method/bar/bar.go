package bar

type Bar struct {
	Name string
}

func (bar *Bar) Init(name string) {
	obar := Bar{Name: name}
	*bar = obar
}
