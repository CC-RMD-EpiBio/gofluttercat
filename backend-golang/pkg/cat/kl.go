package cat

import (
	"fmt"
	"math"

	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/models"
	math2 "github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/math"
	"github.com/mederrata/ndvek"
	"github.com/viterin/vek"
)

type KLSelector struct {
	Temperature    float64
	SurrogateModel *models.IrtModel
}

func (ks KLSelector) Criterion(bs *models.BayesianScorer) map[string]float64 {
	abilities, err := ndvek.NewNdArray([]int{len(bs.AbilityGridPts)}, bs.AbilityGridPts)
	if err != nil {
		panic(err)
	}
	nAbilities := abilities.Shape()[0]

	probs := bs.Model.Prob(abilities)
	admissable := make([]*models.Item, 0)
	answered := make([]*models.Item, 0)
	for _, itm := range bs.Model.GetItems() {
		if hasResponse(itm.Name, bs.Answered) {
			answered = append(answered, itm)
			continue
		}
		admissable = append(admissable, itm)
	}
	piAlpha := bs.Running.Density()
	lpInfy := bs.Running.Energy // log pi_{\alpha_t}
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
	pi_infty := make([]float64, len(lpInfy))
	for i := 0; i < len(lpInfy); i++ {
		pi_infty[i] = math.Exp(lpInfy[i])
	}
	lpInfty_Z := math2.Trapz2(pi_infty, bs.AbilityGridPts)
	for i := 0; i < len(lpInfy); i++ {
		pi_infty[i] /= lpInfty_Z
	}
	// Now compute Eq (8)
	deltaItem := make(map[string]float64, 0)

	for itm, p := range probs {
		var lpItem float64 = 0
		for k := 0; k < p.Shape()[1]; k++ {
			integrand1 := make([]float64, len(bs.AbilityGridPts))
			integrand2 := make([]float64, len(bs.AbilityGridPts))

			for i := 0; i < len(bs.AbilityGridPts); i++ {
				ell, err := p.Get([]int{i, k})
				if err != nil {
					panic(err)
				}
				integrand1[i] = pi_infty[i] * math.Log(ell)
				integrand2[i] = ell * piAlpha[i]
			}
			integral1 := math2.Trapz2(integrand1, bs.AbilityGridPts)
			integral2 := math2.Trapz2(integrand2, bs.AbilityGridPts)

			lpItem += integral2 * (integral1 - math.Log(integral2))
		}
		deltaItem[itm] = -lpItem
	}
	return deltaItem
}

func (ks KLSelector) NextItem(bs *models.BayesianScorer) *models.Item {
	deltaItem := ks.Criterion(bs)
	T := ks.Temperature

	if T == 0 {
		var selected string
		var maxval float64
		for key, value := range deltaItem {
			if value > maxval {
				selected = key
				maxval = value
			}
		}
		return getItemByName(selected, bs.Model.GetItems())
	}

	selectionProbs := make(map[string]float64)
	for key, value := range deltaItem {
		selectionProbs[key] = math.Exp(value / T)
	}

	selected := sample(selectionProbs)
	return getItemByName(selected, bs.Model.GetItems())
}

type McKlSelector struct {
	Temperature float64
	NumSamples  int
}

func NewMcKlSelector(temperature float64, nsamples int) McKlSelector {
	return McKlSelector{
		Temperature: temperature, NumSamples: nsamples}
}

func (ks McKlSelector) Criterion(bs *models.BayesianScorer) map[string]float64 {
	crit := make(map[string]float64, 0)
	abilitySamples := bs.Running.Sample(ks.NumSamples)

	for s, theta := range abilitySamples {
		x := bs.Model.Sample(theta)
		integrand := make([]float64, len(bs.AbilityGridPts))

	}
	fmt.Printf("abilitySamples: %v\n", abilitySamples)
	return crit
}

func (ks McKlSelector) NextItem(bs *models.BayesianScorer) *models.Item {
	crit := ks.Criterion(bs)
	fmt.Printf("density: %v\n", crit)

	// sample
	return nil
}
