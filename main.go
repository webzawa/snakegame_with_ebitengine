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

	"github.com/hajimehoshi/ebiten/v2"            // Ebitengineのコアパッケージ
	"github.com/hajimehoshi/ebiten/v2/audio"      // オーディオ再生の基盤
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"  // MP3デコーダー
	"github.com/hajimehoshi/ebiten/v2/ebitenutil" // 矩形描画やデバッグ表示などのユーティリティ
	"github.com/hajimehoshi/ebiten/v2/inpututil"  // キーの「押された瞬間」判定用
)

// ──────────────────────────────────────────────
// アセットの埋め込み
// ──────────────────────────────────────────────

//go:embed assets/*.png assets/*.mp3
var assetsFS embed.FS

// ──────────────────────────────────────────────
// 定数の定義
// ──────────────────────────────────────────────

const (
	screenWidth    = 640   // ゲームウィンドウの幅（ピクセル）
	screenHeight   = 480   // ゲームウィンドウの高さ（ピクセル）
	gridSize       = 32    // 1マスのサイズ（ピクセル）。蛇も食べ物もこのサイズで描画される
	columns        = screenWidth / gridSize  // グリッドの横マス数 (640/32 = 20マス)
	rows           = screenHeight / gridSize // グリッドの縦マス数 (480/32 = 15マス)
	spriteSize     = 128   // 元スプライトのサイズ（128x128px）
	sampleRate     = 48000 // オーディオのサンプリングレート（Hz）
	swipeThreshold = 16    // スワイプと判定する最小移動距離（ピクセル）
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

// ──────────────────────────────────────────────
// 方向の定数（iota を使って連番を自動生成）
// ──────────────────────────────────────────────

const (
	dirUp    = iota // 0: 上方向
	dirDown         // 1: 下方向
	dirLeft         // 2: 左方向
	dirRight        // 3: 右方向
)

// 難易度ごとの移動間隔（フレーム数）。値が小さいほど速い。
// Update()は毎秒60回呼ばれるので、例えば20なら 60/20 = 3回/秒の移動速度。
var difficultySpeed = [3]int{20, 12, 7}
var difficultyName = [3]string{"EASY", "NORMAL", "HARD"}
var difficultyColor = [3]color.RGBA{
	{0, 180, 0, 255},   // Easy: 緑
	{200, 180, 0, 255}, // Normal: 黄
	{220, 40, 40, 255}, // Hard: 赤
}

// ──────────────────────────────────────────────
// 型の定義
// ──────────────────────────────────────────────

// Point はグリッド上の座標を表す構造体。
// ピクセル座標ではなく、マス目の位置（0〜19, 0〜14）を保持する。
type Point struct {
	X, Y int
}

// dirDelta は各方向に対応する移動量を定義するマップ。
// ※ Ebitengineの座標系では、Y軸は下方向が正なので、上に移動 = Y が減る
var dirDelta = map[int]Point{
	dirUp:    {0, -1}, // 上: Y座標を1減らす
	dirDown:  {0, 1},  // 下: Y座標を1増やす
	dirLeft:  {-1, 0}, // 左: X座標を1減らす
	dirRight: {1, 0},  // 右: X座標を1増やす
}

// ──────────────────────────────────────────────
// スプライト・オーディオ（パッケージレベル変数）
// ──────────────────────────────────────────────

// これらの変数は init() で一度だけ読み込まれ、全ゲームで共有される。
// Ebitengineの *ebiten.Image はGPU上のテクスチャを表す。
var (
	backgroundImg    *ebiten.Image            // 背景画像（640x480、画面全体に描画）
	headSprites      map[int]*ebiten.Image    // 頭のスプライト（方向別に4枚）
	tailSprites      map[int]*ebiten.Image    // 尻尾のスプライト（方向別に4枚）
	bodyVertical     *ebiten.Image            // 縦方向の直線セグメント
	bodyHorizontal   *ebiten.Image            // 横方向の直線セグメント
	curveTopLeft     *ebiten.Image            // カーブ: 上と左を繋ぐ角
	curveTopRight    *ebiten.Image            // カーブ: 上と右を繋ぐ角
	curveBottomLeft  *ebiten.Image            // カーブ: 下と左を繋ぐ角
	curveBottomRight *ebiten.Image            // カーブ: 下と右を繋ぐ角
	foodSprites      [3]*ebiten.Image         // 食べ物のスプライト（3種類からランダム）

	audioCtx       = audio.NewContext(sampleRate) // オーディオコンテキスト（全体で1つ）
	bgmPlayer      *audio.Player                  // BGMプレイヤー（ループ再生用）
	seEatData      []byte                         // 食べ物を食べた時のSE（デコード済みPCM）
	seGameOverData []byte                         // ゲームオーバー時のSE（デコード済みPCM）
)

// spriteScale はスプライト(128px)をグリッド(32px)に収めるための縮小倍率。
var spriteScale = float64(gridSize) / float64(spriteSize)

// loadImage は埋め込みファイルシステムからPNG画像を読み込み、
// Ebitengineの画像オブジェクト（GPUテクスチャ）に変換して返す。
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

// decodeSE はMP3の効果音ファイルを読み込み、デコード済みPCMバイト列を返す。
// 毎回再生のたびにデコードするのではなく、起動時に一度だけデコードしてメモリに保持する。
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

// BGM / SE の音量定数（元の2/3に設定）
const (
	bgmVolume = 0.2  // BGM音量
	seVolume  = 0.33 // SE音量
)

// playSE はデコード済みPCMデータからプレイヤーを作成して再生する。
// ミュート中は何もしない。毎回新しいプレイヤーを作成するため、同じSEの重複再生も可能。
func playSE(data []byte, muted bool) {
	if muted {
		return
	}
	player, err := audioCtx.NewPlayer(bytes.NewReader(data))
	if err != nil {
		return
	}
	player.SetVolume(seVolume)
	player.Play()
}

// applyVolume はミュート状態に応じてBGM音量を設定する。
func (g *Game) applyVolume() {
	if g.muted {
		bgmPlayer.SetVolume(0)
	} else {
		bgmPlayer.SetVolume(bgmVolume)
	}
}

// toggleMute はミュート状態をトグルし、BGM音量に反映する。
func (g *Game) toggleMute() {
	g.muted = !g.muted
	g.applyVolume()
}

// initBGM はBGMファイルを読み込み、無限ループ再生を開始する（初期音量は0）。
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
	bgmPlayer.SetVolume(0)  // タイトル画面では無音で開始
	bgmPlayer.Play()
}

