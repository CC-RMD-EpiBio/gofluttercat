package models

import (
	"encoding/json"
	"log"
	"os"

	"github.com/mederrata/ndvek"
)

type Item struct {
	Name          string                 `json:"item"`
	Question      string                 `json:"question"`
	Choices       map[string]Choice      `json:"responses"`
	ScaleLoadings map[string]Calibration `json:"scales"`
	Version       float32                `json:"version"`
	ScoredValues  []int                  `json:"scored_vales"`
}

type ItemDb struct {
	Items *[]Item
}

type Choice struct {
	Text  string `json:"text"`
	Value uint   `json:"value"`
}

type Calibration struct {
	Difficulties   []float64 `json:"difficulties"`
	Discrimination float64   `json:"discrimination"`
}

func LoadItem(path string) *Item {
	dat, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	item := &Item{
		ScoredValues: []int{1, 2, 3, 4, 5},
	}
	if err := json.Unmarshal(dat, &item); err != nil {
		log.Fatal(err)
	}
	return item
}

func (itm Item) Prob(ability float64) *ndvek.NdArray {
	nScored := len(itm.ScoredValues)
	probs, err := ndvek.NewNdArray([]int{nScored}, nil)
	if err != nil {
		panic(err)
	}
	return probs
}
