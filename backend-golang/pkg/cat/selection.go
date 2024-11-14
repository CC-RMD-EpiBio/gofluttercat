package cat

import (
	"fmt"
	"math/rand/v2"

	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/models"
	"github.com/mederrata/ndvek"
)

type KLSelector struct {
	Temperature float64
}

type FisherSelector struct {
	Temperature float64
}

type VarianceSelector struct {
	Temperature float64
}

func sample(map[string]float64) {
	r := rand.Float64()
}

func (ks KLSelector) NextItem(bs *models.BayesianScorer) *models.Item {
	abilities, err := ndvek.NewNdArray([]int{1}, []float64{bs.Running.Mean()})
	if err != nil {
		panic(err)
	}
	fish := bs.Model.FisherInformation(abilities)
	fmt.Printf("fish: %v\n", fish)
	return nil
}

func (vs VarianceSelector) NextItem(s *models.BayesianScorer) *models.Item {
	return nil
}

func (fs FisherSelector) NextItem(bs *models.BayesianScorer) *models.Item {
	abilities, err := ndvek.NewNdArray([]int{1}, []float64{bs.Running.Mean()})
	if err != nil {
		panic(err)
	}
	fish := bs.Model.FisherInformation(abilities)
	fmt.Printf("fish: %v\n", fish)
	return nil
}
