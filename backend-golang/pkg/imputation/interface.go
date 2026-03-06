package imputation

// ImputationModel defines the interface for any imputation model that can
// produce PMFs for missing responses. Both MiceBayesianLoo and
// IrtMixedImputationModel satisfy this interface.
type ImputationModel interface {
	// PredictPMF returns a probability distribution over nCategories
	// response categories for the target variable, given observed items.
	PredictPMF(items map[string]float64, target string, nCategories int, uncertaintyPenalty float64) ([]float64, error)
}
