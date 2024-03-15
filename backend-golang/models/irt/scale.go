package irt

type Scale struct {
	Loc     float64  `yaml:"loc"`
	Scale   float64  `yaml:"scale"`
	Name    string   `yaml:"name"`
	Version string   `yaml:"version"`
	Diff    Diff     `yaml:"diff"`
	Tags    []string `yaml:"tags"`
}

type Diff struct {
	Excluded map[string]string `yaml:"excluded"`
}

type ScaleInfo struct {
	ScaleLoadings map[string]Scale
}
