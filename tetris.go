package main

import (
	"fmt"
	//gc "github.com/rthornton128/goncurses"
    gc  "github.com/RickBadertscher/goncurses"
	"github.com/skelterjohn/geom"
	"log"
	"os"
	"time"
	"sync"
	"bytes"
	"sort"
	"math/rand"
)

type ShapeType int

type Orientation int

const (
	_ Orientation = iota
	North
	South
	East
	West
)

type TetriminoType int
const (
	O TetriminoType = 0
	I TetriminoType = 1
	T TetriminoType = 2
	J TetriminoType = 3
	L TetriminoType = 4
	Z TetriminoType = 5
	S TetriminoType = 6
)

func (o* Orientation) RotateRight() (Orientation) {
	switch *o {
	case North:
		return East
	case East:
		return South
	case South:
		return West
	case West:
		return North
	}
	return North
}
func (o* Orientation) RotateLeft() (Orientation) {
	switch *o {
	case North:
		return West
	case East:
		return North
	case South:
		return East
	case West:
		return South
	}
	return North
}

type Game struct {
	top, bottom, right, left   int
	shapes      []*Shape
	fallingShape *Shape
	win         *gc.Window
	shapeWin    *gc.Window
	floor       *Fill
	mutex		*sync.Mutex
}
type Shape struct {
	win		*gc.Window
	tetriminoTemplate *Tetrimino
	path     *Path
	orientation Orientation
	poly     *geom.Polygon
	x  		 int
	y 		 int
	w   	 int
	h        int
}

type Coord struct {
	x int
	y int
}

func (c Coord) translate(x, y int) {
	c.x = c.x + x;
	c.y = c.y + y;
}

func (c Coord) translateCopy(x, y int) (Coord){
	return Coord{c.x+x, c.y+y}
}

type Path struct {
	points []Coord
}

func (p* Path) String() string {
	var b bytes.Buffer // A Buffer needs no initialization.
	for _, c := range p.points {
		fmt.Fprintf(&b, "{%v,%v}", c.x, c.y);
	}
	return b.String()
}

func (p *Path) translateCopy(x, y int) (*Path){
	newpath := new(Path)
	for i, _ := range p.points {
		newpath.points = append(newpath.points, p.points[i].translateCopy(x, y))
	}
	return newpath
}


func (p *Path) translate(x, y int) {
	for i, _ := range p.points {
		p.points[i].x = p.points[i].x + x
		p.points[i].y = p.points[i].y + y
	}

}

func (p* Path) getRightEdge() (int) {

	var edge int = 0
	for _,c := range p.points {

		if c.x > edge {
			edge = c.x
		}
	}

	return edge
}


func (p* Path) getLeftEdge() (int) {

	var edge int = 100
	for _,c := range p.points {

		if c.x < edge {
			edge = c.x
		}
	}
	return edge
}


type IntSet struct {
	slice []int
}

func (i *IntSet) Add(v int) {
	if i.Contains(v) {
		return
	}
	i.slice = append(i.slice, v)
}
func (i *IntSet) Contains(v int) (bool) {
	for _,val := range i.slice {
		if val == v {
			return true
		}
	}
	return false
}
func NewIntSet() (*IntSet) {
	var i *IntSet = new(IntSet)
	i.slice = make([]int, 0)
	return i
}
type Fill struct {
	rows    map[int]*IntSet
	bottomRow int
}

func NewFill(bottomRow int) (*Fill) {
	f := new(Fill)
	f.bottomRow = bottomRow
	f.rows = make(map[int]*IntSet)
	return f
}

func (f *Fill) compress() {

    keys := make([]int, 0, len(f.rows))
	for k := range f.rows {
		keys = append(keys, k)
	}

	sort.Sort(sort.Reverse(sort.IntSlice(keys)))

	// now we have reverse sorted by rows:   example   20, 19, 17, 14

	currentRow := f.bottomRow
	newmap := make(map[int]*IntSet)

	for i := 0; i < len(keys); i++ {
		clog("setting row %v to new row %v\n", keys[i], currentRow)
		newmap[currentRow] = f.rows[keys[i]]
		currentRow -= 1
	}

	f.rows = newmap

}



