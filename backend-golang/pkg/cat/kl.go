package cat

import (
	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/models"
	"github.com/mederrata/ndvek"
)

type KLSelector struct {
	Temperature float64
}

func (ks KLSelector) NextItem(bs *models.BayesianScorer) *models.Item {
	abilities, err := ndvek.NewNdArray([]int{len(bs.AbilityGridPts)}, bs.AbilityGridPts)
	if err != nil {
		panic(err)
	}
	density := bs.Running.Density()
	probs := bs.Model.Prob(abilities)
	admissable := make([]*models.Item, 0)
	for _, itm := range bs.Model.GetItems() {
		if hasResponse(itm.Name, bs.Answered) {
			continue
		}
		admissable = append(admissable, itm)
	}

	lpObs := bs.Running.Energy

	// compute pi_infty

	return nil
}
