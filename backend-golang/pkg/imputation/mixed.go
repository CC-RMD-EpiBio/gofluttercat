package imputation

import "fmt"

// IrtMixedImputationModel blends MICE and IRT baseline imputation PMFs
// using precomputed per-item weights. For each missing item, the resulting
// PMF is:
//
//	q_mixed(k) = w_mice * q_mice(k) + (1 - w_mice) * q_baseline(k)
//
// where w_mice is derived from comparable ELPD scores (MICE LOO-ELPD vs
// IRT per-item WAIC) via softmax, and q_baseline is either the baseline
// IRT model's marginalized PMF (when a BaselinePredictor is set) or
// uniform (1/K) as fallback.
//
// The weights are computed offline (in Python, during model training) and
// stored as a simple map from item name to MICE weight.
type IrtMixedImputationModel struct {
	MiceModel *MiceBayesianLoo
	// Weights maps item name to the MICE mixing weight (0 to 1).
	// Higher values mean more trust in the MICE model's predictions.
	Weights map[string]float64
	// Baseline provides marginalized PMFs from the baseline IRT model.
	// When nil, uniform (1/K) is used as the IRT fallback.
	Baseline BaselinePredictor
}

// NewIrtMixedImputationModel creates a mixed imputation model from a MICE
// model and precomputed per-item weights.
func NewIrtMixedImputationModel(mice *MiceBayesianLoo, weights map[string]float64) *IrtMixedImputationModel {
	return &IrtMixedImputationModel{
		MiceModel: mice,
		Weights:   weights,
	}
}

// PredictPMF returns a blended PMF mixing the MICE prediction with the
// baseline IRT model's marginalized PMF (or uniform if no baseline is set)
// according to the per-item weight.
func (m *IrtMixedImputationModel) PredictPMF(items map[string]float64, target string, nCategories int, uncertaintyPenalty float64) ([]float64, error) {
	if nCategories <= 0 {
		return nil, fmt.Errorf("nCategories must be positive, got %d", nCategories)
	}

	wMice := 0.5 // default if item not in weights
	if w, ok := m.Weights[target]; ok {
		wMice = w
	}

	// Get MICE PMF
	micePMF, err := m.MiceModel.PredictPMF(items, target, nCategories, uncertaintyPenalty)
	if err != nil {
		// Fall back to uniform if MICE can't predict
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

	// Blend: w_mice * mice_pmf + (1 - w_mice) * baseline_pmf
	result := make([]float64, nCategories)
	var total float64
	for k := range nCategories {
		mice := 0.0
		if k < len(micePMF) {
			mice = micePMF[k]
		}
		result[k] = wMice*mice + (1.0-wMice)*baselinePMF[k]
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
