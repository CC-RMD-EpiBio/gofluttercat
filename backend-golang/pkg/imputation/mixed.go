package imputation

import "fmt"

// IrtMixedImputationModel blends pairwise stacking and IRT baseline imputation PMFs
// using precomputed per-item weights.
// PMF is:
//
//	q_mixed(k) = w_pairwise * q_pairwise(k) + (1 - w_pairwise) * q_baseline(k)
//
// where w_pairwise is derived from comparable ELPD scores (pairwise stacking LOO-ELPD vs
// IRT per-item WAIC) via softmax, and q_baseline is either the baseline
// IRT model's marginalized PMF (when a BaselinePredictor is set) or
// uniform (1/K) as fallback.
//
// The weights are computed offline (in Python, during model training) and
// stored as a simple map from item name to pairwise weight.
type IrtMixedImputationModel struct {
	PairwiseModel *PairwiseStackingModel
	// Weights maps item name to the pairwise mixing weight (0 to 1).
	// Higher values mean more trust in the pairwise model's predictions.
	Weights map[string]float64
	// Baseline provides marginalized PMFs from the baseline IRT model.
	// When nil, uniform (1/K) is used as the IRT fallback.
	Baseline BaselinePredictor
}

// NewIrtMixedImputationModel creates a mixed imputation model from a pairwise
// stacking model and precomputed per-item weights.
func NewIrtMixedImputationModel(pairwise *PairwiseStackingModel, weights map[string]float64) *IrtMixedImputationModel {
	return &IrtMixedImputationModel{
		PairwiseModel: pairwise,
		Weights:   weights,
	}
}

// PredictPMF returns a blended PMF mixing the pairwise prediction with the
// baseline IRT model's marginalized PMF (or uniform if no baseline is set)
// according to the per-item weight.
func (m *IrtMixedImputationModel) PredictPMF(items map[string]float64, target string, nCategories int, uncertaintyPenalty float64) ([]float64, error) {
	if nCategories <= 0 {
		return nil, fmt.Errorf("nCategories must be positive, got %d", nCategories)
	}

	wPairwise := 0.5 // default if item not in weights
	if w, ok := m.Weights[target]; ok {
		wPairwise = w
	}

	// Get pairwise PMF
	pairwisePMF, err := m.PairwiseModel.PredictPMF(items, target, nCategories, uncertaintyPenalty)
	if err != nil {
		// Fall back to uniform if pairwise model can't predict
		uniform := make([]float64, nCategories)
		for k := range nCategories {
			uniform[k] = 1.0 / float64(nCategories)
		}
		return uniform, nil
	}

	// Get baseline IRT PMF (marginalized over baseline posterior) or uniform
	baselinePMF := make([]float64, nCategories)
	if m.Baseline != nil {
		bp, err := m.Baseline.BaselinePMF(target, nCategories)
		if err == nil && len(bp) == nCategories {
			copy(baselinePMF, bp)
		} else {
			for k := range nCategories {
				baselinePMF[k] = 1.0 / float64(nCategories)
			}
		}
	} else {
		for k := range nCategories {
			baselinePMF[k] = 1.0 / float64(nCategories)
		}
	}

	// Blend: w_pairwise * pairwise_pmf + (1 - w_pairwise) * baseline_pmf
	result := make([]float64, nCategories)
	var total float64
	for k := range nCategories {
		pw := 0.0
		if k < len(pairwisePMF) {
			pw = pairwisePMF[k]
		}
		result[k] = wPairwise*pw + (1.0-wPairwise)*baselinePMF[k]
		total += result[k]
	}

	// Normalize
	if total > 0 {
		for k := range result {
			result[k] /= total
		}
	}
	return result, nil
}
