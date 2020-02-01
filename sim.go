package main

import (
	"image/color"
	"math/rand"
)

// CELLSIZE is the radius of each cell
var CELLSIZE = 10

// MASKARRAY is an array of masks used to replace the traits
var MASKARRAY []int = []int{0xFFFFF0, 0xFFFF0F, 0xFFF0FF, 0xFF0FFF, 0xF0FFFF, 0x0FFFFF}

// Cell is a representation of a cell within the grid
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

// create the initial population
func createPopulation() {
	cells = make([]Cell, *width*(*width))
	n := 0
	for i := 1; i <= *width; i++ {
		for j := 1; j <= *width; j++ {
			p := rand.Float64()
			if p < *coverage {
				cells[n] = createCell(i*CELLSIZE, j*CELLSIZE, rand.Intn(0xFFFFFF))
			} else {
				cells[n] = createCell(i*CELLSIZE, j*CELLSIZE, 0x000000)
			}
			n++
		}
	}
	fdistances, changes, uniques = []string{"distance"}, []string{"change"}, []string{"unique"}
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
			if cells[neighbour].getRGB() != 0x0000 {
				count++
				dist = dist + featureDistance(cells[c].getRGB(), cells[neighbour].getRGB())
			}
		}
	}
	return int(float64(dist / *width) * (*coverage))
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
