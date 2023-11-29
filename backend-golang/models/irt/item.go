package irt

type Item struct {
	Name    string
	Choices []ItemChoice
}

type ItemChoice struct {
	ID uint
}

type Calibration struct {
}

func (i Item) Prob(resp *Respondent, m *IRTModel) map[*ItemChoice]float64 {
	return nil
}
