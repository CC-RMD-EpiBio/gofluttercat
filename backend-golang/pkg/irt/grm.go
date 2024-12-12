package irt

import (
	"fmt"
	"math"

	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/models"
	math2 "github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/math"

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
	}
	discriminations_, err := ndvek.NewNdArray(
		[]int{nItems}, discriminations)
	if err != nil {
		panic(err)
	}
	model.Discriminations = *discriminations_

	return model
}

func (grm GradedResponseModel) FisherInformation(abilities *ndvek.NdArray) map[string]*ndvek.NdArray {
	scaleName := grm.Scale.Name
	nAbilities := abilities.Shape()[0]
	abilities = abilities.InsertAxis(1)
	fish := make(map[string]*ndvek.NdArray, 0)
	for _, itm := range grm.Items {
		cal, ok := itm.ScaleLoadings[scaleName]
		if !ok {
			continue
		}
		plogits, err := ndvek.NewNdArray([]int{1, len(cal.Difficulties)}, cal.Difficulties)
		if err != nil {
			panic(err)
		}
		plogits, err = ndvek.Subtract(plogits, abilities)
		plogits = plogits.MulScalar(cal.Discrimination)
		if err != nil {
			panic(err)
		}
		err = plogits.ApplyHadamardOp(math2.Sigmoid)
		if err != nil {
			panic(err)
		}
		plogits2 := plogits.AddScalar(-1).MulScalar(-1)
		plogits2, err = ndvek.Multiply(plogits2, plogits)
		if err != nil {
			panic(err)
		}
		plogits2 = plogits2.MulScalar(cal.Discrimination * cal.Discrimination)

		nCats := len(cal.Difficulties) + 1
		data := []float64{}
		for j := 0; j < nAbilities; j++ {
			var agg float64 = 0
			for i := 0; i < nCats-1; i++ {
				if i == 0 {
					p, err := plogits.Get([]int{j, 0})
					if err != nil {
						panic(err)
					}
					f, err := plogits2.Get([]int{j, 0})
					agg += p * f
					if err != nil {
						panic(err)
					}

					continue
				}
				value1, err := plogits.Get([]int{j, i})
				if err != nil {
					panic(err)
				}
				f1, err := plogits2.Get([]int{j, i})
				if err != nil {
					panic(err)
				}
				value2, err := plogits.Get([]int{j, i - 1})
				if err != nil {
					panic(err)
				}
				f2, err := plogits2.Get([]int{j, i - 1})
				if err != nil {
					panic(err)
				}
				p := value1 - value2
				f := f1 + f2
				agg += p * f

			}
			p, err := plogits.Get([]int{j, nCats - 2})
			if err != nil {
				panic(err)
			}
			p = 1 - p
			f, err := plogits2.Get([]int{j, nCats - 2})
			if err != nil {
				panic(err)
			}
			agg += p * f
			data = append(data, agg)
		}

		fish[itm.Name], err = ndvek.NewNdArray([]int{nAbilities}, data)
		if err != nil {
			panic(err)
		}
	}
	return fish
}

func (grm GradedResponseModel) LogLikelihood(abilities *ndvek.NdArray, resp []models.Response) *ndvek.NdArray {
	// Shape of abilities is n_abilities x n_scale

	prob := grm.Prob(abilities)
	shape := abilities.Shape()
	n := shape[0]
	ll := []float64{}
	for i := 0; i < n; i++ {
		ll = append(ll, 0.0)
	}
	for _, r := range resp {
		ndx, err := findIndex(r.Item.ScoredValues, r.Value)
		if err != nil {
			continue
		}
		data := []float64{}
		for i := 0; i < n; i++ {
			p, err := prob[r.Item.Name].Get([]int{i, ndx})
			if err != nil {
				continue
			}
			data = append(data, p)
		}

		for i := 0; i < n; i++ {
			ll[i] += math.Log(data[i])
		}

	}
	ll_, err := ndvek.NewNdArray(shape, ll)
	if err != nil {
		panic(err)
	}
	return ll_
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
		err = plogits.ApplyHadamardOp(math2.Sigmoid)
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

func (m GradedResponseModel) GetItems() []*models.Item {
	return m.Items
}

type AutoencodedGradedResponseModel struct {
	Scales          []models.Scale
	Items           []*models.Item
	Discriminations map[string]ndvek.NdArray
	Difficulties    map[string]ndvek.NdArray
}

func (m AutoencodedGradedResponseModel) GetItems() []*models.Item {
	return m.Items
}

func (m GradedResponseModel) Sample(abilities *ndvek.NdArray) map[string][]int {
	probs := m.Prob(abilities)
	x := make(map[string][]int, 0)
	for label, p := range probs {
		N := p.Shape()[0]
		K := p.Shape()[1]

		for n := range N {
			q := make([]float64, K)
			for k := range K {
				val, err := p.Get([]int{n, k})
				if err != nil {
					panic(err)
				}
				q[k] = val
			}
			x[label] = append(x[label], math2.SampleCategorical(q))
		}

	}
	return x
}
