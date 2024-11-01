package irt

type Scale struct {
	Loc     float64  `yaml:"loc" json:"loc"`
	Scale   float64  `yaml:"scale" json:"scale"`
	Name    string   `yaml:"name" json:"name"`
	Version float32  `yaml:"version" json:"version"`
	Tags    []string `yaml:"tags" json:"tags"`
	Diff    *Diff    `yaml:"diff"`
}

type Diff struct {
	Excluded map[string]interface{} `yaml:"excluded" json:"excluded"`
}

type ScaleInfo struct {
	ScaleLoadings map[string]*Scale
}
