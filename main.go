// スネークゲーム - Ebitengineを使ったクトゥルフ風スネークゲーム
//
// ゲームの流れ:
//
//  1. タイトル画面で難易度を選択（Easy / Normal / Hard）
//  2. 触手の蛇が画面中央からスタートし、自動的に移動する
//  3. プレイヤーは矢印キー or スワイプで蛇の方向を操作する
//  4. 食べ物（ネクロノミコン・脳・エルダーサイン）を食べるとスコアが増え、蛇が伸びる
//  5. 壁や自分の体にぶつかるとゲームオーバー
//  6. タイトル画面に戻って再挑戦
package main

import (
	"bytes"
	"embed"
	"fmt"
	"image/color"
	"image/png"
	"io"
	"log"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ──────────────────────────────────────────────
// アセットの埋め込み
// ──────────────────────────────────────────────

//go:embed assets/*.png assets/*.mp3
var assetsFS embed.FS

// ──────────────────────────────────────────────
// 定数
// ──────────────────────────────────────────────

const (
	screenWidth  = 640
	screenHeight = 480
	gridSize     = 32
	columns      = screenWidth / gridSize  // 20
	rows         = screenHeight / gridSize // 15
	spriteSize   = 128
	sampleRate   = 48000
	swipeThreshold = 16
)

// シーン
const (
	sceneTitle    = iota // タイトル画面
	scenePlaying         // プレイ中
	sceneGameOver        // ゲームオーバー
)

// 難易度
const (
	diffEasy   = iota
	diffNormal
	diffHard
)

// 方向
const (
	dirUp = iota
	dirDown
	dirLeft
	dirRight
)

// 難易度ごとの移動間隔（フレーム数）。値が小さいほど速い。
var difficultySpeed = [3]int{20, 12, 7}
var difficultyName = [3]string{"EASY", "NORMAL", "HARD"}
var difficultyColor = [3]color.RGBA{
	{0, 180, 0, 255},   // Easy: 緑
	{200, 180, 0, 255}, // Normal: 黄
	{220, 40, 40, 255}, // Hard: 赤
}

// ──────────────────────────────────────────────
// 型
// ──────────────────────────────────────────────

type Point struct {
	X, Y int
}

var dirDelta = map[int]Point{
	dirUp: {0, -1}, dirDown: {0, 1}, dirLeft: {-1, 0}, dirRight: {1, 0},
}

// ──────────────────────────────────────────────
// スプライト・オーディオ（パッケージレベル変数）
// ──────────────────────────────────────────────

var (
	backgroundImg    *ebiten.Image
	headSprites      map[int]*ebiten.Image
	tailSprites      map[int]*ebiten.Image
	bodyVertical     *ebiten.Image
	bodyHorizontal   *ebiten.Image
	curveTopLeft     *ebiten.Image
	curveTopRight    *ebiten.Image
	curveBottomLeft  *ebiten.Image
	curveBottomRight *ebiten.Image
	foodSprites      [3]*ebiten.Image

	audioCtx       = audio.NewContext(sampleRate)
	bgmPlayer      *audio.Player
	seEatData      []byte
	seGameOverData []byte
)

var spriteScale = float64(gridSize) / float64(spriteSize)

func loadImage(name string) *ebiten.Image {
	f, err := assetsFS.Open("assets/" + name + ".png")
	if err != nil {
		log.Fatalf("画像の読み込みに失敗: %s: %v", name, err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		log.Fatalf("PNGのデコードに失敗: %s: %v", name, err)
	}
	return ebiten.NewImageFromImage(img)
}

func decodeSE(name string) []byte {
	data, err := assetsFS.ReadFile("assets/" + name + ".mp3")
	if err != nil {
		log.Fatalf("SEの読み込みに失敗: %s: %v", name, err)
	}
	stream, err := mp3.DecodeWithoutResampling(bytes.NewReader(data))
	if err != nil {
		log.Fatalf("SEのデコードに失敗: %s: %v", name, err)
	}
	decoded, err := io.ReadAll(stream)
	if err != nil {
		log.Fatalf("SEの読み出しに失敗: %s: %v", name, err)
	}
	return decoded
}

func playSE(data []byte) {
	player, err := audioCtx.NewPlayer(bytes.NewReader(data))
	if err != nil {
		return
	}
	player.SetVolume(0.5)
	player.Play()
}

func initBGM() {
	bgmData, err := assetsFS.ReadFile("assets/bgm.mp3")
	if err != nil {
		log.Fatalf("BGMの読み込みに失敗: %v", err)
	}
	stream, err := mp3.DecodeWithoutResampling(bytes.NewReader(bgmData))
	if err != nil {
		log.Fatalf("BGMのデコードに失敗: %v", err)
	}
	loop := audio.NewInfiniteLoop(stream, stream.Length())
	bgmPlayer, err = audioCtx.NewPlayer(loop)
	if err != nil {
		log.Fatalf("BGMプレイヤーの作成に失敗: %v", err)
	}
	bgmPlayer.SetVolume(0.3)
	bgmPlayer.Play()
}

func init() {
	backgroundImg = loadImage("background")

	headSprites = map[int]*ebiten.Image{
		dirUp: loadImage("head_up"), dirDown: loadImage("head_down"),
		dirLeft: loadImage("head_left"), dirRight: loadImage("head_right"),
	}
	tailSprites = map[int]*ebiten.Image{
		dirUp: loadImage("tail_up"), dirDown: loadImage("tail_down"),
		dirLeft: loadImage("tail_left"), dirRight: loadImage("tail_right"),
	}

	bodyVertical = loadImage("body_vertical")
	bodyHorizontal = loadImage("body_horizontal")
	curveTopLeft = loadImage("body_segment_curved_top_left")
	curveTopRight = loadImage("body_segment_curved_top_right")
	curveBottomLeft = loadImage("body_segment_curved_bottom_left")
	curveBottomRight = loadImage("body_segment_curved_bottom_right")

	foodSprites[0] = loadImage("food_necronomicon")
	foodSprites[1] = loadImage("food_brain")
	foodSprites[2] = loadImage("food_elder_sign")

	seEatData = decodeSE("se_eat")
	seGameOverData = decodeSE("se_gameover")
	initBGM()
}

// ──────────────────────────────────────────────
// Game 構造体
// ──────────────────────────────────────────────

type Game struct {
	scene      int // sceneTitle, scenePlaying, sceneGameOver
	difficulty int // diffEasy, diffNormal, diffHard

	// プレイ中の状態
	snake     []Point
	direction int
	nextDir   int
	food      Point
	foodType  int
	score     int
	paused    bool
	tickCount int

	// タイトル画面
	selectedDiff int // カーソル位置（0〜2）

	// タッチ操作
	touchID       ebiten.TouchID
	touchStartX   int
	touchStartY   int
	touchTracking bool

	// アニメーション用カウンタ
	frameCount int
}

func NewGame() *Game {
	return &Game{
		scene:        sceneTitle,
		selectedDiff: diffNormal, // デフォルト: Normal
	}
}

func (g *Game) startPlaying() {
	g.scene = scenePlaying
	g.difficulty = g.selectedDiff
	g.paused = false
	g.tickCount = 0
	g.score = 0
	g.direction = dirRight
	g.nextDir = dirRight
	g.snake = nil

	centerX := columns / 2
	centerY := rows / 2
	for i := 0; i < 3; i++ {
		g.snake = append(g.snake, Point{X: centerX - i, Y: centerY})
	}
	g.spawnFood()
}

func (g *Game) spawnFood() {
	g.foodType = rand.Intn(3)
	for {
		g.food = Point{X: rand.Intn(columns), Y: rand.Intn(rows)}
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

// ──────────────────────────────────────────────
// Update
// ──────────────────────────────────────────────

func (g *Game) Update() error {
	g.frameCount++
	g.handleTouch()

	switch g.scene {
	case sceneTitle:
		g.updateTitle()
	case scenePlaying:
		g.updatePlaying()
	case sceneGameOver:
		g.updateGameOver()
	}
	return nil
}

func (g *Game) updateTitle() {
	// キーボードで難易度選択
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		g.selectedDiff = (g.selectedDiff + 2) % 3 // 上に移動
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		g.selectedDiff = (g.selectedDiff + 1) % 3 // 下に移動
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.startPlaying()
		return
	}
	// 数字キーでも選択可能
	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		g.selectedDiff = diffEasy
		g.startPlaying()
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.Key2) {
		g.selectedDiff = diffNormal
		g.startPlaying()
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.Key3) {
		g.selectedDiff = diffHard
		g.startPlaying()
		return
	}

	// タッチでボタンタップ
	touchIDs := inpututil.AppendJustPressedTouchIDs(nil)
	for _, id := range touchIDs {
		tx, ty := ebiten.TouchPosition(id)
		for i := 0; i < 3; i++ {
			bx, by, bw, bh := g.diffButtonRect(i)
			if tx >= bx && tx <= bx+bw && ty >= by && ty <= by+bh {
				g.selectedDiff = i
				g.startPlaying()
				return
			}
		}
	}
}

func (g *Game) updatePlaying() {
	// ポーズ切り替え（キーボード: Space）
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.paused = !g.paused
	}

	// ポーズボタンのタッチ判定（右上の⏸エリア）
	touchIDs := inpututil.AppendJustPressedTouchIDs(nil)
	for _, id := range touchIDs {
		tx, ty := ebiten.TouchPosition(id)
		px, py, pw, ph := g.pauseButtonRect()
		if tx >= px && tx <= px+pw && ty >= py && ty <= py+ph {
			g.paused = !g.paused
			return
		}
	}

	if g.paused {
		// ポーズ中: タップで再開（ポーズボタン以外の領域）
		for _, id := range touchIDs {
			tx, ty := ebiten.TouchPosition(id)
			px, py, pw, ph := g.pauseButtonRect()
			if !(tx >= px && tx <= px+pw && ty >= py && ty <= py+ph) {
				g.paused = false
				return
			}
		}
		return
	}

	// キー入力
	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) && g.direction != dirDown {
		g.nextDir = dirUp
	} else if ebiten.IsKeyPressed(ebiten.KeyArrowDown) && g.direction != dirUp {
		g.nextDir = dirDown
	} else if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) && g.direction != dirRight {
		g.nextDir = dirLeft
	} else if ebiten.IsKeyPressed(ebiten.KeyArrowRight) && g.direction != dirLeft {
		g.nextDir = dirRight
	}

	// 移動タイミング
	g.tickCount++
	if g.tickCount < difficultySpeed[g.difficulty] {
		return
	}
	g.tickCount = 0

	// 蛇の移動
	g.direction = g.nextDir
	delta := dirDelta[g.direction]
	newHead := Point{X: g.snake[0].X + delta.X, Y: g.snake[0].Y + delta.Y}

	// 壁衝突
	if newHead.X < 0 || newHead.X >= columns || newHead.Y < 0 || newHead.Y >= rows {
		g.scene = sceneGameOver
		playSE(seGameOverData)
		return
	}

	// 自身衝突
	for _, p := range g.snake {
		if p == newHead {
			g.scene = sceneGameOver
			playSE(seGameOverData)
			return
		}
	}

	g.snake = append([]Point{newHead}, g.snake...)

	if newHead == g.food {
		g.score++
		g.spawnFood()
		playSE(seEatData)
	} else {
		g.snake = g.snake[:len(g.snake)-1]
	}
}