func (f* Fill) removeRow(row int) {
	clog("removing row from floor: %v\n", row)
	delete(f.rows, row)
}

func (f* Fill) addRow(y int, width int) {
	if f.rows[y] == nil {
		f.rows[y] = NewIntSet()
	}
	for i := 0; i < width; i++ {
		f.rows[y].Add(i)
	}
}

func (f* Fill) addCoords(coords[] Coord) {
	for _,c := range coords {
		if f.rows[c.y] == nil {
			f.rows[c.y] = NewIntSet()
		}

		f.rows[c.y].Add(c.x)
	}
}

func (f* Fill) String() string {
	var b bytes.Buffer
	for k, _ := range f.rows {
		fmt.Fprintf(&b, "%v : %v\n", k, *(f.rows[k]))
	}
	return b.String()
}

func (f* Fill) intersects(other* Path) (bool) {

	for _, c := range other.points {
		iset := f.rows[c.y]
		if iset != nil && iset.Contains(c.x) {
			return true
		}
	}
	return false;
}



type Tetrimino struct {
	H int
	W int
	North 	[]Coord
	East 	[]Coord
	South 	[]Coord
	West 	[]Coord



}

var tetriminos map[TetriminoType]*Tetrimino

func init() {

	tetriminos = make(map[TetriminoType]*Tetrimino)
	tetriminos[O] = &Tetrimino{2, 2,
		[]Coord{{0,0},{0,1},{1,0},{1,1}},
		[]Coord{{0,0},{0,1},{1,0},{1,1}},
		[]Coord{{0,0},{0,1},{1,0},{1,1}},
		[]Coord{{0,0},{0,1},{1,0},{1,1}}}

	tetriminos[I] = &Tetrimino{4, 4,
		[]Coord{{3,0},{3,1},{3,2},{3,3}},
		[]Coord{{0,1},{1,1},{2,1},{3,1}},
		[]Coord{{3,0},{3,1},{3,2},{3,3}},
		[]Coord{{0,2},{1,2},{2,2},{3,2}}}

	tetriminos[T] = &Tetrimino{3, 3,
		[]Coord{{1,1},{0,2},{1,2},{2,2}},
		[]Coord{{0,1},{1,0},{1,1},{1,2}},
		[]Coord{{0,1},{1,1},{2,1},{1,2}},
		[]Coord{{2,1},{1,0},{1,1},{1,2}}}

	tetriminos[J] = &Tetrimino{3, 3,
		[]Coord{{0,1},{1,1},{2,1},{2,2}},
		[]Coord{{1,0},{1,1},{1,2},{0,2}},
		[]Coord{{0,1},{0,2},{1,2},{2,2}},
		[]Coord{{1,0},{2,0},{1,1},{1,2}}}

	tetriminos[L] = &Tetrimino{3, 3,
		[]Coord{{0,1},{1,1},{2,1},{0,2}},
		[]Coord{{0,0},{1,0},{1,1},{2,1}},
		[]Coord{{0,2},{1,2},{2,2},{2,1}},
		[]Coord{{1,0},{1,1},{1,2},{2,2}}}

	tetriminos[Z] = &Tetrimino{3, 3,
		[]Coord{{0,1},{1,1},{1,2},{2,2}},
		[]Coord{{2,0},{1,1},{2,1},{1,2}},
		[]Coord{{0,1},{1,1},{1,2},{2,2}},
		[]Coord{{1,0},{0,1},{1,1},{0,2}}}

	tetriminos[S] = &Tetrimino{3, 3,
		[]Coord{{0,2},{1,1},{1,2},{2,1}},
		[]Coord{{1,0},{1,1},{2,1},{2,2}},
		[]Coord{{0,2},{1,1},{1,2},{2,1}},
		[]Coord{{0,0},{0,1},{1,1},{1,2}}}


}

