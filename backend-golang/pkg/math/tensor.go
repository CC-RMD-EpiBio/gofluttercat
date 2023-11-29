package math

import (
	"fmt"

	"gorgonia.org/tensor"
)

func AddWithBroadcast(a, b *tensor.Dense) (*tensor.Dense, error) {
	// Check if broadcasting is needed and possible
	aShape := a.Shape()
	bShape := b.Shape()

	resultShape, err := broadcastShapes(aShape, bShape)
	if err != nil {
		return nil, err
	}

	// Initialize the result tensor
	result := tensor.New(tensor.Of(a.Dtype()), tensor.WithShape(resultShape...))

	// Perform element-wise addition with broadcasting
	err = elementwiseAddWithBroadcast(result, a, b)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// broadcastShapes returns the shape of the result tensor after broadcasting a and b.
func broadcastShapes(a, b []int) ([]int, error) {
	// Determine the maximum dimensionality
	maxDim := len(a)
	if len(b) > maxDim {
		maxDim = len(b)
	}

	// Pad shapes with 1 to make them have the same dimensionality
	a = padShape(a, maxDim)
	b = padShape(b, maxDim)

	// Check if broadcasting is possible
	for i := 0; i < maxDim; i++ {
		if a[i] != b[i] && a[i] != 1 && b[i] != 1 {
			return nil, fmt.Errorf("shapes %v and %v are not broadcastable", a, b)
		}
	}

	// Determine the result shape after broadcasting
	resultShape := make([]int, maxDim)
	for i := 0; i < maxDim; i++ {
		resultShape[i] = max(a[i], b[i])
	}

	return resultShape, nil
}

// padShape pads the shape with 1 to make it have the specified dimensionality.
func padShape(shape []int, targetDim int) []int {
	padding := make([]int, targetDim-len(shape))
	return append(padding, shape...)
}

// elementwiseAddWithBroadcast performs element-wise addition with broadcasting.
func elementwiseAddWithBroadcast(result, a, b *tensor.Dense) error {
	// Iterate over the result tensor and perform element-wise addition
	iter := result.Iterator()
	for _, err := iter.Start(); err == nil; _, err = iter.Next() {
		indices := iter.Coord()

		// Broadcast indices to the original shapes of a and b
		aIndex := broadcastIndices(indices, a.Shape())
		bIndex := broadcastIndices(indices, b.Shape())
		A, _ := a.At(aIndex...)
		B, _ := b.At(bIndex...)

		// Perform addition
		result.SetAt(indices, A+B)
	}

	return nil
}

// broadcastIndices returns the broadcasted indices based on the original shape.
func broadcastIndices(indices, originalShape []int) []int {
	result := make([]int, len(originalShape))
	for i := 0; i < len(indices); i++ {
		if originalShape[i] == 1 {
			result[i] = 0
		} else {
			result[i] = indices[i]
		}
	}
	return result
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
