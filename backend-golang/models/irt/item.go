package irt

type Item struct {
	Name       string `yaml:"item"`
	Question   string `yaml:"question"`
	DomainName string `yaml:"domain"`

	Responses     map[string]Response     `yaml:"responses"`
	ScaleLoadings map[string]ScaleLoading `yaml:"scales"`
}

type Response struct {
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
