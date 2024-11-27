package rwas

import (
	"fmt"
	"testing"
)

func Test_item_unmarshal(t *testing.T) {
	items_autoencoded := LoadItems("../../rwas/autoencoded")
	fmt.Printf("len(items): %v\n", len(items_autoencoded))
	fmt.Printf("items: %v\n", items_autoencoded)

	items_factorized := LoadItems("../../rwas/factorized")
	fmt.Printf("len(items): %v\n", len(items_factorized))
	fmt.Printf("items: %v\n", items_factorized)

}
