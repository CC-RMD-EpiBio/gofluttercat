package irt

type Item struct {
	Name          string                  `yaml:"item"`
	Question      string                  `yaml:"question"`
	Choices       map[string]Choice       `yaml:"responses"`
	ScaleLoadings map[string]ScaleLoading `yaml:"scales"`
}

type Choice struct {
	Text  string `yaml:"text"`
	Value uint   `yaml:"value"`
}

type Calibration struct {
	Difficulties   []float64 `yaml:"difficulties"`
	Discrimination float64   `yaml:"discrimination"`
}

type ScaleLoading struct {
	Name        string
	Calibration Calibration
}
