package main

import (
	"fmt"
	"testing"
	"uk.ac.bris.cs/gameoflife/gol"
)

/*func BenchmarkGol(b *testing.B) {
	os.Stdout = nil
	tests := []gol.Params{
	//	{ImageWidth: 16, ImageHeight: 16},
	//	{ImageWidth: 64, ImageHeight: 64},
		//{ImageWidth: 128, ImageHeight: 128},
		//{ImageWidth: 256, ImageHeight: 256},
		{ImageWidth: 512, ImageHeight: 512},
	}
	for _, p := range tests {

			p.Turns = 100

			for threads := 1; threads <= 16; threads++ {
				p.Threads = threads
				testName := fmt.Sprintf("%dx%dx%d-%d", p.ImageWidth, p.ImageHeight, p.Turns, p.Threads)
				b.Run(testName, func(b *testing.B) {
					events := make(chan gol.Event)
					go gol.Run(p, events, nil)

					for  range events {

					}
				})
			}

	}
} */

const benchLength = 100

func BenchmarkGol(b *testing.B) {
	for threads := 1; threads <= 16; threads++ {
		//runtime.GOMAXPROCS(threads)
		//debug.SetGCPercent(-1)

		//os.Stdout = nil // Disable all program output apart from benchmark results
		p := gol.Params{
			Turns:       3, //benchLength
			Threads:     threads,
			ImageWidth:  4096,
			ImageHeight: 4096,
		}
		name := fmt.Sprintf("%dx%dx%d-%d", p.ImageWidth, p.ImageHeight, p.Turns, p.Threads)
		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				events := make(chan gol.Event)
				go gol.Run(p, events, nil)
				for range events {

				}
			}
		})
	}
}
