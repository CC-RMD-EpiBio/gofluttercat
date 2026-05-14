// Package biasleverage implements the per-item bias-leverage diagnostic
// described in the cat_optimalcontrol manuscript:
//
//	B_i = E[ |log pi_pairwise(y_i | y_{-i}) - log p_IRT(y_i | theta_p_hat)| ]
//	F_i = Fisher information of item i at the population mean ability
//
// Items with high B_i / F_i are the most "dangerous to omit" in CAT
// design: they contribute strongly to subset bias when dropped but
// contribute little to ability identification.
//
// This package is intentionally dependency-free (no irtcat imports) and
// mirrors the structure of compute_item_bias_leverage.py in the
// cat_optimalcontrol repository, so the Go and Python implementations
// can be cross-validated on the same input.
package biasleverage

import (
	"fmt"
	"math"
	"sort"
)

// GRMItem is the minimal set of GRM parameters needed for PMF and Fisher
// info computation for one item.
type GRMItem struct {
	DDiff []float64 // monotone-positive successive differences, length K-2
	A     float64   // discrimination
	D0    float64   // first difficulty (cutpoint)
	K     int       // number of response categories (>= 2)
}

// GRMPMF returns the K-element PMF for item under the GRM, evaluated at
// ability theta. Categories are 0..K-1.
//
// Convention matches the bayesianquilts production GRM:
//
//	eta_k        = a * (theta - tau_k),  k = 1..K-1
//	P(Y >= k)    = sigmoid(eta_k)
//	P(Y =  0)    = 1 - sigmoid(eta_1)
//	P(Y =  k)    = sigmoid(eta_k) - sigmoid(eta_{k+1}),  1 <= k <= K-2
//	P(Y = K-1)   = sigmoid(eta_{K-1})
//
// where tau_k are monotone-increasing thresholds built from D0 + cumulative
// positive DDiff increments.
func GRMPMF(theta float64, item GRMItem) []float64 {
	if item.K < 2 {
		return []float64{1.0}
	}
	thresholds := make([]float64, item.K-1)
	thresholds[0] = item.D0
	for k := 1; k < item.K-1; k++ {
		thresholds[k] = thresholds[k-1] + item.DDiff[k-1]
	}
	// cdf[k] = P(Y >= k); cdf[0] = 1 (always), cdf[K] = 0 (always).
	cdf := make([]float64, item.K+1)
	cdf[0] = 1.0
	cdf[item.K] = 0.0
	for k := 1; k < item.K; k++ {
		eta := item.A * (theta - thresholds[k-1])
		cdf[k] = 1.0 / (1.0 + math.Exp(-eta))
	}
	pmf := make([]float64, item.K)
	for k := 0; k < item.K; k++ {
		pmf[k] = cdf[k] - cdf[k+1]
	}
	return pmf
}

// FisherInfo returns the Fisher information of item evaluated at theta,
// computed by central finite difference of log p with eps = 1e-3 (matches
// compute_item_bias_leverage.py).
func FisherInfo(theta float64, item GRMItem) float64 {
	const eps = 1e-3
	const floor = 1e-12
	p := GRMPMF(theta, item)
	pp := GRMPMF(theta+eps, item)
	pm := GRMPMF(theta-eps, item)
	var info float64
	for k := 0; k < item.K; k++ {
		pk := p[k]
		if pk < floor {
			pk = floor
		}
		ppk := pp[k]
		if ppk < floor {
			ppk = floor
		}
		pmk := pm[k]
		if pmk < floor {
			pmk = floor
		}
		dlog := (math.Log(ppk) - math.Log(pmk)) / (2 * eps)
		info += pk * dlog * dlog
	}
	return info
}

// PMFProvider produces a PMF for the target item conditional on the
// observed responses. Both PairwiseStackingModel and IrtMixedImputationModel
// in the imputation package satisfy this shape, but importing them here
// would create a cycle with irtcat — clients pass an adapter.
type PMFProvider interface {
	// PredictPMF returns the imputation PMF for target given items.
	// items maps item key -> response value.
	PredictPMF(items map[string]float64, target string, K int) ([]float64, error)
}

