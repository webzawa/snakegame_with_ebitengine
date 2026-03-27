// スネークゲーム - Ebitengineを使ったクトゥルフ風スネークゲーム
//
// ゲームの流れ:
//
//  1. 触手の蛇が画面中央からスタートし、自動的に移動する
//  2. プレイヤーは矢印キーで蛇の方向を操作する
//  3. 食べ物（ネクロノミコン・脳・エルダーサイン）を食べるとスコアが増え、蛇が伸びる
//  4. 壁や自分の体にぶつかるとゲームオーバー
//  5. Enter/Spaceキーでリスタート
package main

import (
	"embed"
	"fmt"
	"image/color"
	"image/png"
	"log"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"            // Ebitengineのコアパッケージ
	"github.com/hajimehoshi/ebiten/v2/ebitenutil" // 矩形描画やデバッグ表示などのユーティリティ
)

// ──────────────────────────────────────────────
// アセットの埋め込み
// ──────────────────────────────────────────────

// go:embed ディレクティブで assets/ フォルダ内の全PNGファイルをバイナリに埋め込む。
// これにより、実行ファイル単体でゲームが動作する（外部ファイル不要）。
//
//go:embed assets/*.png
var assetsFS embed.FS

// ──────────────────────────────────────────────
// 定数の定義
// ──────────────────────────────────────────────

