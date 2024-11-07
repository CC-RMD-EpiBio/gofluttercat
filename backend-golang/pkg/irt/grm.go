package irt

import (
	"fmt"
	"math"

	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/models"
	ndvek "github.com/mederrata/ndvek"
)

// GradedResponseModel is a univariate
type GradedResponseModel struct {
	Scale           models.Scale
	Items           []*models.Item
	Discriminations ndvek.NdArray
	Difficulties    ndvek.NdArray
}

func findIndex(arr []int, element int) (int, error) {
	for i, v := range arr {
		if v == element {
			return i, nil
		}
	}
	return -1, fmt.Errorf("element %d not found in array", element)
}

func NewGRM(items []*models.Item, scale models.Scale) GradedResponseModel {
	model := GradedResponseModel{
		Items: items,
		Scale: scale,
	}
	nItems := len(items)
	discriminations := make([]float64, 0)
	scaleName := scale.Name
	for _, item := range items {
		cal, ok := item.ScaleLoadings[scaleName]
		var discrim float64
		if !ok {
			discrim = 0.0
		} else {
			discrim = cal.Discrimination
		}
		discriminations = append(discriminations, discrim)
		fmt.Printf("item: %v\n", item)
	}
	discriminations_, err := ndvek.NewNdArray(
		[]int{nItems}, discriminations)
	if err != nil {
		panic(err)
	}
	model.Discriminations = *discriminations_

	return model
}

func (grm GradedResponseModel) LogLikelihood(abilities *ndvek.NdArray, resp *models.SessionResponses) *ndvek.NdArray {
	// Shape of abilities is n_abilities x n_scale

	prob := grm.Prob(abilities)
	for _, r := range resp.Responses {
		ndx, err := findIndex(r.Item.ScoredValues, r.Value)
		if err != nil {
			continue
		}
		fmt.Printf("prob: %v\n", prob)
		fmt.Printf("ndx: %v\n", ndx)
	}
	return nil
}

func sigmoid(x float64) float64 {
	exp := math.Exp(x)
	return exp / (1 + exp)
}

func (grm GradedResponseModel) Prob(abilities *ndvek.NdArray) map[string]*ndvek.NdArray {

	nAbilities := abilities.Shape()[0]
	abilities = abilities.InsertAxis(1)
	probs := map[string]*ndvek.NdArray{}
	for _, itm := range grm.Items {
		calibration, ok := itm.ScaleLoadings[grm.Scale.Name]
		if !ok {
			return probs
		}

		plogits, err := ndvek.NewNdArray([]int{1, len(calibration.Difficulties)}, calibration.Difficulties)
		if err != nil {
			panic(err)
		}
		plogits, err = ndvek.Subtract(plogits, abilities)
		plogits = plogits.MulScalar(calibration.Discrimination)
		if err != nil {
			panic(err)
		}
		err = plogits.ApplyHadamardOp(sigmoid)
		if err != nil {
			panic(err)
		}

		nCats := len(calibration.Difficulties) + 1
		data := []float64{}
		for j := 0; j < nAbilities; j++ {
			for i := 0; i < nCats-1; i++ {
				if i == 0 {
					value, err := plogits.Get([]int{j, 0})
					if err != nil {
						panic(err)
					}
					data = append(data, value)
					continue
				}
				value1, err := plogits.Get([]int{j, i})
				if err != nil {
					panic(err)
				}
				value2, err := plogits.Get([]int{j, i - 1})
				if err != nil {
					panic(err)
				}
				data = append(data, value1-value2)
			}
			value, err := plogits.Get([]int{j, nCats - 2})
			if err != nil {
				panic(err)
			}
			data = append(data, 1-value)
		}
		probs[itm.Name], err = ndvek.NewNdArray([]int{nAbilities, nCats}, data)
		if err != nil {
			panic(err)
		}

	}
	return probs
}
