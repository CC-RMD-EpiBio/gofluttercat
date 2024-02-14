package irt

type Scale struct {
	Domain  string   `yaml:"domain"`
	Loc     string   `yaml:"loc"`
	Scale   string   `yaml:"scale"`
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
