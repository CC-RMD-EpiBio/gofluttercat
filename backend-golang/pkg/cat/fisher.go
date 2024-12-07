package cat

import (
	"fmt"
	"math"
	"math/rand/v2"

	math2 "github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/math"

	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/models"
	"github.com/mederrata/ndvek"
)

type FisherSelector struct {
	Temperature float64
}

type BayesianFisherSelector struct {
	Temperature float64
}

func sample(weights map[string]float64) string {
	r := rand.Float64()
	var cumulative float64 = 0
	var lastKey string
	var Z float64
	for _, w := range weights {
		Z += w
	}
	for label, prob := range weights {
		cumulative += prob / Z
		lastKey = label
		if r < cumulative {
			return label
		}
	}
	return lastKey
}

func (fs FisherSelector) Criterion(bs *models.BayesianScorer) map[string]float64 {
	abilities, err := ndvek.NewNdArray([]int{1}, []float64{bs.Running.Mean()})
	if err != nil {
		panic(err)
	}
	fish := bs.Model.FisherInformation(abilities)

	crit := make(map[string]float64, 0)
	for label, value := range fish {
		crit[label] = value.Data[0]
	}

	return crit
}

func (fs FisherSelector) NextItem(bs *models.BayesianScorer) *models.Item {

	crit := fs.Criterion(bs)

	var Z float64 = 0
	T := fs.Temperature

	probs := make(map[string]float64, 0)
	for key, value := range crit {
		if hasResponse(key, bs.Answered) {
			continue
		}
		E := value
		probs[key] = E
	}

	if T == 0 {
		var selected string
		var maxval float64
		for key, value := range probs {
			if value > maxval {
				selected = key
				maxval = value
			}
		}
		return getItemByName(selected, bs.Model.GetItems())
	}

	for key, value := range probs {
		probs[key] = math.Exp(value / T)
		Z += probs[key]
	}
	for key, _ := range probs {
		probs[key] /= Z
	}
	selected := sample(probs)
	fmt.Printf("selected: %v\n", selected)
	return getItemByName(selected, bs.Model.GetItems())
}

func itemIn(itemName string, itemList []*models.Item) bool {
	for _, itm := range itemList {
		if itm.Name == itemName {
			return true
		}
	}
	return false
}

func hasResponse(itemName string, responses []*models.Response) bool {
	for _, r := range responses {
		if r.Item.Name == itemName {
			return true
		}
	}
	return false
}

func getItemByName(itemName string, itemList []*models.Item) *models.Item {
	for _, itm := range itemList {
		if itm.Name == itemName {
			return itm
		}
	}
	return nil
}

func (fs BayesianFisherSelector) Criterion(bs *models.BayesianScorer) map[string]float64 {
	abilities, err := ndvek.NewNdArray([]int{len(bs.AbilityGridPts)}, bs.AbilityGridPts)
	if err != nil {
		panic(err)
	}
	fish := bs.Model.FisherInformation(abilities)
	density := bs.Running.Density()
	fishB := make(map[string]float64, 0)

	for key, val := range fish {
		if hasResponse(key, bs.Answered) {
			continue
		}
		fishB[key] = math2.Trapz2(density, val.Data)
	}
	return fishB
}

func (fs BayesianFisherSelector) NextItem(bs *models.BayesianScorer) *models.Item {
	fishB := fs.Criterion(bs)
	var Z float64 = 0
	T := fs.Temperature
	if T == 0 {
		var selected string
		var maxval float64
		for key, value := range fishB {
			if value > maxval {
				selected = key
				maxval = value
			}
		}
		return getItemByName(selected, bs.Model.GetItems())
	}
	probs := make(map[string]float64, 0)
	for key, value := range fishB {
		probs[key] = math.Exp(value/T) / Z
		Z += probs[key]
	}
	for key, _ := range probs {
		probs[key] /= Z
	}
	selected := sample(probs)
	fmt.Printf("selected: %v\n", selected)
	return getItemByName(selected, bs.Model.GetItems())
}