func (g *Game) updateGameOver() {
	// Enter / タップでタイトルに戻る
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.scene = sceneTitle
		return
	}
	touchIDs := inpututil.AppendJustPressedTouchIDs(nil)
	if len(touchIDs) > 0 {
		g.scene = sceneTitle
	}
}

// ──────────────────────────────────────────────
// Draw
// ──────────────────────────────────────────────

func (g *Game) Draw(screen *ebiten.Image) {
	switch g.scene {
	case sceneTitle:
		g.drawTitle(screen)
	case scenePlaying:
		g.drawPlaying(screen)
		if g.paused {
			g.drawPauseOverlay(screen)
		}
	case sceneGameOver:
		g.drawPlaying(screen) // ゲーム画面を背景に
		g.drawGameOver(screen)
	}
}

func (g *Game) drawTitle(screen *ebiten.Image) {
	// 背景（暗め）
	op := &ebiten.DrawImageOptions{}
	op.ColorScale.Scale(0.15, 0.15, 0.15, 1)
	screen.DrawImage(backgroundImg, op)

	// タイトル
	titleY := 100
	ebitenutil.DebugPrintAt(screen, "SNAKE GAME", screenWidth/2-32, titleY)
	ebitenutil.DebugPrintAt(screen, "~ Eldritch Tentacle Horror ~", screenWidth/2-90, titleY+20)

	// 難易度ボタン
	for i := 0; i < 3; i++ {
		bx, by, bw, bh := g.diffButtonRect(i)

		// ボタン背景
		btnColor := color.RGBA{30, 30, 50, 255}
		if i == g.selectedDiff {
			btnColor = color.RGBA{40, 60, 40, 255}
		}
		ebitenutil.DrawRect(screen, float64(bx), float64(by), float64(bw), float64(bh), btnColor)

		// ボタン枠
		borderColor := color.RGBA{60, 60, 80, 255}
		if i == g.selectedDiff {
			borderColor = difficultyColor[i]
		}
		drawBorder(screen, bx, by, bw, bh, borderColor)

		// ラベル
		label := fmt.Sprintf("[%d] %s", i+1, difficultyName[i])
		labelX := bx + bw/2 - len(label)*3
		labelY := by + bh/2 - 4
		ebitenutil.DebugPrintAt(screen, label, labelX, labelY)

		// 選択中インジケータ
		if i == g.selectedDiff {
			ebitenutil.DebugPrintAt(screen, ">", bx+8, labelY)
		}
	}

	// 操作説明
	helpY := 370
	ebitenutil.DebugPrintAt(screen, "Arrow Keys / Tap to select", screenWidth/2-82, helpY)
	ebitenutil.DebugPrintAt(screen, "Enter / Tap to start", screenWidth/2-62, helpY+16)
}

