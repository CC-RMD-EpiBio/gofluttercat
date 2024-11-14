package models

import (
	"github.com/mederrata/ndvek"
)

type IrtModel interface {
	LogLikelihood(*ndvek.NdArray, []Response) *ndvek.NdArray // log-likelihood
	Prob(*ndvek.NdArray) map[string]*ndvek.NdArray
	FisherInformation(*ndvek.NdArray) map[string]*ndvek.NdArray // probs for every item

}

type MultiDimensionalIrtModel interface {
	LogLikelihood(*ndvek.NdArray, []Response) map[string]*ndvek.NdArray // log-likelihood
	Prob(*ndvek.NdArray) map[string]*ndvek.NdArray
}