const (
	screenWidth  = 640 // ゲームウィンドウの幅（ピクセル）
	screenHeight = 480 // ゲームウィンドウの高さ（ピクセル）
	gridSize     = 20  // 1マスのサイズ（ピクセル）。蛇も食べ物もこのサイズで描画される
	moveInterval = 8   // 蛇が移動する間隔（フレーム数）。Update()は毎秒60回呼ばれるので、
	//                    8フレームごと = 約7.5回/秒の移動速度になる

	columns = screenWidth / gridSize  // グリッドの横マス数 (640/20 = 32マス)
	rows    = screenHeight / gridSize // グリッドの縦マス数 (480/20 = 24マス)

	spriteSize = 128 // 元スプライトのサイズ（128x128px）
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

// ──────────────────────────────────────────────
// 型の定義
// ──────────────────────────────────────────────

// Point はグリッド上の座標を表す構造体。
// ピクセル座標ではなく、マス目の位置（0〜31, 0〜23）を保持する。
type Point struct {
	X, Y int
}

// dirDelta は各方向に対応する移動量を定義するマップ。
// 例えば dirUp（上方向）の場合、Xは変化なし(0)、Yは-1（上に1マス移動）。
// ※ Ebitengineの座標系では、Y軸は下方向が正なので、上に移動 = Y が減る
var dirDelta = map[int]Point{
	dirUp:    {0, -1}, // 上: Y座標を1減らす
	dirDown:  {0, 1},  // 下: Y座標を1増やす
	dirLeft:  {-1, 0}, // 左: X座標を1減らす
	dirRight: {1, 0},  // 右: X座標を1増やす
}

// ──────────────────────────────────────────────
// スプライト画像（パッケージレベル変数）
// ──────────────────────────────────────────────

// これらの変数は init() で一度だけ読み込まれ、全ゲームで共有される。
// Ebitengineの *ebiten.Image はGPU上のテクスチャを表す。
var (
	// 背景画像（640x480、画面全体に描画）
	backgroundImg *ebiten.Image

	// 頭のスプライト（方向別に4枚。それぞれ事前に正しい向きで用意されている）
	headSprites map[int]*ebiten.Image

	// 体のスプライト
	bodyVertical   *ebiten.Image // 縦方向の直線セグメント
	bodyHorizontal *ebiten.Image // 横方向の直線セグメント

	// カーブ（曲がり角）のスプライト
	// 蛇が方向転換する箇所に使用。名前は曲がり角の位置を表す。
	// 例: curveTopRight は「上と右を繋ぐカーブ」= 隣接セグメントが上と右にある
	curveTopLeft     *ebiten.Image
	curveTopRight    *ebiten.Image
	curveBottomLeft  *ebiten.Image
	curveBottomRight *ebiten.Image

	// 食べ物のスプライト（3種類からランダムに選ばれる）
	foodSprites [3]*ebiten.Image
)

// spriteScale はスプライト(128px)をグリッド(20px)に収めるための縮小倍率。
var spriteScale = float64(gridSize) / float64(spriteSize)

// loadImage は埋め込みファイルシステムからPNG画像を読み込み、
// Ebitengineの画像オブジェクトに変換して返す。
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

// init はプログラム起動時に自動的に呼ばれる。
// 全スプライト画像をここで一括読み込みする。
func init() {
	// 背景画像
	backgroundImg = loadImage("background")

	// 頭（方向別に4枚）
	headSprites = map[int]*ebiten.Image{
		dirUp:    loadImage("head_up"),
		dirDown:  loadImage("head_down"),
		dirLeft:  loadImage("head_left"),
		dirRight: loadImage("head_right"),
	}

	// 体の直線セグメント
	bodyVertical = loadImage("body_vertical")
	bodyHorizontal = loadImage("body_horizontal")

	// 体のカーブセグメント（4方向の曲がり角）
	curveTopLeft = loadImage("body_segment_curved_top_left")
	curveTopRight = loadImage("body_segment_curved_top_right")
	curveBottomLeft = loadImage("body_segment_curved_bottom_left")
	curveBottomRight = loadImage("body_segment_curved_bottom_right")

	// 食べ物（ネクロノミコン、脳、エルダーサイン）
	foodSprites[0] = loadImage("food_necronomicon")
	foodSprites[1] = loadImage("food_brain")
	foodSprites[2] = loadImage("food_elder_sign")
}

// ──────────────────────────────────────────────
// Game 構造体（ebiten.Game インターフェースを実装）
// ──────────────────────────────────────────────

// Game はゲーム全体の状態を保持する構造体。
// Ebitengineでは、この構造体に Update(), Draw(), Layout() の3つのメソッドを
// 実装することでゲームが動作する。
type Game struct {
	snake     []Point // 蛇の体を構成するマス目の座標のスライス。snake[0] が頭、末尾が尻尾
	direction int     // 蛇の現在の移動方向（dirUp, dirDown, dirLeft, dirRight のいずれか）
	nextDir   int     // 次の移動時に適用される方向。入力を一時的にバッファリングすることで、
	//                   1フレーム内に逆方向キーを押して即死するバグを防ぐ
	food      Point // 食べ物の現在位置（グリッド座標）
	foodType  int   // 食べ物の種類（0: ネクロノミコン, 1: 脳, 2: エルダーサイン）
	score     int   // 現在のスコア（食べ物を食べた回数）
	gameOver  bool  // ゲームオーバー状態かどうか
	tickCount int   // フレームカウンタ。moveInterval に達するたびに蛇が1マス移動する
}

// ──────────────────────────────────────────────
// ゲームの初期化
// ──────────────────────────────────────────────

// NewGame はゲームの初期状態を作成して返す。
// ゲーム開始時とリスタート時に呼ばれる。
func NewGame() *Game {
	g := &Game{
		direction: dirRight, // 初期方向: 右向き
		nextDir:   dirRight, // 次の方向も右向きで初期化
	}

	// 蛇の初期位置: 画面中央に3マス分の蛇を横向きに配置
	// centerX=16, centerY=12 の場合、蛇は (16,12), (15,12), (14,12) の3マス
	centerX := columns / 2
	centerY := rows / 2
	for i := 0; i < 3; i++ {
		g.snake = append(g.snake, Point{X: centerX - i, Y: centerY})
	}

	// 食べ物を最初の位置に配置
	g.spawnFood()
	return g
}

// ──────────────────────────────────────────────
// 食べ物の生成
// ──────────────────────────────────────────────

// spawnFood は蛇と重ならないランダムな位置に食べ物を配置する。
// 食べ物の種類（3種類）もランダムに決定する。
func (g *Game) spawnFood() {
	g.foodType = rand.Intn(3)

	for {
		// ランダムなグリッド座標を生成
		g.food = Point{
			X: rand.Intn(columns), // 0 〜 31 のランダムな整数
			Y: rand.Intn(rows),    // 0 〜 23 のランダムな整数
		}

		// 蛇の体と重なっていないかチェック
		overlap := false
		for _, p := range g.snake {
			if p == g.food {
				overlap = true
				break
			}
		}

		// 重なっていなければ、この位置に決定して終了
		if !overlap {
			return
		}
		// 重なっていたら、forループの先頭に戻って別の位置を試す
	}
}

// ──────────────────────────────────────────────
// Update() - ゲームロジックの更新（毎秒60回呼ばれる）
// ──────────────────────────────────────────────

// Update はEbitengineから毎秒60回（60TPS）呼び出される。
// ここでキー入力の処理、蛇の移動、衝突判定などのゲームロジックを行う。
// error を返すとゲームが終了する（通常は nil を返す）。
func (g *Game) Update() error {

	// --- ゲームオーバー中の処理 ---
	// ゲームオーバー状態では、Enter または Space キーが押されたらゲームをリスタート
	if g.gameOver {
		if ebiten.IsKeyPressed(ebiten.KeyEnter) || ebiten.IsKeyPressed(ebiten.KeySpace) {
			// NewGame() で新しいゲーム状態を作り、現在のGameを丸ごと上書きする
			// ポインタの中身を差し替えることで、Ebitengineが保持しているポインタはそのまま使える
			*g = *NewGame()
		}
		return nil // ゲームオーバー中は以降の処理をスキップ
	}

	// --- キー入力の処理 ---
	// 矢印キーの入力を nextDir に保存する。
	// ただし、現在の進行方向と真逆の方向は無視する（即死防止）。
	// 例: 右に進んでいるときに左キーを押しても無視される
	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) && g.direction != dirDown {
		g.nextDir = dirUp
	} else if ebiten.IsKeyPressed(ebiten.KeyArrowDown) && g.direction != dirUp {
		g.nextDir = dirDown
	} else if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) && g.direction != dirRight {
		g.nextDir = dirLeft
	} else if ebiten.IsKeyPressed(ebiten.KeyArrowRight) && g.direction != dirLeft {
		g.nextDir = dirRight
	}

	// --- 移動タイミングの制御 ---
	// Update() は毎秒60回呼ばれるが、蛇を毎フレーム動かすと速すぎる。
	// tickCount をカウントアップし、moveInterval(=8) フレームごとに1回だけ蛇を動かす。
	g.tickCount++
	if g.tickCount < moveInterval {
		return nil // まだ移動タイミングでないので何もしない
	}
	g.tickCount = 0 // カウンタをリセット

	// --- 蛇の移動処理 ---
	// バッファリングしていた nextDir を実際の direction に適用
	g.direction = g.nextDir

	// 現在の方向に対応する移動量（delta）を取得
	delta := dirDelta[g.direction]

	// 新しい頭の位置を計算（現在の頭の位置 + 移動量）
	newHead := Point{
		X: g.snake[0].X + delta.X,
		Y: g.snake[0].Y + delta.Y,
	}

	// --- 壁との衝突判定 ---
	// 新しい頭の位置がグリッドの範囲外に出たらゲームオーバー
	if newHead.X < 0 || newHead.X >= columns || newHead.Y < 0 || newHead.Y >= rows {
		g.gameOver = true
		return nil
	}

	// --- 自分自身との衝突判定 ---
	// 新しい頭の位置が蛇の体のどこかと重なったらゲームオーバー
	for _, p := range g.snake {
		if p == newHead {
			g.gameOver = true
			return nil
		}
	}

	// --- 蛇を前に進める ---
	// 新しい頭をスライスの先頭に追加する
	// append([]Point{newHead}, g.snake...) で「新しい頭 + 既存の体」の新しいスライスを作る
	g.snake = append([]Point{newHead}, g.snake...)

	// --- 食べ物の判定 ---
	if newHead == g.food {
		// 食べ物を食べた場合:
		// - スコアを1増やす
		// - 新しい食べ物を生成する
		// - 尻尾は削除しない（= 蛇が1マス伸びる）
		g.score++
		g.spawnFood()
	} else {
		// 食べ物を食べていない場合:
		// - 尻尾の最後の1マスを削除する（= 蛇の長さを維持して前に進む）
		// スライスの末尾を切り詰めることで尻尾を削除
		g.snake = g.snake[:len(g.snake)-1]
	}

	return nil
}

