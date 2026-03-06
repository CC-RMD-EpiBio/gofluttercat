package imputation

import "fmt"

// IrtMixedImputationModel blends MICE and IRT baseline imputation PMFs
// using precomputed per-item weights. For each missing item, the resulting
// PMF is:
//
//	q_mixed(k) = w_mice * q_mice(k) + (1 - w_mice) * (1/K)
//
// where w_mice is derived from comparable ELPD scores (MICE LOO-ELPD vs
// IRT per-item WAIC) via softmax.
//
// The weights are computed offline (in Python, during model training) and
// stored as a simple map from item name to MICE weight.
type IrtMixedImputationModel struct {
	MiceModel *MiceBayesianLoo
	// Weights maps item name to the MICE mixing weight (0 to 1).
	// Higher values mean more trust in the MICE model's predictions.
	Weights map[string]float64
}

// NewIrtMixedImputationModel creates a mixed imputation model from a MICE
// model and precomputed per-item weights.
func NewIrtMixedImputationModel(mice *MiceBayesianLoo, weights map[string]float64) *IrtMixedImputationModel {
	return &IrtMixedImputationModel{
		MiceModel: mice,
		Weights:   weights,
	}
}

// PredictPMF returns a blended PMF mixing the MICE prediction with a
// uniform (ignorable) distribution according to the per-item weight.
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

	// Blend: w_mice * mice_pmf + (1 - w_mice) * uniform
	uniformProb := 1.0 / float64(nCategories)
	result := make([]float64, nCategories)
	var total float64
	for k := range nCategories {
		mice := 0.0
		if k < len(micePMF) {
			mice = micePMF[k]
		}
		result[k] = wMice*mice + (1.0-wMice)*uniformProb
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