func (t *Tetrimino) draw(w *gc.Window, o Orientation) {

	var p []Coord
	switch o {
	case North:
		p = t.North
	case East:
		p = t.East
	case West:
		p = t.West
	case South:
		p = t.South

	}
	for _, p := range p {
		w.MoveAddChar(p.y, p.x, '*')
	}

}

func NewShape(tetrimino TetriminoType, win *gc.Window) *Shape {
	s := new(Shape)
	s.x = 5
	s.y = 0
	s.tetriminoTemplate = tetriminos[tetrimino]
	s.orientation = North

	s.w = s.tetriminoTemplate.W
	s.h = s.tetriminoTemplate.H

	s.win = win
	s.win.Erase()
	s.win.Resize(s.w, s.h)
	s.win.MoveWindow(s.y, s.x)
	s.tetriminoTemplate.draw(s.win, s.orientation)

	s.updatePath()

	return s
}

func (shape *Shape) getTopEdge() (int) {

	var edge int = 100
	for _,c := range shape.path.points {

		if c.y < edge {
			edge = c.y
		}
	}
	return edge
}


func (shape *Shape) move(x int, y int) {

	clog("shape.move by: %v, %v, new location = {%v,%v}\n", x, y, shape.x+x, shape.y+y)
	shape.win.MoveWindow(shape.y + y, shape.x + x)

	shape.x += x
	shape.y += y

	shape.path.translate(x, y)



}


func (shape* Shape) updatePath() (*Path) {

	var p *Path = new(Path)

	switch(shape.orientation) {
	case North:
		p.points = make([]Coord, len(shape.tetriminoTemplate.North))
		copy(p.points, shape.tetriminoTemplate.North)
	case East:
		p.points = make([]Coord, len(shape.tetriminoTemplate.East))
		copy(p.points, shape.tetriminoTemplate.East)
	case West:
		p.points = make([]Coord, len(shape.tetriminoTemplate.West))
		copy(p.points, shape.tetriminoTemplate.West)
	case South:
		p.points = make([]Coord, len(shape.tetriminoTemplate.South))
		copy(p.points, shape.tetriminoTemplate.South)

	}

	p.translate(shape.x, shape.y)
	shape.path = p


	return p
}

func (shape *Shape) draw(parent *gc.Window) {
	parent.Overlay(shape.win)

}
func (shape* Shape) rotateLeft() {

	shape.orientation = shape.orientation.RotateLeft()
	shape.win.Clear()
	shape.tetriminoTemplate.draw(shape.win, shape.orientation)
	shape.path = shape.updatePath()


}
func (shape* Shape) rotateRight() {
	shape.orientation = shape.orientation.RotateRight()
	shape.win.Clear()
	shape.tetriminoTemplate.draw(shape.win, shape.orientation)
	shape.path = shape.updatePath()

}


var stdscr *gc.Window
func main() {

	f, err := os.OpenFile("testlogfile", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("error opening file: %v", err)
	}
	defer f.Close()

	log.SetOutput(f)
	log.Println("This is a test log entry")

	stdscr, err = gc.Init()
	stdscr.Resize(12, 12)


	game := NewGame(stdscr)

	quitchannel := make(chan int)
	go game.gameloop()

	select {
	case <- quitchannel:
		clog("exiting)")
	}


}