// init はプログラム起動時に自動的に呼ばれる。
// 全スプライト画像・SE・BGMをここで一括読み込みする。
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
// Game 構造体（ebiten.Game インターフェースを実装）
// ──────────────────────────────────────────────

// Game はゲーム全体の状態を保持する構造体。
// シーン管理（タイトル→プレイ→ゲームオーバー→タイトル）で画面遷移を制御する。
type Game struct {
	scene      int // 現在のシーン（sceneTitle / scenePlaying / sceneGameOver）
	difficulty int // 選択された難易度（diffEasy / diffNormal / diffHard）

	// プレイ中の状態
	snake     []Point // 蛇の体。snake[0] が頭、末尾が尻尾
	direction int     // 蛇の現在の移動方向
	nextDir   int     // 次の移動時に適用される方向（180度反転防止用バッファ）
	food      Point   // 食べ物の現在位置（グリッド座標）
	foodType  int     // 食べ物の種類（0: ネクロノミコン, 1: 脳, 2: エルダーサイン）
	score     int     // 現在のスコア（食べ物を食べた回数）
	paused    bool    // 一時停止中かどうか
	tickCount int     // フレームカウンタ。移動間隔に達するたびに蛇が1マス移動する

	// タイトル画面
	selectedDiff int  // 難易度カーソル位置（0: Easy, 1: Normal, 2: Hard）
	muted        bool // 音量オフかどうか（タイトル画面のボタンでトグル）

	// タッチ操作用（スマホ・タブレット対応）
	touchID       ebiten.TouchID // 現在追跡中のタッチID
	touchStartX   int            // タッチ開始X座標
	touchStartY   int            // タッチ開始Y座標
	touchTracking bool           // タッチを追跡中かどうか

	// アニメーション用カウンタ
	frameCount int
}

// NewGame はゲームの初期状態を作成して返す。タイトル画面から始まる。
func NewGame() *Game {
	return &Game{
		scene:        sceneTitle,
		selectedDiff: diffNormal, // デフォルト: Normal
		muted:        true,       // タイトル画面は無音で開始
	}
}

// startPlaying は難易度を確定してプレイシーンに遷移する。
// 蛇の初期配置（画面中央に3マス、右向き）と食べ物の初期生成を行う。
// ミュートがオフならBGMの音量を復元する。
func (g *Game) startPlaying() {
	g.scene = scenePlaying
	g.difficulty = g.selectedDiff
	// プレイ開始時にBGM音量を適用
	g.applyVolume()
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

// spawnFood は蛇と重ならないランダムな位置に食べ物を配置する。
// 食べ物の種類（3種類）もランダムに決定する。
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
// Update() - ゲームロジックの更新（毎秒60回呼ばれる）
// ──────────────────────────────────────────────

// Update はEbitengineから毎秒60回（60TPS）呼び出される。
// 現在のシーンに応じて処理を振り分ける。
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

// updateTitle はタイトル画面のロジック。
// 矢印キー↑↓で難易度カーソル移動、Enter/Space/数字キーで開始。タッチでボタンタップ。
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

	// Mキーでミュートトグル
	if inpututil.IsKeyJustPressed(ebiten.KeyM) {
		g.toggleMute()
	}

	// タッチでボタンタップ
	touchIDs := inpututil.AppendJustPressedTouchIDs(nil)
	for _, id := range touchIDs {
		tx, ty := ebiten.TouchPosition(id)

		// 音量ボタンの判定
		sx, sy, sw, sh := g.soundButtonRect()
		if tx >= sx && tx <= sx+sw && ty >= sy && ty <= sy+sh {
			g.toggleMute()
			return
		}

		// 難易度ボタンの判定
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

// updatePlaying はプレイ中のロジック。
// キー入力・スワイプ処理、移動タイミング制御、衝突判定、食べ物判定を行う。
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
		playSE(seGameOverData, g.muted)
		return
	}

	// 自身衝突
	for _, p := range g.snake {
		if p == newHead {
			g.scene = sceneGameOver
			playSE(seGameOverData, g.muted)
			return
		}
	}

	g.snake = append([]Point{newHead}, g.snake...)

	if newHead == g.food {
		g.score++
		g.spawnFood()
		playSE(seEatData, g.muted)
	} else {
		g.snake = g.snake[:len(g.snake)-1]
	}
}

