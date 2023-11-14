package irt

import "github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/models"

type gender int

const (
	male   gender = iota + 1
	female gender = iota + 1
	inter  gender = iota + 1
)

type Respondent struct {
	models.ModelBase
	Name   string
	Gender gender
	Age    uint16
	Flags  []string
}

type Sex struct {
}