// ──────────────────────────────────────────────
// Draw() - 画面の描画（毎フレーム呼ばれる）
// ──────────────────────────────────────────────

// Draw はEbitengineから毎フレーム呼び出される。
// screen は描画先の画像（ゲーム画面全体）。
// 毎回画面をクリアしてから全てを描き直す（ダブルバッファリングはEbitengineが自動で行う）。
// スプライト画像を使って蛇・食べ物・UIを描画する。
func (g *Game) Draw(screen *ebiten.Image) {
	// --- 背景画像の描画 ---
	screen.DrawImage(backgroundImg, nil)

	// --- 蛇の体の描画（頭以外のセグメント） ---
	for i := 1; i < len(g.snake); i++ {
		p := g.snake[i]
		px, py := float64(p.X*gridSize), float64(p.Y*gridSize)

		if i < len(g.snake)-1 {
			// 中間セグメント: 前後のセグメントを見て直線/カーブを判定
			prev := g.snake[i-1] // 頭側の隣接セグメント
			next := g.snake[i+1] // 尻尾側の隣接セグメント

			if prev.X == next.X {
				// 前後が同じX座標 → 縦方向の直線
				drawSprite(screen, bodyVertical, px, py)
			} else if prev.Y == next.Y {
				// 前後が同じY座標 → 横方向の直線
				drawSprite(screen, bodyHorizontal, px, py)
			} else {
				// 前後が異なる軸 → カーブ（曲がり角）
				curve := getCurveSprite(p, prev, next)
				drawSprite(screen, curve, px, py)
			}
		} else {
			// 尻尾（最後のセグメント）: 一つ前のセグメントとの位置関係で直線を選択
			prev := g.snake[i-1]
			if prev.X == p.X {
				drawSprite(screen, bodyVertical, px, py)
			} else {
				drawSprite(screen, bodyHorizontal, px, py)
			}
		}
	}

	// --- 食べ物の描画 ---
	drawSprite(screen, foodSprites[g.foodType],
		float64(g.food.X*gridSize), float64(g.food.Y*gridSize))

	// --- 頭の描画 ---
	// 方向別に事前に用意されたスプライトを使うので、回転処理は不要
	head := g.snake[0]
	drawSprite(screen, headSprites[g.direction],
		float64(head.X*gridSize), float64(head.Y*gridSize))

	// --- スコアの表示 ---
	// 画面の左上にスコアを表示する（DebugPrint は常に左上に表示される）
	ebitenutil.DebugPrint(screen, fmt.Sprintf("Score: %d", g.score))

	// --- ゲームオーバー表示 ---
	// ゲームオーバー状態の場合、画面中央にメッセージを表示する
	if g.gameOver {
		// DebugPrintAt を使って指定位置にテキストを表示
		// 半透明の黒オーバーレイを描画して視認性を上げる
		overlay := ebiten.NewImage(screenWidth, screenHeight)
		overlay.Fill(color.RGBA{0, 0, 0, 160})
		screen.DrawImage(overlay, nil)

		ebitenutil.DebugPrintAt(screen, "GAME OVER", screenWidth/2-30, screenHeight/2-10)
		ebitenutil.DebugPrintAt(screen, "Press Enter or Space to restart", screenWidth/2-95, screenHeight/2+10)
	}
}

