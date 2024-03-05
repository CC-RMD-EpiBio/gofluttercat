package math

import (
	"golang.org/x/exp/constraints"
)

type Major int

const (
	Row Major = iota
	Column
)

type Number interface {
	constraints.Integer | constraints.Float
}

type NDArray struct {
	data   []float64
	shape  []int
	stride []int
	major  Major
}

func NewNDArray(data []float64, shape []int) *NDArray {
	nd := &NDArray{
		data:  data,
		shape: shape,
		major: Row,
	}
	// assert.Equal(len(data), prod(shape), "Shape and data are incompatible.")
	nd.calculateStride()
	return nd
}

func (nd *NDArray) calculateStride() {
	nd.stride = make([]int, len(nd.shape))
	stride := 1
	for i := len(nd.shape) - 1; i >= 0; i-- {
		nd.stride[i] = stride
		stride *= nd.shape[i]
	}
}

func (nd *NDArray) Index(indices ...int) int {
	if len(indices) != len(nd.shape) {
		panic("Invalid number of indices")
	}
	var index int
	for i, idx := range indices {
		if idx >= nd.shape[i] || idx < 0 {
			panic("Index out of range")
		}
		index += idx * nd.stride[i]
	}
	return index
}

func (nd *NDArray) At(indices ...int) float64 {
	idx := nd.Index(indices...)
	return nd.data[idx]
}

func (nd *NDArray) Set(value float64, indices ...int) {
	idx := nd.Index(indices...)
	nd.data[idx] = value
}

func (nd *NDArray) BroadcastAdd(other *NDArray) *NDArray {
	resultShape := broadcastShape(nd.shape, other.shape)
	resultData := make([]float64, prod(resultShape))
	result := NewNDArray(resultData, resultShape)

	// pad

	for i := 0; i < len(result.data); i++ {
		indices := indicesFromFlatIndex(i, result.shape)
		idx1 := indicesToFlatIndex(broadcastIndex(indices, nd.shape), nd.shape)
		idx2 := indicesToFlatIndex(broadcastIndex(indices, other.shape), other.shape)
		result.data[i] = nd.data[idx1] + other.data[idx2]
	}

	return result
}

func broadcastIndex(index, shape []int) []int {
	result := make([]int, len(index))
	for i := 0; i < len(index); i++ {
		if index[i] > shape[i]-1 {
			result[i] = 0
		} else {
			result[i] = index[i]
		}
	}
	return result
}

func broadcastShape(shape1, shape2 []int) []int {
	maxLen := max(len(shape1), len(shape2))
	resultShape := make([]int, maxLen)
	for i := 0; i < maxLen; i++ {
		idx1 := len(shape1) - 1 - i
		idx2 := len(shape2) - 1 - i
		if idx1 >= 0 && idx2 >= 0 {
			resultShape[maxLen-1-i] = max(shape1[idx1], shape2[idx2])
		} else if idx1 >= 0 {
			resultShape[maxLen-1-i] = shape1[idx1]
		} else if idx2 >= 0 {
			resultShape[maxLen-1-i] = shape2[idx2]
		}
	}
	return resultShape
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func prod(arr []int) int {
	result := 1
	for _, val := range arr {
		result *= val
	}
	return result
}

func indicesFromFlatIndex(flatIndex int, shape []int) []int {
	indices := make([]int, len(shape))
	for i := len(shape) - 1; i >= 0; i-- {
		indices[i] = flatIndex % shape[i]
		flatIndex /= shape[i]
	}
	return indices
}

func indicesToFlatIndex(indices, shape []int) int {
	var flatIndex int
	for i := 0; i < len(shape); i++ {
		flatIndex += indices[i] * prod(shape[i+1:])
	}
	return flatIndex
}
