package main

import (
	"bytes"
	"encoding/base64"
	"encoding/csv"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/llgcode/draw2d/draw2dimg"
	"github.com/nsf/termbox-go"
)

// WIDTH is the number of cells on one side of the image
var WIDTH = 36

// CELLSIZE is the radius of each cell
var CELLSIZE = 10

// MASKARRAY is an array of masks used to replace the traits
var MASKARRAY []int = []int{0xFFFFF0, 0xFFFF0F, 0xFFF0FF, 0xFF0FFF, 0xF0FFFF, 0x0FFFFF}

var img *image.RGBA
var cells []Cell

var fdistances []string
var changes []string
var uniques []string

// Cell is a representation of a cell
type Cell struct {
	X     int
	Y     int
	R     int
	Color color.Color
}

// get the color integer back from the cell in the form 0x1A2B3C
func (c *Cell) getRGB() int {
	r, g, b, _ := c.Color.RGBA()
	return int((r & 0x00FF << 16) + (g & 0x00FF << 8) + b&0x00FF)
}

// set the color using the color interger in the form 0x1A2B3C
func (c *Cell) setRGB(i int) {
	c.Color = color.RGBA{getR(i), getG(i), getB(i), uint8(255)}
}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	n := flag.Int("n", 100, "number of interactions between cultures per simulation tick")
	flag.Parse()

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
	for !endSim {
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
		for c := 0; c < *n; c++ {
			r := rand.Intn(WIDTH * WIDTH)
			neighbours := findNeighboursIndex(r)
			for _, neighbour := range neighbours {
				d := diff(r, neighbour)
				probability := 1 - float64(d)/96.0
				dp := rand.Float64()
				if dp < probability {
					i := rand.Intn(6)
					if d != 0 {
						var rp int
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
			dist = featureDistAvg()
			uniq = similarCount()
		}

		img = draw(WIDTH*CELLSIZE+CELLSIZE, WIDTH*CELLSIZE+CELLSIZE, cells)
		printImage(img.SubImage(img.Rect))
		fmt.Println("\nNumber of cultural interactions per simulation tick:", *n)
		fmt.Println("\naverage distance between cultures:", dist,
			"\nnumber of unique cultures        :", uniq,
			"\nnumber of cultural exchanges     :", chg)
		fmt.Println("\nCtrl-Q to quit simulation and log data.")
		fdistances = append(fdistances, strconv.Itoa(dist))
		changes = append(changes, strconv.Itoa(chg/WIDTH))
		uniques = append(uniques, strconv.Itoa(uniq))
	}
	termbox.Close()
	t := time.Now()
	fileid := t.Format("20060102150405")
	writeLog(*n, fileid)
	fmt.Printf("Simulation ended, data written to lo-%d-%s.csv.\n", *n, fileid)
}

func writeLog(n int, fileid string) {
	data := [][]string{
		fdistances,
		changes,
		uniques}
	csvfile, err := os.Create(fmt.Sprintf("log-%d-%s.csv", n, fileid))

	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}

	csvwriter := csv.NewWriter(csvfile)

	for _, line := range data {
		_ = csvwriter.Write([]string(line))
	}
	csvwriter.Flush()
	csvfile.Close()

}

// create the initial population
func createPopulation() {
	cells = make([]Cell, WIDTH*WIDTH)
	n := 0
	for i := 1; i <= WIDTH; i++ {
		for j := 1; j <= WIDTH; j++ {
			cells[n] = createCell(i*CELLSIZE, j*CELLSIZE, rand.Intn(0xFFFFFF))
			n++
		}
	}
	fdistances, changes, uniques = []string{"distance"}, []string{"change"}, []string{"unique"}
}

// create a cell
func createCell(x, y, clr int) (c Cell) {
	c = Cell{
		X:     x,
		Y:     y,
		R:     CELLSIZE, // radius of cell
		Color: color.RGBA{getR(clr), getG(clr), getB(clr), uint8(255)},
	}
	return
}

// total distance between traits for all features, between 2 cultures
func diff(a1, a2 int) int {
	var d int
	for i := 0; i < 5; i++ {
		d = d + traitDistance(cells[a1].getRGB(), cells[a2].getRGB(), uint(i))
	}
	return d
}

// average feature distance for the whole grid
func featureDistAvg() int {
	var count int
	var dist int
	for c := range cells {
		neighbours := findNeighboursIndex(c)
		for _, neighbour := range neighbours {
			count++
			dist = dist + featureDistance(cells[c].getRGB(), cells[neighbour].getRGB())
		}
	}
	return dist / WIDTH
}

// distance between 2 features
func featureDistance(n1, n2 int) int {
	var features int = 0
	for i := 0; i < 5; i++ {
		f1, f2 := extract(n1, uint(i)), extract(n2, uint(i))
		if f1 == f2 {
			features++
		}
	}
	return 6 - features
}

// count unique colors
func similarCount() int {
	uniques := make(map[int]int)
	for _, c := range cells {
		uniques[c.getRGB()] = c.getRGB()
	}
	return len(uniques)
}

// find the distance of 2 numbers at position pos
func traitDistance(n1, n2 int, pos uint) int {
	d := extract(n1, pos) - extract(n2, pos)
	if d < 0 {
		return d * -1
	}
	return d
}

// extract trait for 1 feature
func extract(n int, pos uint) int {
	return (n >> (4 * pos)) & 0x00000F
}

// replace the trait in 1 feature
func replace(n, replacement int, pos uint) int {
	i1 := n & MASKARRAY[pos]
	mask2 := replacement << (4 * pos)
	return (i1 ^ mask2)
}

// the color integer is 0x1A2B3CFF where
// 1A is the red, 2B is green and 3C is blue

// get the red (R) from the color integer i
func getR(i int) uint8 {
	return uint8((i >> 16) & 0x0000FF)
}

// get the green (G) from the color integer i
func getG(i int) uint8 {
	return uint8((i >> 8) & 0x0000FF)
}

// get the blue (B) from the color integer i
func getB(i int) uint8 {
	return uint8(i & 0x0000FF)
}

// draw the cells
func draw(w int, h int, cells []Cell) *image.RGBA {
	dest := image.NewRGBA(image.Rect(0, 0, w, h))
	gc := draw2dimg.NewGraphicContext(dest)
	for _, cell := range cells {
		gc.SetFillColor(cell.Color)
		gc.MoveTo(float64(cell.X), float64(cell.Y))
		gc.ArcTo(float64(cell.X), float64(cell.Y),
			float64(cell.R/2), float64(cell.R/2), 0, 6.283185307179586)
		gc.Close()
		gc.Fill()
	}
	return dest
}

// Print the image to iTerm2 terminal
func printImage(img image.Image) {
	var buf bytes.Buffer
	png.Encode(&buf, img)
	imgBase64Str := base64.StdEncoding.EncodeToString(buf.Bytes())
	fmt.Printf("\x1b[2;0H\x1b]1337;File=inline=1:%s\a", imgBase64Str)
}

// Find the indices of the neighbouring cells
func findNeighboursIndex(n int) (nb []int) {
	switch {
	// corner cases
	case topLeft(n):
		nb = append(nb, c5(n))
		nb = append(nb, c7(n))
		nb = append(nb, c8(n))
		return
	case topRight(n):
		nb = append(nb, c4(n))
		nb = append(nb, c6(n))
		nb = append(nb, c7(n))
		return
	case bottomLeft(n):
		nb = append(nb, c2(n))
		nb = append(nb, c3(n))
		nb = append(nb, c5(n))
		return
	case bottomRight(n):
		nb = append(nb, c1(n))
		nb = append(nb, c2(n))
		nb = append(nb, c4(n))
		return
		// side cases
	case top(n):
		nb = append(nb, c4(n))
		nb = append(nb, c5(n))
		nb = append(nb, c6(n))
		nb = append(nb, c7(n))
		nb = append(nb, c8(n))
		return
	case left(n):
		nb = append(nb, c2(n))
		nb = append(nb, c3(n))
		nb = append(nb, c5(n))
		nb = append(nb, c7(n))
		nb = append(nb, c8(n))
		return
	case right(n):
		nb = append(nb, c1(n))
		nb = append(nb, c2(n))
		nb = append(nb, c4(n))
		nb = append(nb, c6(n))
		nb = append(nb, c7(n))
		return
	case bottom(n):
		nb = append(nb, c1(n))
		nb = append(nb, c2(n))
		nb = append(nb, c3(n))
		nb = append(nb, c4(n))
		nb = append(nb, c5(n))
		return
		// everything else
	default:
		nb = append(nb, c1(n))
		nb = append(nb, c2(n))
		nb = append(nb, c3(n))
		nb = append(nb, c4(n))
		nb = append(nb, c5(n))
		nb = append(nb, c6(n))
		nb = append(nb, c7(n))
		nb = append(nb, c8(n))
	}
	return
}

// functions to check for corners and sides
func topLeft(n int) bool     { return n == 0 }
func topRight(n int) bool    { return n == WIDTH-1 }
func bottomLeft(n int) bool  { return n == WIDTH*(WIDTH-1) }
func bottomRight(n int) bool { return n == (WIDTH*WIDTH)-1 }

func top(n int) bool    { return n < WIDTH }
func left(n int) bool   { return n%WIDTH == 0 }
func right(n int) bool  { return n%WIDTH == WIDTH-1 }
func bottom(n int) bool { return n >= WIDTH*(WIDTH-1) }

// functions to get the index of the neighbours
func c1(n int) int { return n - WIDTH - 1 }
func c2(n int) int { return n - WIDTH }
func c3(n int) int { return n - WIDTH + 1 }
func c4(n int) int { return n - 1 }
func c5(n int) int { return n + 1 }
func c6(n int) int { return n + WIDTH - 1 }
func c7(n int) int { return n + WIDTH }
func c8(n int) int { return n + WIDTH + 1 }
