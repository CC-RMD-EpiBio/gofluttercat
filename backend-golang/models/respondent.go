package models

type gender int

const (
	Male   gender = iota + 1
	Female gender = iota + 1
	Inter  gender = iota + 1
)

type Respondent struct {
	ModelBase
}
