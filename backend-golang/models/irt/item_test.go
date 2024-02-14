package irt

import (
	"fmt"
	"log"
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}
func Test_item_unmarshal(t *testing.T) {
	dat, err := os.ReadFile("/Users/changjc/workspace/gofabulouscat/v3/items.yaml")
	check(err)
	var c []Item
	if err := yaml.Unmarshal(dat, &c); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("c: %v\n", c)
}