func (g *Game) drawPlaying(screen *ebiten.Image) {
	// 背景
	op := &ebiten.DrawImageOptions{}
	op.ColorScale.Scale(0.3, 0.3, 0.3, 1)
	screen.DrawImage(backgroundImg, op)

	// 蛇の体
	for i := 1; i < len(g.snake); i++ {
		p := g.snake[i]
		px, py := float64(p.X*gridSize), float64(p.Y*gridSize)

		if i < len(g.snake)-1 {
			prev := g.snake[i-1]
			next := g.snake[i+1]
			if prev.X == next.X {
				drawSprite(screen, bodyVertical, px, py)
			} else if prev.Y == next.Y {
				drawSprite(screen, bodyHorizontal, px, py)
			} else {
				drawSprite(screen, getCurveSprite(p, prev, next), px, py)
			}
		} else {
			prev := g.snake[i-1]
			var tailDir int
			switch {
			case prev.Y < p.Y:
				tailDir = dirDown
			case prev.Y > p.Y:
				tailDir = dirUp
			case prev.X < p.X:
				tailDir = dirRight
			default:
				tailDir = dirLeft
			}
			drawSprite(screen, tailSprites[tailDir], px, py)
		}
	}

	// 食べ物
	drawSprite(screen, foodSprites[g.foodType],
		float64(g.food.X*gridSize), float64(g.food.Y*gridSize))

	// 頭
	head := g.snake[0]
	drawSprite(screen, headSprites[g.direction],
		float64(head.X*gridSize), float64(head.Y*gridSize))

	// HUD: スコア & 難易度
	hudText := fmt.Sprintf("Score: %d  [%s]", g.score, difficultyName[g.difficulty])
	ebitenutil.DebugPrint(screen, hudText)

	// ポーズボタン（右上に常時表示）
	px, py, pw, ph := g.pauseButtonRect()
	ebitenutil.DrawRect(screen, float64(px), float64(py), float64(pw), float64(ph), color.RGBA{0, 0, 0, 100})
	drawBorder(screen, px, py, pw, ph, color.RGBA{80, 80, 80, 255})
	// ⏸ アイコン（2本の縦線）
	barW := 4.0
	barH := float64(ph) - 10
	gap := 6.0
	cx := float64(px) + float64(pw)/2
	cy := float64(py) + 5
	ebitenutil.DrawRect(screen, cx-gap/2-barW, cy, barW, barH, color.RGBA{200, 200, 200, 255})
	ebitenutil.DrawRect(screen, cx+gap/2, cy, barW, barH, color.RGBA{200, 200, 200, 255})
}

