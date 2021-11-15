package gol

import (
	"strconv"
	"strings"
	"sync"
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
func worker(p Params, world, emptyWorld [][]byte, thread, workerHeight int, group *sync.WaitGroup, workerChan chan [][]byte) {
	yBound := (thread+1) * workerHeight

	for y := thread * workerHeight; y < yBound; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			xRight, xLeft := x+1, x-1
			yUp, yDown := y+1, y-1

			//pixel at far right connected to the pixel at far left
			if xRight >= p.ImageWidth {
				xRight %= p.ImageWidth
			}
			if xLeft < 0 {
				xLeft += p.ImageWidth
			}
			//pixel at the top connected to pixel at the bottom
			if yUp >= p.ImageHeight {
				yUp %= p.ImageHeight
			}
			if yDown < 0 {
				yDown += p.ImageHeight
			}
			count := 0 //count the number of neighbouring living cells
			count += int(world[yUp][xLeft]) +
				int(world[yUp][x]) +
				int(world[yUp][xRight]) +
				int(world[y][xLeft]) +
				int(world[y][xRight]) +
				int(world[yDown][xLeft]) +
				int(world[yDown][x]) +
				int(world[yDown][xRight])
			count /= 255
			if (world[y][x] == 0xFF && count == 2) || (world[y][x] == 0xFF && count == 3) || (count == 3) {
				emptyWorld[y][x] = 0xFF
			} else {
				emptyWorld[y][x] = 0
			}
		}
	}
	workerChan <- emptyWorld
	group.Done()
}

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
	updateWorld := createSlice(p, p.ImageHeight)
	workerHeight := p.ImageHeight / p.Threads // 'split' the work (like in Median Filter lab)

	//request to read in pgm file
	c.ioCommand <- ioInput
	//read in the concatenated ImageWidth and ImageHeight and pass it to the channel
	c.ioFilename <- strings.Join([]string{strconv.Itoa(p.ImageWidth), strconv.Itoa(p.ImageHeight)}, "x")

	//add values to the 2D slice
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			val := <-c.ioInput
			world[y][x] = val
		}
	}

	turn := 0

	//THREAD IMPLEMENTATION
	threadChannels := make([]chan [][]byte, p.Threads)
	part := make ([] [][]byte,p.Threads)

	// TODO: Execute all turns of the Game of Life.
	if p.Turns != 0 {
		for t := 0; t < p.Turns; t++ {

			var wg = &sync.WaitGroup{}
			wg.Add(p.Threads)
			for i := 0; i < p.Threads; i++ { //for each threads make channel and worker
				threadChannels[i] = make(chan [][]byte)
				go worker(p, world, updateWorld, i, workerHeight, wg, threadChannels[i])
				//tmp := world
				//world = updateWorld
				//updateWorld = tmp
			}
			wg.Wait()

			for i := 0; i < p.Threads; i++ { //reassemble slices from workers
				part[i] = <- threadChannels[i]
				for _, bytes := range part[i] {
					updateWorld = append(updateWorld, bytes)
				}
				tmp := world
				world = updateWorld
				updateWorld = tmp
			}
			turn++ //add turn count
		}
	} else {
		updateWorld = world
	}

	//update the 2D world slice

	// TODO: Report the final state using FinalTurnCompleteEvent.

	var aliveCells []util.Cell

	// go through the 'world' and append cells that are still alive
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			if world[y][x] != 0 { //if pixel is not 0 (black/dead), we append
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
