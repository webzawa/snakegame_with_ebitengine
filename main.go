package main

import (
	"fmt"
	"image/color"
	"log"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	screenWidth  = 640
	screenHeight = 480
	gridSize     = 20
	moveInterval = 8

	columns = screenWidth / gridSize
	rows    = screenHeight / gridSize
)

const (
	dirUp = iota
	dirDown
	dirLeft
	dirRight
)

type Point struct {
	X, Y int
}

var dirDelta = map[int]Point{
	dirUp:    {0, -1},
	dirDown:  {0, 1},
	dirLeft:  {-1, 0},
	dirRight: {1, 0},
}

type Game struct {
	snake     []Point
	direction int
	nextDir   int
	food      Point
	score     int
	gameOver  bool
	tickCount int
}

func NewGame() *Game {
	g := &Game{
		direction: dirRight,
		nextDir:   dirRight,
	}
	centerX := columns / 2
	centerY := rows / 2
	for i := 0; i < 3; i++ {
		g.snake = append(g.snake, Point{X: centerX - i, Y: centerY})
	}
	g.spawnFood()
	return g
}

func (g *Game) spawnFood() {
	for {
		g.food = Point{
			X: rand.Intn(columns),
			Y: rand.Intn(rows),
		}
		overlap := false
		for _, p := range g.snake {
			if p == g.food {
				overlap = true
				break
			}
		}
		if !overlap {
			return
		}
	}
}

func (g *Game) Update() error {
	if g.gameOver {
		if ebiten.IsKeyPressed(ebiten.KeyEnter) || ebiten.IsKeyPressed(ebiten.KeySpace) {
			*g = *NewGame()
		}
		return nil
	}

	// Input
	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) && g.direction != dirDown {
		g.nextDir = dirUp
	} else if ebiten.IsKeyPressed(ebiten.KeyArrowDown) && g.direction != dirUp {
		g.nextDir = dirDown
	} else if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) && g.direction != dirRight {
		g.nextDir = dirLeft
	} else if ebiten.IsKeyPressed(ebiten.KeyArrowRight) && g.direction != dirLeft {
		g.nextDir = dirRight
	}

	// Tick
	g.tickCount++
	if g.tickCount < moveInterval {
		return nil
	}
	g.tickCount = 0

	// Move
	g.direction = g.nextDir
	delta := dirDelta[g.direction]
	newHead := Point{
		X: g.snake[0].X + delta.X,
		Y: g.snake[0].Y + delta.Y,
	}

	// Wall collision
	if newHead.X < 0 || newHead.X >= columns || newHead.Y < 0 || newHead.Y >= rows {
		g.gameOver = true
		return nil
	}

	// Self collision
	for _, p := range g.snake {
		if p == newHead {
			g.gameOver = true
			return nil
		}
	}

	// Add new head
	g.snake = append([]Point{newHead}, g.snake...)

	// Eat food or trim tail
	if newHead == g.food {
		g.score++
		g.spawnFood()
	} else {
		g.snake = g.snake[:len(g.snake)-1]
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0, 0, 0, 255})

	// Draw snake
	for _, p := range g.snake {
		ebitenutil.DrawRect(screen,
			float64(p.X*gridSize), float64(p.Y*gridSize),
			float64(gridSize-1), float64(gridSize-1),
			color.RGBA{0, 220, 0, 255})
	}

	// Draw food
	ebitenutil.DrawRect(screen,
		float64(g.food.X*gridSize), float64(g.food.Y*gridSize),
		float64(gridSize-1), float64(gridSize-1),
		color.RGBA{220, 0, 0, 255})

	// Draw score
	ebitenutil.DebugPrint(screen, fmt.Sprintf("Score: %d", g.score))

	// Game over
	if g.gameOver {
		ebitenutil.DebugPrintAt(screen, "GAME OVER", screenWidth/2-30, screenHeight/2-10)
		ebitenutil.DebugPrintAt(screen, "Press Enter or Space to restart", screenWidth/2-95, screenHeight/2+10)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Snake Game")
	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
}
