package gol

import (
	"strconv"
	"strings"
	"uk.ac.bris.cs/gameoflife/util"
)

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioInput    <-chan uint8
	ioOutput   chan<- uint8
}

//GOL Logic
//func worker(p Params, world [][]byte, threadNum int, )

// func to create an empty 2D slice (world)
func createSlice(p Params, height int) [][]byte {
	newSlice := make([][]byte, height)
	for i := 0; i < height; i++ {
		newSlice[i] = make([]byte, p.ImageWidth)
	}
	return newSlice
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {

	// TODO: Create a 2D slice to store the world.
	world := createSlice(p, p.ImageHeight)

	//request to read in pgm file
	c.ioCommand <- ioInput
	//read in the concatenated ImageWidth and ImageHeight and pass it to the channel
	c.ioFilename <- strings.Join([]string{strconv.Itoa(p.ImageWidth), strconv.Itoa(p.ImageHeight)}, "x")

	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			val := <-c.ioInput
			if val != 0 {
				world[y][x] = val
			}
		}
	}

	turn := 0

	// TODO: Execute all turns of the Game of Life.
	for t := 0; t < p.Turns; t++ {

		for i := 0; i < p.Threads; i++ { //for each threads make the worker work??
			//go worker()
		}
		turn++ //add turn count
	}
	// TODO: Report the final state using FinalTurnCompleteEvent.

	var aliveCells []util.Cell

	// go through the 'world' and append cells that are still alive
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			if world[y][x] != 0 {
				aliveCells = append(aliveCells, util.Cell{X: x, Y: y})
			}
		}
	}

	// put FinalTurnComplete into events channel
	c.events <- FinalTurnComplete{turn, aliveCells}
	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
