package main

import (
	"flag"
	"fmt"
	"io"
	"math"
	"os"

	"github.com/unixpickle/essentials"
	"github.com/unixpickle/ffmpego"
)

func main() {
	var inputFile string
	var outputFile string
	var compareDuration float64
	var maxOverlap float64
	var skipTime float64
	flag.StringVar(&inputFile, "input", "", "path to input audio file")
	flag.StringVar(&outputFile, "output", "", "path to output audio file")
	flag.Float64Var(&compareDuration, "compare-duration", 1.0,
		"window of time to compare for")
	flag.Float64Var(&maxOverlap, "max-overlap", 15.0,
		"maximum amount of time overlap between loops")
	flag.Float64Var(&skipTime, "skip-time", 1.0,
		"amount of time to skip at start of repeated track")
	flag.Parse()
	if inputFile == "" || outputFile == "" {
		fmt.Fprintln(os.Stderr, "Missing required -input or -output flag.")
		fmt.Fprintln(os.Stderr)
		flag.Usage()
		os.Exit(1)
	}

	samples, audioInfo := ReadSamples(inputFile)
	compareSamples := int(float64(audioInfo.Frequency) * compareDuration)
	maxOverlapSamples := int(float64(audioInfo.Frequency) * maxOverlap)
	skipSamples := int(float64(audioInfo.Frequency) * skipTime)

	if skipSamples+compareSamples > len(samples) {
		essentials.Die("audio clip is not long enough for input settings")
	}

	matchSamples := samples[skipSamples : skipSamples+compareSamples]
	startIdx := essentials.MaxInt(0, len(samples)-compareSamples-maxOverlapSamples)
	endIdx := len(samples) - compareSamples
	bestCorrelation := -1.01
	bestIndex := endIdx
	for i := endIdx; i >= startIdx; i-- {
		baseSamples := samples[i : i+compareSamples]
		corr := ComputeCorrelation(matchSamples, baseSamples)
		if corr > bestCorrelation {
			bestCorrelation = corr
			bestIndex = i
		}
		if i%1000 == 0 || i == startIdx {
			fmt.Printf(
				"\rcompleted=%.2f%%    correlation=%.2f   ",
				100*float64(1+endIdx-i)/float64(endIdx-startIdx+1),
				bestCorrelation,
			)
		}
	}
	fmt.Println()
	fmt.Println("best overlap time:", float64(bestIndex)/float64(audioInfo.Frequency))

	combined := append(samples[:bestIndex], samples[skipSamples:]...)
	WriteSamples(outputFile, audioInfo, combined)
}

func ReadSamples(path string) ([]float64, *ffmpego.AudioInfo) {
	reader, err := ffmpego.NewAudioReader(path)
	essentials.Must(err)
	defer reader.Close()

	var res []float64
	buf := make([]float64, 65536)
	for {
		count, err := reader.ReadSamples(buf)
		res = append(res, buf[:count]...)
		if err == io.EOF {
			break
		}
		essentials.Must(err)
	}
	return res, reader.AudioInfo()
}

func WriteSamples(path string, info *ffmpego.AudioInfo, samples []float64) {
	writer, err := ffmpego.NewAudioWriter(path, info.Frequency)
	essentials.Must(err)
	essentials.Must(writer.WriteSamples(samples))
	essentials.Must(writer.Close())
}

func ComputeCorrelation(s1, s2 []float64) float64 {
	if len(s1) != len(s2) {
		panic("mismatch in length")
	}
	var m1, m2 float64
	var dotProd float64
	for i, x := range s1 {
		m1 += x * x
		y := s2[i]
		m2 += y * y
		dotProd += x * y
	}
	return dotProd / math.Sqrt(m1*m2)
}
