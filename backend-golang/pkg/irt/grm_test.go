package irt

import (
	"fmt"
	"testing"

	"gonum.org/v1/gonum/mat"
	"gorgonia.org/tensor"
)

func addWithBroadcast(a, b *mat.Dense) (*mat.Dense, error) {
	r, c := a.Dims()
	rb, cb := b.Dims()

	// Check if broadcasting is needed and possible
	if r != rb && c != cb {
		return nil, fmt.Errorf("shapes %v and %v are not broadcastable", a.Shape(), b.Shape())
	}

	// Initialize the result tensor
	result := mat.NewDense(r, c, nil)

	// Perform element-wise addition with broadcasting
	for i := 0; i < r; i++ {
		for j := 0; j < c; j++ {
			result.Set(i, j, a.At(i, j)+b.At(i%rb, j%cb))
		}
	}

	return result, nil
}

func Test_grm(t *testing.T) {
	theta := tensor.New(
		tensor.WithBacking([]int{0, 0, 0, 0, 0, 0}),
		tensor.WithShape(3, 2),
	)
	theta2 := tensor.New(
		tensor.WithBacking([]int{0, 0, 0}),
		tensor.WithShape(3, 1),
	)
	out, err := theta.Add(theta2)
	if err != nil {
		panic(err)
	}
	fmt.Printf("a:\n%v\n", out)
}
