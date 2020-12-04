package sdk

type ModelA struct {
	String                   string      `json:"string"`
	Property                 Property    `json:"property"`
	PointerOfProperty        *Property   `json:"pointer_of_property"`
	ArrayOfString            []string    `json:"array_of_string"`
	PointerOfArrayOfString   *[]string   `json:"pointer_of_array_of_string"`
	ArrayOfProperty          []Property  `json:"array_of_property"`
	PointerOfArrayOfProperty *[]Property `json:"pointer_of_array_of_property"`
	ArrayOfPointerOfProperty []*Property `json:"array_of_pointer_of_property"`
}

type Property struct {
	Int int `json:"int"`
}

type Animal interface {
	isAnimal()
}

type Dog struct {
	RunSpeed int `json:"run_speed"`
}

func (d Dog) isAnimal() {}

type Fish struct {
	SwimSpeed int `json:"swim_speed"`
}

func (f Fish) isAnimal() {}

type OneAnimal struct {
	Name   string `json:"name"`
	Animal Animal `json:"animal"`
}

type Zoo struct {
	Animals []Animal `json:"animals"`
}
