package irt

import (
	"fmt"
	"log"
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

func Test_scale_unmarshal(t *testing.T) {
	dat, err := os.ReadFile("/Users/changjc/workspace/gofabulouscat/v3/scales.yaml")
	check(err)
	var c ScaleInfo
	if err := yaml.Unmarshal(dat, &c); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("c: %v\n", c)
}
