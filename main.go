package main

import (
	"flag"
	"fmt"
	"io"
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
	var numLoops int
	flag.StringVar(&inputFile, "input", "", "path to input audio file")
	flag.StringVar(&outputFile, "output", "", "path to output audio file")
	flag.Float64Var(&compareDuration, "compare-duration", 1.0,
		"window of time to compare for")
	flag.Float64Var(&maxOverlap, "max-overlap", 15.0,
		"maximum amount of time overlap between loops")
	flag.Float64Var(&skipTime, "skip-time", 1.0,
		"amount of time to skip at start of repeated track")
	flag.IntVar(&numLoops, "num-loops", 1, "number of times to loop")
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
	slidingWindow := SlidingWindow(samples, startIdx, endIdx, compareSamples)

	bestCorrelation := float32(-1.01)
	bestIndex := startIdx
	index := startIdx
	for corr := range ComputeCorrelations(matchSamples, slidingWindow) {
		if corr > bestCorrelation {
			bestCorrelation = corr
			bestIndex = index
		}
		index++
		if index%1000 == 0 || index == endIdx-1 {
			fmt.Printf(
				"\rcompleted=%.2f%%    correlation=%.2f   ",
				100*float64(1+index-startIdx)/float64(endIdx-startIdx+1),
				bestCorrelation,
			)
		}
	}
	fmt.Println()
	fmt.Println("best overlap time:", float64(bestIndex)/float64(audioInfo.Frequency))

	var combined []float32
	for i := 0; i < numLoops+1; i++ {
		subSamples := samples
		if i <= numLoops {
			subSamples = subSamples[:bestIndex]
		}
		if i > 0 {
			subSamples = subSamples[skipSamples:]
		}
		combined = append(combined, subSamples...)
	}
	WriteSamples(outputFile, audioInfo, combined)
}

func ReadSamples(path string) ([]float32, *ffmpego.AudioInfo) {
	reader, err := ffmpego.NewAudioReader(path)
	essentials.Must(err)
	defer reader.Close()

	var res []float32
	buf := make([]float64, 65536)
	for {
		count, err := reader.ReadSamples(buf)
		for _, x := range buf[:count] {
			res = append(res, float32(x))
		}
		if err == io.EOF {
			break
		}
		essentials.Must(err)
	}
	return res, reader.AudioInfo()
}

func WriteSamples(path string, info *ffmpego.AudioInfo, samples []float32) {
	samples64 := make([]float64, len(samples))
	for i, x := range samples {
		samples64[i] = float64(x)
	}
	writer, err := ffmpego.NewAudioWriter(path, info.Frequency)
	essentials.Must(err)
	essentials.Must(writer.WriteSamples(samples64))
	essentials.Must(writer.Close())
}