// updateGameOver はゲームオーバー画面のロジック。
// Enter / Space / タップでタイトル画面に戻る。
func (g *Game) updateGameOver() {
	goToTitle := false
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		goToTitle = true
	}
	touchIDs := inpututil.AppendJustPressedTouchIDs(nil)
	if len(touchIDs) > 0 {
		goToTitle = true
	}
	if goToTitle {
		g.scene = sceneTitle
		// ユーザーのミュート設定はそのまま維持する
	}
}

// ──────────────────────────────────────────────
// Draw() - 画面の描画（毎フレーム呼ばれる）
// ──────────────────────────────────────────────

// Draw はEbitengineから毎フレーム呼び出される。
// 現在のシーンに応じた画面を描画する。
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

// drawTitle はタイトル画面を描画する。背景（暗め）、タイトル文字、難易度ボタン3つ、操作説明。
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

	// 音量ボタン（右上）
	sx, sy, sw, sh := g.soundButtonRect()
	ebitenutil.DrawRect(screen, float64(sx), float64(sy), float64(sw), float64(sh), color.RGBA{30, 30, 50, 255})
	borderC := color.RGBA{80, 80, 100, 255}
	if !g.muted {
		borderC = color.RGBA{0, 180, 0, 255}
	}
	drawBorder(screen, sx, sy, sw, sh, borderC)
	soundLabel := "Sound ON"
	if g.muted {
		soundLabel = "Sound OFF"
	}
	ebitenutil.DebugPrintAt(screen, soundLabel, sx+sw/2-len(soundLabel)*3, sy+sh/2-4)

	// 操作説明
	helpY := 370
	ebitenutil.DebugPrintAt(screen, "Arrow Keys / Tap to select", screenWidth/2-82, helpY)
	ebitenutil.DebugPrintAt(screen, "Enter / Tap to start", screenWidth/2-62, helpY+16)
	ebitenutil.DebugPrintAt(screen, "[M] Toggle sound", screenWidth/2-50, helpY+32)
}

// drawPlaying はプレイ中の画面を描画する。
// 背景→蛇の体→食べ物→頭→HUD→ポーズボタン の順で重ねて描画。
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

// drawPauseOverlay はポーズ中のオーバーレイを描画する。
func (g *Game) drawPauseOverlay(screen *ebiten.Image) {
	overlay := ebiten.NewImage(screenWidth, screenHeight)
	overlay.Fill(color.RGBA{0, 0, 0, 140})
	screen.DrawImage(overlay, nil)

	cy := screenHeight / 2
	ebitenutil.DebugPrintAt(screen, "PAUSED", screenWidth/2-20, cy-16)
	ebitenutil.DebugPrintAt(screen, "Space / Tap to resume", screenWidth/2-65, cy+4)
}

// drawGameOver はゲームオーバーのオーバーレイを描画する。スコアと難易度を表示。
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

// soundButtonRect はタイトル画面の音量ボタンの矩形を返す（右上）。
func (g *Game) soundButtonRect() (x, y, w, h int) {
	w = 80
	h = 28
	x = screenWidth - w - 16
	y = 16
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

// handleTouch はタッチ入力を処理し、スワイプ方向に応じて nextDir を設定する。
// タッチ開始位置から swipeThreshold 以上スワイプしたら方向を確定する。
// プレイ中のみ動作し、ポーズボタン領域のタッチはスワイプ追跡から除外する。
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

// drawSprite はスプライト画像(128x128)をグリッドサイズ(32x32)に縮小して描画する。
func drawSprite(screen *ebiten.Image, sprite *ebiten.Image, x, y float64) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(spriteScale, spriteScale)
	op.GeoM.Translate(x, y)
	screen.DrawImage(sprite, op)
}

// getCurveSprite は体のカーブセグメントに使うスプライトを返す。
// 現在のセグメント位置(p)と、頭側(prev)・尻尾側(next)の隣接セグメントの位置から、
// 4種類のカーブスプライトのどれを使うか判定する。
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
// Layout() - 論理画面サイズの指定
// ──────────────────────────────────────────────

// Layout はEbitengineから呼び出され、ゲームの論理的な画面サイズを返す。
// 固定サイズを返すので、ウィンドウサイズが変わってもEbitengineが自動スケーリングする。
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

// ──────────────────────────────────────────────
// main() - エントリーポイント
// ──────────────────────────────────────────────

// main はプログラムのエントリーポイント。
// ウィンドウの設定を行い、ゲームループを開始する。
func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Snake Game - Eldritch Tentacle Horror")
	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
}
