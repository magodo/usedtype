package sdk

type ModelA struct {
	String                   string        `json:"string"`
	Property                 Property      `json:"property"`
	PointerOfProperty        *Property     `json:"pointer_of_property"`
	ArrayOfString            []string      `json:"array_of_string"`
	PointerOfArrayOfString   *[]string     `json:"pointer_of_array_of_string"`
	ArrayOfProperty          []Property    `json:"array_of_property"`
	PointerOfArrayOfProperty *[]Property   `json:"pointer_of_array_of_property"`
	ArrayOfPointerOfProperty []*Property   `json:"array_of_pointer_of_property"`
	PropWrapper              PropWrapper   `json:"prop_wrapper"`
	ArrOfPropWrapper         []PropWrapper `json:"array_of_prop_wrapper"`
}

type PropWrapper struct {
	Prop Property `json:"prop"`
}

type Property struct {
	Int int `json:"int"`
}

type Animal interface {
	isAnimal()
}

type Dog struct {
	Name     string `json:"name"`
	RunSpeed int    `json:"run_speed"`
}

func (d Dog) isAnimal() {}

type Fish struct {
	Name      string `json:"name"`
	SwimSpeed int    `json:"swim_speed"`
}

func (f Fish) isAnimal() {}

type Bird struct {
	Name     string `json:"name"`
	FlySpeed int    `json:"fly_speed"`
}

func (b Bird) isAnimal() {}

type AnimalFamily interface {
	IsFamily()
}

type DogFamily struct {
	Animals []Animal `json:"animals"`
}

func (f DogFamily) IsFamily() {}

type FishFamily struct {
	Animals []Animal `json:"animals"`
}

func (f FishFamily) IsFamily() {}

type BirdFamily struct {
	Animals []Animal `json:"animals"`
}

func (f BirdFamily) IsFamily() {}

type Zoo struct {
	AnimalFamilies []AnimalFamily `json:"animal_family"`
}
