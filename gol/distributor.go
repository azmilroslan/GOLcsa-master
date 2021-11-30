package gol

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
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
func worker(p Params, c distributorChannels, world, emptyWorld [][]byte, thread, workerHeight, turn int, aliveCells *[]util.Cell, waitGroup *sync.WaitGroup) {

	yBound := (thread + 1) * workerHeight
	extra := p.ImageHeight % p.Threads

	//fmt.Printf("worker thread %d \n", thread + 1)

	if !isPowOfTwo(p.Threads) {
		yBound += extra
	}

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
			count := 0 //count the number of neighbouring live cells
			count += int(world[yUp][xLeft]) +
				int(world[yUp][x]) +
				int(world[yUp][xRight]) +
				int(world[y][xLeft]) +
				int(world[y][xRight]) +
				int(world[yDown][xLeft]) +
				int(world[yDown][x]) +
				int(world[yDown][xRight])
			count /= 255

			if (world[y][x] == 0xFF && count == 2) || (world[y][x] == 0xFF && count == 3) {
				emptyWorld[y][x] = 0xFF
				c.events <- CellFlipped{
					CompletedTurns: turn,
					Cell:           util.Cell{X: x, Y: y},
				}
			} else if world[y][x] == 0xFF && count < 2 {
				emptyWorld[y][x] = 0
				c.events <- CellFlipped{
					CompletedTurns: turn,
					Cell:           util.Cell{X: x, Y: y},
				}
			} else if world[y][x] == 0xFF && count > 3 {
				emptyWorld[y][x] = 0
				c.events <- CellFlipped{
					CompletedTurns: turn,
					Cell:           util.Cell{X: x, Y: y},
				}
			} else if count == 3 {
				emptyWorld[y][x] = 0xFF
			} else {
				emptyWorld[y][x] = 0
			}
		}
	}

	//time.Sleep(5 * time.Millisecond)
	//wl14928

	waitGroup.Done() //-1 in the wait group
}

// func to count the number of alive cells
func countAliveCells(p Params, world [][]byte) int {
	alive := 0
	for y := 0; y < p.ImageWidth; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			if world[y][x] == 0xFF {
				alive++
			}
		}
	}
	return alive
}

//func to output file to a pgm file
func outputFileToPGM(p Params, c distributorChannels, world [][]byte, turn int) {
	c.ioCommand <- ioOutput
	c.ioFilename <- strings.Join([]string{strconv.Itoa(p.ImageWidth), strconv.Itoa(p.ImageHeight), strconv.Itoa(turn)}, "x")
	for y := range world { //send world via output channel byte by byte
		for x := range world[y] {
			c.ioOutput <- world[y][x]
		}
	}
	c.events <- ImageOutputComplete{turn, strings.Join([]string{strconv.Itoa(p.ImageWidth), strconv.Itoa(p.ImageHeight), strconv.Itoa(turn)}, "x")}
}

// func to create an empty 2D slice (world)
func createSlice(p Params, height int) [][]byte {
	newSlice := make([][]byte, height)
	for i := 0; i < height; i++ {
		newSlice[i] = make([]byte, p.ImageWidth)
	}
	return newSlice
}

//helper function to det. thread dimensions
//Since the image HxW are power of two's, it can only be split
//perfectly if the number of threads are power of two's
//this part will check if p.Threads is a power of two
func isPowOfTwo(n int) bool {
	if n == 0 {
		return false
	} else {
		for n != 0 {
			if n%2 != 1 {
				return false
			}
			n = n / 2
		}
		return true
	}
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels, keyChan <-chan rune) {

	// TODO: Create a 2D slice to store the world.

	world := createSlice(p, p.ImageHeight)
	workerHeight := p.ImageHeight / p.Threads // 'split' the work (like in Median Filter lab)

	//request to read in pgm file
	c.ioCommand <- ioInput
	//read in the concatenated ImageWidth and ImageHeight and pass it to the channel
	c.ioFilename <- strings.Join([]string{strconv.Itoa(p.ImageWidth), strconv.Itoa(p.ImageHeight)}, "x")

	//add values to the 'world' 2D slice
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			val := <-c.ioInput
			if val != 0 {
				world[y][x] = val
			}
		}
	}

	turn := 0
	ticker := time.NewTicker(2 * time.Second) //create a new ticker
	var aliveCells []util.Cell
	updateWorld := createSlice(p, p.ImageHeight)
	// TODO: Execute all turns of the Game of Life.

	if p.Turns != 0 {
		for t := 0; t < p.Turns; t++ {

			//SDL logic
			select {
			case k := <-keyChan: //this bit will take en the key presses and do what it's supposed to do
				if k == 's' {
					outputFileToPGM(p, c, world, turn)
				} else if k == 'q' {
					outputFileToPGM(p, c, world, turn)
					return
				} else if k == 'p' {
					fmt.Printf("Current turn : %d \n", turn)
					for {
						kp := <-keyChan
						if kp == 'p' {
							fmt.Println("Continuing....")
							break
						}
					}
				}

			//AliveCell logic
			case <-ticker.C: //this bit will update AliveCellCount every 2 seconds
				alive := 0
				alive += countAliveCells(p, world)
				if turn != 0 {
					c.events <- AliveCellsCount{turn, alive}
				} else {
					break
				}
			default:
				break
			}

			//BASELINE GOL LOGIC
			var wg = sync.WaitGroup{} //used to make sure all goroutines have done executing before resuming
			start := time.Now()
			for i := 0; i < p.Threads; i++ { //for each thread make the worker work??
				wg.Add(1) //add number of threads the wait group needs to wait
				go worker(p, c, world, updateWorld, i, workerHeight, turn, &aliveCells, &wg)
			}

			wg.Wait() //wait till all goroutines is done (wg == 0)
			elapsed := time.Since(start)
			fmt.Printf("time : %s, turn : %d @ C=%d\n", elapsed, turn, p.Threads)
			turn = t + 1
			c.events <- TurnComplete{turn}

			//update the 2D world slice
			tmp := world
			world = updateWorld
			updateWorld = tmp
			//wg.Wait()
		}
	}

	//after all turn complete, output world as pgm file
	if turn == p.Turns {
		outputFileToPGM(p, c, world, turn)
	}

	// TODO: Report the final state using FinalTurnCompleteEvent.

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
