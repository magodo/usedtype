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
