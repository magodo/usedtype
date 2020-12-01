package sdk

type Req struct {
	Name        string `json:"name,omitempty"`
	*Properties `json:"properties,omitempty"`
	Metadata    Metadata `json:"metadata,omitempty"`
	Regions *[]Region `json:"regions,omitempty"`
}

type AdditionalInfo struct {
	Foo string `json:"foo"`
}

type Metadata struct {
	Scope          string         `json:"scope"`
	Version        string         `json:"version"`
	AdditionalInfo AdditionalInfo `json:"additional_info"`
}

type Properties struct {
	Prop1 int    `json:"prop1"`
	Prop2 string `json:"prop2"`
}

type Region struct {
	State string `json:"state"`
}