// Person is one training row: item key -> response value (-1 or NaN means
// missing). Bias-leverage iterates only over the items present in this map
// and within [0, K).
type Person map[string]float64

// Result is the per-item leverage summary computed by Compute.
type Result struct {
	Item  string
	B     float64 // mean |log pi_pw - log p_IRT| across people for this item
	F     float64 // Fisher info at population theta
	Ratio float64 // B / F (descending = most dangerous to omit)
	NEff  int     // number of (person, item) deltas accumulated
}

// Compute computes B_i, F_i, and the ratio for every item key in itemParams.
//
//   - items:    map item key -> GRMItem parameters
//   - itemKeys: ordering for tie-breaking and CSV output
//   - training: per-person observed responses (item key -> response value)
//   - thetaHat: per-person baseline EAP scores (length == len(training))
//   - imp:      PMFProvider (e.g. *PairwiseStackingModel)
//   - thetaBar: population mean ability for Fisher-info evaluation
//
// Returns one Result per item, sorted descending by Ratio.
func Compute(
	itemParams map[string]GRMItem,
	itemKeys []string,
	training []Person,
	thetaHat []float64,
	imp PMFProvider,
	thetaBar float64,
) ([]Result, error) {
	if len(thetaHat) != len(training) {
		return nil, fmt.Errorf("len(thetaHat)=%d != len(training)=%d",
			len(thetaHat), len(training))
	}

	const floor = 1e-12

	// Accumulators per item.
	sumAbsDelta := make(map[string]float64, len(itemKeys))
	count := make(map[string]int, len(itemKeys))

	for p, person := range training {
		theta := thetaHat[p]
		if math.IsNaN(theta) || math.IsInf(theta, 0) {
			continue
		}
		// Build the observed-only map (filter NaN / out-of-range).
		obs := make(map[string]float64, len(person))
		for key, v := range person {
			ip, ok := itemParams[key]
			if !ok {
				continue
			}
			if math.IsNaN(v) {
				continue
			}
			iv := int(v)
			if iv < 0 || iv >= ip.K {
				continue
			}
			obs[key] = v
		}
		for key, ip := range itemParams {
			yi, present := obs[key]
			if !present {
				continue
			}
			others := make(map[string]float64, len(obs)-1)
			for k, v := range obs {
				if k != key {
					others[k] = v
				}
			}
			if len(others) == 0 {
				continue
			}
			pmfPw, err := imp.PredictPMF(others, key, ip.K)
			if err != nil {
				continue
			}
			pmfIrt := GRMPMF(theta, ip)
			ki := int(yi)
			pPw := pmfPw[ki]
			if pPw < floor {
				pPw = floor
			}
			pIrt := pmfIrt[ki]
			if pIrt < floor {
				pIrt = floor
			}
			delta := math.Log(pPw) - math.Log(pIrt)
			if delta < 0 {
				delta = -delta
			}
			sumAbsDelta[key] += delta
			count[key]++
		}
	}

	out := make([]Result, 0, len(itemKeys))
	for _, key := range itemKeys {
		ip := itemParams[key]
		var B float64
		if c := count[key]; c > 0 {
			B = sumAbsDelta[key] / float64(c)
		} else {
			B = math.NaN()
		}
		F := FisherInfo(thetaBar, ip)
		ratio := math.NaN()
		if F > 1e-6 {
			ratio = B / F
		}
		out = append(out, Result{
			Item:  key,
			B:     B,
			F:     F,
			Ratio: ratio,
			NEff:  count[key],
		})
	}
	// Sort by Ratio descending (highest = most dangerous to omit), with
	// NaN sorted to the end.
	sort.SliceStable(out, func(i, j int) bool {
		ri, rj := out[i].Ratio, out[j].Ratio
		switch {
		case math.IsNaN(ri) && math.IsNaN(rj):
			return out[i].Item < out[j].Item
		case math.IsNaN(ri):
			return false
		case math.IsNaN(rj):
			return true
		default:
			return ri > rj
		}
	})
	return out, nil
}
