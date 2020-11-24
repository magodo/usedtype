package sdk

type Req struct {
	Name        string `json:"name,omitempty"`
	*Properties `json:"properties,omitempty"`
}

type Properties struct {
	Prop1 int    `json:"prop1"`
	Prop2 string `json:"prop2"`
}
