package sdk

type Req struct {
	Name        string `json:"name,omitempty"`
	*Properties `json:"properties,omitempty"`
	Metadata    Metadata `json:"metadata,omitempty"`
}

type Metadata struct {
	Scope   string `json:"scope"`
	Version string `json:"version"`
}

type Properties struct {
	Prop1 int    `json:"prop1"`
	Prop2 string `json:"prop2"`
}