func (g *Game) drawPauseOverlay(screen *ebiten.Image) {
	overlay := ebiten.NewImage(screenWidth, screenHeight)
	overlay.Fill(color.RGBA{0, 0, 0, 140})
	screen.DrawImage(overlay, nil)

	cy := screenHeight / 2
	ebitenutil.DebugPrintAt(screen, "PAUSED", screenWidth/2-20, cy-16)
	ebitenutil.DebugPrintAt(screen, "Space / Tap to resume", screenWidth/2-65, cy+4)
}

func (g *Game) drawGameOver(screen *ebiten.Image) {
	overlay := ebiten.NewImage(screenWidth, screenHeight)
	overlay.Fill(color.RGBA{0, 0, 0, 170})
	screen.DrawImage(overlay, nil)

	cy := screenHeight/2 - 20
	ebitenutil.DebugPrintAt(screen, "GAME OVER", screenWidth/2-30, cy)
	scoreText := fmt.Sprintf("Score: %d  (%s)", g.score, difficultyName[g.difficulty])
	ebitenutil.DebugPrintAt(screen, scoreText, screenWidth/2-len(scoreText)*3, cy+24)
	ebitenutil.DebugPrintAt(screen, "Tap or Enter for title", screenWidth/2-68, cy+52)
}

// ──────────────────────────────────────────────
// UI ヘルパー
// ──────────────────────────────────────────────

