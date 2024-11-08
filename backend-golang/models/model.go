package models

import (
	"github.com/mederrata/ndvek"
)

type IrtModel interface {
	LogLikelihood(*ndvek.NdArray, *SessionResponses) *ndvek.NdArray // log-likelihood
	Prob(*ndvek.NdArray) map[string]*ndvek.NdArray                  // probs for every item

}

type MultiDimensionalIrtModel interface {
	LogLikelihood(*ndvek.NdArray, *SessionResponses) map[string]*ndvek.NdArray // log-likelihood
	Prob(*ndvek.NdArray) map[string]*ndvek.NdArray
}
