package imputation

// ImputationModel defines the interface for any imputation model that can
// produce PMFs for missing responses. Both MiceBayesianLoo and
// IrtMixedImputationModel satisfy this interface.
type ImputationModel interface {
	// PredictPMF returns a probability distribution over nCategories
	// response categories for the target variable, given observed items.
	PredictPMF(items map[string]float64, target string, nCategories int, uncertaintyPenalty float64) ([]float64, error)
}

// BaselinePredictor provides marginalized PMFs from a baseline IRT model.
// The BayesianScorer in the irtcat package implements this interface,
// allowing the mixed imputation model to use baseline IRT predictions
// instead of uniform fallback without creating a circular import.
type BaselinePredictor interface {
	// BaselinePMF returns a PMF for the target item by marginalizing
	// P(Y=k|theta) over the current baseline posterior.
	BaselinePMF(target string, nCategories int) ([]float64, error)
}
