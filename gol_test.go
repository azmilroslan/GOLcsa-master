package main

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"testing"

	"uk.ac.bris.cs/gameoflife/gol"
	"uk.ac.bris.cs/gameoflife/util"
)

/*func BenchmarkTestGol(b *testing.B) {
	//disables all program output apart from benchmark results
	//os.Stdout = nil

	tests := []gol.Params{
		{ImageWidth: 16, ImageHeight: 16},
		{ImageWidth: 64, ImageHeight: 64},
		{ImageWidth: 512, ImageHeight: 512},
	}
	for _, p := range tests {

			p.Turns = 500

			//run sub-benchmarks for all threads
			for threads := 1; threads <= 16; threads*=2 {
				p.Threads = threads
				testName := fmt.Sprintf("%d_workers", p.Threads)
				b.Run(testName, func(b *testing.B) {
					events := make(chan gol.Event)
					for i:=0; i < b.N; i++ {
						go gol.Run(p, events, nil)
					}
					for range events {

					}
				})
			}

	}
}
*/

//simpler Benchmark
/*func BenchmarkTestGol(b *testing.B) {
	//disables all program output apart from benchmark results
	//os.Stdout = nil
	p := gol.Params{
		Turns: 1,
		ImageWidth: 16,
		ImageHeight: 16,
	}
	//run sub-benchmarks for all threads
	for threads := 1; threads <= 16; threads*=2 {
		p.Threads = threads
		testName := fmt.Sprintf("%d_workers", p.Threads)
		b.Run(testName, func(b *testing.B) {
			events := make(chan gol.Event)
			for i:=0; i < b.N; i++ {
				gol.Run(p, events, nil)
			}
		})
	}
}*/

// TestGol tests 16x16, 64x64 and 512x512 images on 0, 1 and 100 turns using 1-16 worker threads.
func TestGol(t *testing.T) {
	tests := []gol.Params{
		{ImageWidth: 16, ImageHeight: 16},
		{ImageWidth: 64, ImageHeight: 64},
		{ImageWidth: 512, ImageHeight: 512},
	}
	for _, p := range tests {
		for _, turns := range []int{0, 1, 100} {
			p.Turns = turns
			expectedAlive := readAliveCells(
				"check/images/"+fmt.Sprintf("%vx%vx%v.pgm", p.ImageWidth, p.ImageHeight, turns),
				p.ImageWidth,
				p.ImageHeight,
			)
			for threads := 1; threads <= 16; threads++ {
				p.Threads = threads
				testName := fmt.Sprintf("%dx%dx%d-%d", p.ImageWidth, p.ImageHeight, p.Turns, p.Threads)
				t.Run(testName, func(t *testing.T) {
					events := make(chan gol.Event)
					go gol.Run(p, events, nil)
					var cells []util.Cell
					for event := range events {
						switch e := event.(type) {
						case gol.FinalTurnComplete:
							cells = e.Alive
						}
					}
					assertEqualBoard(t, cells, expectedAlive, p)
				})
			}
		}
	}
}

func boardFail(t *testing.T, given, expected []util.Cell, p gol.Params) bool {
	errorString := fmt.Sprintf("-----------------\n\n  FAILED TEST\n  %vx%v\n  %d Workers\n  %d Turns\n", p.ImageWidth, p.ImageHeight, p.Threads, p.Turns)
	if p.ImageWidth == 16 && p.ImageHeight == 16 {
		errorString = errorString + util.AliveCellsToString(given, expected, p.ImageWidth, p.ImageHeight)
	}
	t.Error(errorString)
	return false
}

func assertEqualBoard(t *testing.T, given, expected []util.Cell, p gol.Params) bool {
	givenLen := len(given)
	expectedLen := len(expected)

	if givenLen != expectedLen {
		return boardFail(t, given, expected, p)
	}

	visited := make([]bool, expectedLen)
	for i := 0; i < givenLen; i++ {
		element := given[i]
		found := false
		for j := 0; j < expectedLen; j++ {
			if visited[j] {
				continue
			}
			if expected[j] == element {
				visited[j] = true
				found = true
				break
			}
		}
		if !found {
			return boardFail(t, given, expected, p)
		}
	}

	return true
}

func readAliveCells(path string, width, height int) []util.Cell {
	data, ioError := ioutil.ReadFile(path)
	util.Check(ioError)

	fields := strings.Fields(string(data))

	if fields[0] != "P5" {
		panic("Not a pgm file")
	}

	imageWidth, _ := strconv.Atoi(fields[1])
	if imageWidth != width {
		panic("Incorrect width")
	}

	imageHeight, _ := strconv.Atoi(fields[2])
	if imageHeight != height {
		panic("Incorrect height")
	}

	maxval, _ := strconv.Atoi(fields[3])
	if maxval != 255 {
		panic("Incorrect maxval/bit depth")
	}

	image := []byte(fields[4])

	var cells []util.Cell
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			cell := image[0]
			if cell != 0 {
				cells = append(cells, util.Cell{
					X: x,
					Y: y,
				})
			}
			image = image[1:]
		}
	}
	return cells
}
