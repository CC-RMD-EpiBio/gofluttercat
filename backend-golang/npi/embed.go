package npi

import "embed"

//go:embed factorized
var FactorizedDir embed.FS

//go:embed baseline_factorized
var BaselineFactorizedDir embed.FS

//go:embed imputation_model
var ImputationModelDir embed.FS
