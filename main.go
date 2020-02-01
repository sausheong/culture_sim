package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"image"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/nsf/termbox-go"
)

// image shown on the screen
var img *image.RGBA

// the simulation grid
var cells []Cell

// the number of cells on one side of the image
var width *int

// number of interactions between cultures per simulation tick
var interactions *int

// percentage of simulation grid that is populated with cultures
var coverage *float64

// number of simulation ticks
var numTicks *int

// simulation data
var fdistances []string // average distance between features
var changes []string    // number of cultural changes
var uniques []string    // number of unique cultures

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	// capture the simulation parameters
	interactions = flag.Int("n", 100, "number of interactions between cultures per simulation tick")
	numTicks = flag.Int("t", 200, "number of simulation ticks")
	width = flag.Int("w", 36, "the number of cells on one side of the image")
	coverage = flag.Float64("c", 1.0, "percentage of simulation grid that is populated with cultures")
	flag.Parse()

	// using termbox to control the simulation
	termbox.Init()
	endSim := false

	// poll for keyboard events in another goroutine
	events := make(chan termbox.Event, 1000)
	go func() {
		for {
			events <- termbox.PollEvent()
		}
	}()

	// create the initial population
	createPopulation()

	// main simulation loop
	for t := 0; !endSim && (t < *numTicks); t++ {
		// data captured for every loop of simulation
		var dist, chg, uniq int

		// capture the ctrl-q key to end the simulation
		select {
		case ev := <-events:
			if ev.Type == termbox.EventKey {
				if ev.Key == termbox.KeyCtrlQ {
					endSim = true
				}
			}
		default:
		}

		// every simulation loop randomly pick a number of cells and
		// get them to have cultural exchange with their neighbours depending
		// the calculated probability. The more similar the cultures are, the
		// more likely there will be cultural exchange
		for c := 0; c < *interactions; c++ {
			// randomly choose one cell
			r := rand.Intn(*width * *width)
			if cells[r].getRGB() != 0x0000 {
				// find all its neighbours
				neighbours := findNeighboursIndex(r)
				for _, neighbour := range neighbours {
					if cells[neighbour].getRGB() != 0x0000 {
						// cultural differences between the neighbour
						d := diff(r, neighbour)
						// probability of a cultural exchange happening
						probability := 1 - float64(d)/96.0
						dp := rand.Float64()
						// cultural exchange happens
						if dp < probability {
							// randomly select one of the features
							i := rand.Intn(6)
							if d != 0 {
								var rp int
								// randomly select either trait to be replaced by the neighbour's
								if rand.Intn(1) == 0 {
									replacement := extract(cells[r].getRGB(), uint(i))
									rp = replace(cells[neighbour].getRGB(), replacement, uint(i))
								} else {
									replacement := extract(cells[neighbour].getRGB(), uint(i))
									rp = replace(cells[r].getRGB(), replacement, uint(i))
								}
								cells[neighbour].setRGB(rp)
								chg++
							}
						}

					}
				}
			}

			// calculate the average distance between all features and the number of unique cultures
			dist = featureDistAvg()
			uniq = similarCount()
		}

		img = draw(*width*CELLSIZE+CELLSIZE, *width*CELLSIZE+CELLSIZE, cells)
		printImage(img.SubImage(img.Rect))
		fmt.Println("\nNumber of cultural interactions per simulation tick:", *interactions)
		fmt.Printf("Simulation ticks: %d/%d", t, *numTicks)
		fmt.Printf("\nSimulation coverage: %2.0f%%", *coverage*100)

		fmt.Println("\n\naverage distance between cultures:", dist,
			"\nnumber of unique cultures        :", uniq,
			"\nnumber of cultural exchanges     :", chg)
		fmt.Println("\nCtrl-Q to quit simulation and save data.")
		fdistances = append(fdistances, strconv.Itoa(dist))
		changes = append(changes, strconv.Itoa(chg/(*width)))
		uniques = append(uniques, strconv.Itoa(uniq))
	}
	termbox.Close()

	simName := fmt.Sprintf("n%d-t%d-w%d-c%1.1f", *interactions, *numTicks, *width, *coverage)
	saveData(simName)
	fmt.Printf("Simulation ended.\n"+"Data written to log-%s.csv \nLast grid saved to"+
		" cells-%s.csv \nLast image saved to %s.png\n",
		simName, simName, simName)
}

// save simulation data
func saveData(name string) {
	// simulation data
	data := [][]string{
		fdistances, // average feature distance
		changes,    // number of changes
		uniques}    // number of unique cultures
	csvfile, err := os.Create(fmt.Sprintf("data/log-%s.csv", name))
	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}
	csvwriter := csv.NewWriter(csvfile)

	for _, line := range data {
		_ = csvwriter.Write([]string(line))
	}
	csvwriter.Flush()
	csvfile.Close()

	// snapshot of grid at the end of the simulation
	grid := make(map[int]int)
	for _, c := range cells {
		grid[c.getRGB()]++
	}
	cellsfile, err := os.Create(fmt.Sprintf("data/cell-%s.csv", name))
	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}
	csvwriter = csv.NewWriter(cellsfile)
	for k, v := range grid {
		_ = csvwriter.Write([]string{strconv.Itoa(k), strconv.Itoa(v)})
	}
	csvwriter.Flush()
	csvfile.Close()

	// save the last image of the grid
	saveImage("data/"+name+".png", img)
}
