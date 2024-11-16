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
	for label, prob := range weights {
		cumulative += prob
		lastKey = label
		if r < cumulative {
			return label
		}
	}
	return lastKey
}

func (fs FisherSelector) NextItem(bs *models.BayesianScorer) *models.Item {
	abilities, err := ndvek.NewNdArray([]int{1}, []float64{bs.Running.Mean()})
	if err != nil {
		panic(err)
	}
	fish := bs.Model.FisherInformation(abilities)

	var Z float64 = 0
	T := fs.Temperature
	if T < 1e-5 {
		T = 1e-5
	}

	probs := make(map[string]float64, 0)
	for key, value := range fish {
		if hasResponse(key, bs.Answered) {
			continue
		}
		E := value.Data[0]
		probs[key] = math.Exp(E / T)
		Z += probs[key]
	}
	for key, value := range probs {
		probs[key] = value / Z
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

func (fs BayesianFisherSelector) NextItem(bs *models.BayesianScorer) *models.Item {

	abilities, err := ndvek.NewNdArray([]int{len(bs.AbilityGridPts)}, bs.AbilityGridPts)
	if err != nil {
		panic(err)
	}
	fish := bs.Model.FisherInformation(abilities)
	density := bs.Running.Density()
	probs := make(map[string]float64, 0)
	var Z float64 = 0
	T := fs.Temperature
	if T < 1e-5 {
		T = 1e-5
	}
	for key, val := range fish {
		if hasResponse(key, bs.Answered) {
			continue
		}
		probs[key] = math.Exp(math2.Trapz2(density, val.Data)) / T
		Z += probs[key]
	}

	for key, value := range probs {
		probs[key] = value / Z
	}
	selected := sample(probs)
	fmt.Printf("selected: %v\n", selected)
	return getItemByName(selected, bs.Model.GetItems())
}
