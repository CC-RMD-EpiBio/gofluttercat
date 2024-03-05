package math

import (
	"fmt"
	"testing"
)

func Test_tensor(t *testing.T) {

	data := []float64{1, 2, 3, 4, 5, 6}
	data2 := []float64{1, 5, 6}
	shape := []int{2, 3}
	shape2 := []int{1, 3}
	ndarray := NewNDArray(data, shape)
	ndarray2 := NewNDArray(data2, shape2)
	idx2 := indicesToFlatIndex(broadcastIndex([]int{1, 0}, ndarray2.shape), ndarray2.shape)
	fmt.Printf("idx2: %v\n", idx2)

	ndarray3 := ndarray.BroadcastAdd(ndarray2)
	fmt.Printf("ndarray: %v\n", ndarray3)
}
