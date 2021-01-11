package main

import (
	"math"

	"gonum.org/v1/gonum/blas"
	"gonum.org/v1/gonum/blas/blas32"
)

const (
	BufferSize   = 128
	NormInterval = 1024
)

type NormVec struct {
	Data []float32
	Norm float32
}

func SlidingWindow(slice []float32, start, end, length int) <-chan NormVec {
	res := make(chan NormVec, 1)
	go func() {
		defer close(res)
		var norm float32
		for i := start; i < end; i++ {
			vec := slice[i : i+length]
			if (i-start)%NormInterval == 0 {
				norm = blas32.Nrm2(blas32.Vector{
					N:    len(vec),
					Inc:  1,
					Data: vec,
				})
				norm = norm * norm
			} else {
				norm -= slice[i-1] * slice[i-1]
				norm += vec[length-1] * vec[length-1]
			}
			res <- NormVec{
				Data: vec,
				Norm: float32(math.Sqrt(float64(norm))),
			}
		}
	}()
	return res
}

func ComputeCorrelations(reference []float32, samples <-chan NormVec) <-chan float32 {
	res := make(chan float32, 1)
	refNorm := blas32.Nrm2(blas32.Vector{
		N:    len(reference),
		Inc:  1,
		Data: reference,
	})
	go func() {
		defer close(res)
		var buffer []float32
		var norms []float32
		var bufferCount int
		flushBuffer := func() {
			matrix := blas32.General{
				Rows:   bufferCount,
				Cols:   len(reference),
				Stride: len(reference),
				Data:   buffer,
			}
			vector := blas32.Vector{
				N:    len(reference),
				Inc:  1,
				Data: reference,
			}
			dots := blas32.Vector{
				N:    bufferCount,
				Inc:  1,
				Data: make([]float32, bufferCount),
			}
			blas32.Gemv(blas.NoTrans, 1, matrix, vector, 0, dots)

			for i, d := range dots.Data {
				res <- d / (norms[i] * refNorm)
			}

			buffer = buffer[:0]
			norms = norms[:0]
			bufferCount = 0
		}
		for sample := range samples {
			buffer = append(buffer, sample.Data...)
			norms = append(norms, sample.Norm)
			bufferCount++
			if bufferCount == BufferSize {
				flushBuffer()
			}
		}
		flushBuffer()
	}()
	return res
}