func (game *Game) checkFloor() {
	clog("check floor: %v\n", game.floor)

	var clearrows []int = make([]int, 0)
	for k, v :=  range game.floor.rows {
	    if len(v.slice) == (game.right - game.left - 1) {
			clearrows = append(clearrows, k)
		}
	}

	for _, v := range clearrows {

		game.win.AttrOn(gc.A_BLINK)
		game.win.HLine(v, 0, '+', game.right - game.left)

		game.win.Refresh()
		gc.Update()
		game.win.AttrOff(gc.A_BLINK)
		gc.Nap(2000);


	}

	for _, v := range clearrows {
		game.floor.removeRow(v)
	}

	if len(clearrows) > 0 {
		game.floor.compress()
	}

}
func (game *Game) MoveFallingShape(x int, y int) bool {


	if (x != 0) {
		tmppath := game.fallingShape.path.translateCopy(x, 0)

		var edge int
		if x > 0 {
			edge = tmppath.getRightEdge()
		} else {
			edge = tmppath.getLeftEdge()
		}
		if edge >= game.right || edge <= game.left {
			clog("cant move any more left or right\n")
			return true
		}
		if (game.floor.intersects(tmppath)) {
			clog("shape hit the stack from the side\n")
			return true
		}

		game.fallingShape.move(x, 0)
		return true

	} else if (y > 0) {
		tmppath := game.fallingShape.path.translateCopy(0,y)

		if game.hitBottom(tmppath) {
			game.floor.addCoords(game.fallingShape.path.points);
			game.checkFloor()
			if game.dropShape() == false {
				return false;
			}
			return true
		}
		game.fallingShape.move(0, y)
		return true

	}


	return true

}


func (game *Game) hitBottom(p *Path) bool {

	for _,c := range p.points {
		if c.y  >= game.bottom {
			clog("shape hit the bottom at %d\n", c.y)
			return true
		}
	}

	if (game.floor.intersects(p)) {
		clog("shape hit the stack\n")
		return true
	}
	return false
}

func NewGame(parent *gc.Window) *Game {

	game := new(Game)
	h, w := parent.MaxYX()


	clog("newGame:  h,w = %v,%v\n", h, w)
	game.top	= 0
	game.bottom = h - 1
	game.left   = 0
	game.right  = w - 1


	game.floor = NewFill(game.bottom - 1)
	game.win,_ = gc.NewWindow(h, w, 0, 0)

	game.shapeWin, _= gc.NewWindow(1, 1, 1, 5)

	y, x := game.win.YX()
	h, w = game.win.MaxYX()
	return game

	}
func (game *Game) draw(win *gc.Window) {

	game.win.Erase()
	game.win.Border('|','|', '-', '-', '+','+','+','+')


	for k, v := range game.floor.rows {
		for _, x := range v.slice {
			game.win.MoveAddChar(k, x, '+')

		}
	}

	game.fallingShape.draw(game.win)

}

func (game *Game) dropShape() (bool){


	r := rand.Intn(6);
	game.fallingShape = NewShape(TetriminoType(r), game.shapeWin)

	if game.fallingShape.getTopEdge() < 0  ||
	   game.hitBottom(game.fallingShape.path) {

		return false
	}

	return true


}
func (game *Game) gameloop() {

	var err error

	game.mutex = &sync.Mutex{}

	if err != nil {
		log.Fatal(err)
	}
	gc.Echo(false)
	gc.CBreak(true)
	gc.Cursor(0)
	gc.TypeAhead(-1)
	defer gc.End()

	c := time.NewTicker(time.Second * 1)

	updates := make(chan int)

	game.dropShape()

	loop:

		for {

			game.draw(stdscr)
			game.win.Refresh()

			go game.handleInput(updates)

		select {


			case <-updates:
			    clog("update\n")
				game.win.Erase()
				game.draw(stdscr)


			case <-c.C:
					if game.MoveFallingShape(0, 1) == false {
						break loop
					}


			}


	}

	game.win.MovePrintf(0, 5, "Game over")

}

func (game *Game) handleInput(updates chan int)  {
	game.win.Timeout(1000)
	for ;; {

		k := game.win.GetChar()

		switch k {
		case 44:
			game.MoveFallingShape(-1, 0)

			updates <- 1

		case 46:
			game.MoveFallingShape(1, 0)

			updates <- 1

		case 122:
			game.fallingShape.rotateLeft()

			updates <- 1

		case 120:
			game.fallingShape.rotateRight()

			updates <- 1

		case 32:
			game.MoveFallingShape(0, 1)

			updates <- 1


		}
	}

}

var logMutex *sync.Mutex;

func init() {
	logMutex = new(sync.Mutex)
}

func clog (s string, args ...interface{}) {
	logMutex.Lock() 
		log.Printf(s, args...)
	logMutex.Unlock()

}