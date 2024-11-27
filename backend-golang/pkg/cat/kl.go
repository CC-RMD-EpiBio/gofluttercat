package cat

import (
	"fmt"
	"math"

	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/models"
	"github.com/mederrata/ndvek"
	"github.com/viterin/vek"
)

type KLSelector struct {
	Temperature float64
}

func (ks KLSelector) NextItem(bs *models.BayesianScorer) *models.Item {
	abilities, err := ndvek.NewNdArray([]int{len(bs.AbilityGridPts)}, bs.AbilityGridPts)
	if err != nil {
		panic(err)
	}
	nAbilities := abilities.Shape()[0]

	density := bs.Running.Density()
	probs := bs.Model.Prob(abilities)
	admissable := make([]*models.Item, 0)
	for _, itm := range bs.Model.GetItems() {
		if hasResponse(itm.Name, bs.Answered) {
			continue
		}
		admissable = append(admissable, itm)
	}

	lpInfy := bs.Running.Energy
	for a, itm := range admissable {
		pr := probs[itm.Name]
		K := pr.Shape()[1]
		for j := 0; j < nAbilities; j++ {
			for k := 0; k < K; k++ {
				val, err := pr.Get([]int{j, k})
				if err != nil {
					panic(err)
				}
				lpInfy[a] += val * math.Log(val)
			}
		}
	}
	lpInfy = vek.SubNumber(lpInfy, vek.Max(lpInfy))
	// compute log_pi_infty for plugin estimator

	fmt.Printf("density: %v\n", density)
	fmt.Printf("probs: %v\n", probs)
	fmt.Printf("lpObs: %v\n", lpInfy)

	return nil
}

type McKlSelector struct {
	Temperature float64
	NumSamples  int
}

func NewMcKlSelector(temperature float64, nsamples int) McKlSelector {
	return McKlSelector{
		Temperature: temperature, NumSamples: nsamples}
}

func (ks McKlSelector) NextItem(bs *models.BayesianScorer) *models.Item {
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
	fmt.Printf("density: %v\n", density)
	fmt.Printf("probs: %v\n", probs)
	fmt.Printf("lpObs: %v\n", lpObs)
	// sample
	return nil
}
