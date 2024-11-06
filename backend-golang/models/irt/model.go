package irt

import "github.com/mederrata/ndvek"

type ScaleDb struct {
}

type SessionResponses struct {
	ItemDb  *ItemDb
	ScaleDB *ScaleDb
}

type IrtModel interface {
	LogLikelihood(*ndvek.NdArray, *SessionResponses) map[string]*ndvek.NdArray // log-likelihood
	Prob(*ndvek.NdArray) map[string]*ndvek.NdArray                             // probs for every item

}

type MultiDimensionalIrtModel interface {
	LogLikelihood(*ndvek.NdArray, *SessionResponses) map[string]*ndvek.NdArray // log-likelihood
	Prob(*ndvek.NdArray) map[string]*ndvek.NdArray
}
