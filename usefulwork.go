package main

import (
	"gonum.org/v1/gonum/mat"
)

func PerformUsefulWork(computations uint64) uint64 {
	rows, cols := 100, 100
	data := make([]float64, rows*cols)
	for i := range data {
		data[i] = float64(i % 100)
	}
	a := mat.NewDense(rows, cols, data)
	b := mat.NewDense(cols, rows, data)
	var result mat.Dense
	result.Mul(a, b)
	return computations
}
