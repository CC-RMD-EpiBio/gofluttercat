package irt

import "github.com/mederrata/ndvek"

type ScaleDb struct {
}

type SessionResponses struct {
	ItemDb  *ItemDb
	ScaleDB *ScaleDb
}

type IrtModel interface {
	LogLikelihood(*ndvek.NdArray, *SessionResponses) *ndvek.NdArray // log-likelihood
	Prob(*ndvek.NdArray) *ndvek.NdArray                              // probs for every item

}