// diffButtonRect は難易度ボタンi (0〜2) の矩形を返す。
func (g *Game) diffButtonRect(i int) (x, y, w, h int) {
	w = 200
	h = 40
	x = screenWidth/2 - w/2
	y = 200 + i*60
	return
}

// pauseButtonRect はポーズボタンの矩形を返す（右上）。
func (g *Game) pauseButtonRect() (x, y, w, h int) {
	w = 32
	h = 32
	x = screenWidth - w - 8
	y = 8
	return
}

// drawBorder は矩形の枠線を描画する。
func drawBorder(screen *ebiten.Image, x, y, w, h int, c color.RGBA) {
	fx, fy, fw, fh := float64(x), float64(y), float64(w), float64(h)
	t := 2.0 // 線の太さ
	ebitenutil.DrawRect(screen, fx, fy, fw, t, c)           // 上
	ebitenutil.DrawRect(screen, fx, fy+fh-t, fw, t, c)      // 下
	ebitenutil.DrawRect(screen, fx, fy, t, fh, c)            // 左
	ebitenutil.DrawRect(screen, fx+fw-t, fy, t, fh, c)      // 右
}

// ──────────────────────────────────────────────
// タッチ入力処理
// ──────────────────────────────────────────────

func (g *Game) handleTouch() {
	// プレイ中のみスワイプ操作を処理
	if g.scene != scenePlaying || g.paused {
		g.touchTracking = false
		return
	}

	touchIDs := inpututil.AppendJustPressedTouchIDs(nil)
	if len(touchIDs) > 0 && !g.touchTracking {
		g.touchID = touchIDs[0]
		g.touchStartX, g.touchStartY = ebiten.TouchPosition(g.touchID)

		// ポーズボタン領域ならスワイプ追跡しない
		px, py, pw, ph := g.pauseButtonRect()
		if g.touchStartX >= px && g.touchStartX <= px+pw && g.touchStartY >= py && g.touchStartY <= py+ph {
			return
		}
		g.touchTracking = true
	}

	if !g.touchTracking {
		return
	}

	if inpututil.IsTouchJustReleased(g.touchID) {
		g.touchTracking = false
		return
	}

	currentX, currentY := ebiten.TouchPosition(g.touchID)
	dx := currentX - g.touchStartX
	dy := currentY - g.touchStartY

	absDx, absDy := dx, dy
	if absDx < 0 { absDx = -absDx }
	if absDy < 0 { absDy = -absDy }

	if absDx < swipeThreshold && absDy < swipeThreshold {
		return
	}

	if absDx > absDy {
		if dx > 0 && g.direction != dirLeft {
			g.nextDir = dirRight
		} else if dx < 0 && g.direction != dirRight {
			g.nextDir = dirLeft
		}
	} else {
		if dy > 0 && g.direction != dirUp {
			g.nextDir = dirDown
		} else if dy < 0 && g.direction != dirDown {
			g.nextDir = dirUp
		}
	}

	g.touchStartX = currentX
	g.touchStartY = currentY
}

// ──────────────────────────────────────────────
// スプライト描画
// ──────────────────────────────────────────────

func drawSprite(screen *ebiten.Image, sprite *ebiten.Image, x, y float64) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(spriteScale, spriteScale)
	op.GeoM.Translate(x, y)
	screen.DrawImage(sprite, op)
}

func getCurveSprite(p, prev, next Point) *ebiten.Image {
	hasUp := (prev.Y < p.Y) || (next.Y < p.Y)
	hasDown := (prev.Y > p.Y) || (next.Y > p.Y)
	hasLeft := (prev.X < p.X) || (next.X < p.X)
	hasRight := (prev.X > p.X) || (next.X > p.X)

	switch {
	case hasUp && hasRight:
		return curveTopRight
	case hasUp && hasLeft:
		return curveTopLeft
	case hasDown && hasRight:
		return curveBottomRight
	case hasDown && hasLeft:
		return curveBottomLeft
	default:
		return curveTopRight
	}
}

// ──────────────────────────────────────────────
// Layout / main
// ──────────────────────────────────────────────

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Snake Game - Eldritch Tentacle Horror")
	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
}
