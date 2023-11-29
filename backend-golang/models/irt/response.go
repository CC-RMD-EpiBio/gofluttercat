package irt

type Response struct {
	Item       *Item
	Choice     *ItemChoice
	Model      *IRTModel
	Respondent *Respondent
}

func (r Response) Likelihood(ic *ItemChoice) float64 {
	like := (*r.Model).Prob(*r.Item, *r.Respondent)
	return like
}