// drawSprite はスプライト画像(128x128)をグリッドサイズ(20x20)に縮小して描画する。
func drawSprite(screen *ebiten.Image, sprite *ebiten.Image, x, y float64) {
	op := &ebiten.DrawImageOptions{}
	// 128x128 → 20x20 に縮小
	op.GeoM.Scale(spriteScale, spriteScale)
	// 画面上の指定位置に配置
	op.GeoM.Translate(x, y)
	screen.DrawImage(sprite, op)
}

// getCurveSprite は体のカーブセグメントに使うスプライトを返す。
// 現在のセグメント位置(p)と、頭側(prev)・尻尾側(next)の隣接セグメントの位置から、
// 4種類のカーブスプライトのどれを使うか判定する。
//
// カーブの名前は「繋がる2方向」を表す:
//
//	top_right:    上と右を繋ぐ角（隣接セグメントが上と右にある）
//	top_left:     上と左を繋ぐ角
//	bottom_right: 下と右を繋ぐ角
//	bottom_left:  下と左を繋ぐ角
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
		return curveTopRight // フォールバック
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
	// ウィンドウサイズを設定（ピクセル単位）
	ebiten.SetWindowSize(screenWidth, screenHeight)

	// ウィンドウのタイトルバーに表示されるテキストを設定
	ebiten.SetWindowTitle("Snake Game")

	// ゲームループを開始する。RunGame は内部で以下を繰り返す:
	//   1. Update() を呼ぶ（毎秒60回）
	//   2. Draw() を呼ぶ（毎フレーム）
	//   3. 画面を更新する
	// ウィンドウが閉じられるまでこの関数はブロックする（戻ってこない）
	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err) // エラーが発生した場合はログに出力して終了
	}
}
