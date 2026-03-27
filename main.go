// スネークゲーム - Ebitengineを使ったシンプルなスネークゲーム
//
// ゲームの流れ:
//
//	1. 蛇が画面中央からスタートし、自動的に移動する
//	2. プレイヤーは矢印キーで蛇の方向を操作する
//	3. 赤い食べ物を食べるとスコアが増え、蛇が伸びる
//	4. 壁や自分の体にぶつかるとゲームオーバー
//	5. Enter/Spaceキーでリスタート
package main

import (
	"fmt"
	"image/color"
	"log"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"            // Ebitengineのコアパッケージ
	"github.com/hajimehoshi/ebiten/v2/ebitenutil" // 矩形描画やデバッグ表示などのユーティリティ
)

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
// 蛇と重なった場合はやり直す（グリッドは32x24=768マスあるので、すぐ見つかる）。
func (g *Game) spawnFood() {
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
func (g *Game) Draw(screen *ebiten.Image) {

	// 画面全体を黒で塗りつぶす（前フレームの描画をクリア）
	screen.Fill(color.RGBA{0, 0, 0, 255}) // R=0, G=0, B=0, A=255（不透明な黒）

	// --- 蛇の描画 ---
	// 蛇の各マスを緑色の矩形として描画する
	for _, p := range g.snake {
		// グリッド座標 → ピクセル座標に変換して描画
		// gridSize-1 にすることで、マス同士の間に1ピクセルの隙間ができ、見やすくなる
		ebitenutil.DrawRect(screen,
			float64(p.X*gridSize),      // X座標（ピクセル）: グリッドX × マスのサイズ
			float64(p.Y*gridSize),      // Y座標（ピクセル）: グリッドY × マスのサイズ
			float64(gridSize-1),        // 幅: 19ピクセル（1ピクセルの隙間を作る）
			float64(gridSize-1),        // 高さ: 19ピクセル
			color.RGBA{0, 220, 0, 255}) // 緑色
	}

	// --- 食べ物の描画 ---
	// 食べ物を赤色の矩形として描画する
	ebitenutil.DrawRect(screen,
		float64(g.food.X*gridSize),
		float64(g.food.Y*gridSize),
		float64(gridSize-1),
		float64(gridSize-1),
		color.RGBA{220, 0, 0, 255}) // 赤色

	// --- スコアの表示 ---
	// 画面の左上にスコアを表示する（DebugPrint は常に左上に表示される）
	ebitenutil.DebugPrint(screen, fmt.Sprintf("Score: %d", g.score))

	// --- ゲームオーバー表示 ---
	// ゲームオーバー状態の場合、画面中央にメッセージを表示する
	if g.gameOver {
		// DebugPrintAt を使って指定位置にテキストを表示
		ebitenutil.DebugPrintAt(screen, "GAME OVER", screenWidth/2-30, screenHeight/2-10)
		ebitenutil.DebugPrintAt(screen, "Press Enter or Space to restart", screenWidth/2-95, screenHeight/2+10)
	}
}

// ──────────────────────────────────────────────
// Layout() - 論理画面サイズの指定
// ──────────────────────────────────────────────

// Layout はEbitengineから呼び出され、ゲームの論理的な画面サイズを返す。
// outsideWidth, outsideHeight はウィンドウの実際のサイズ（ユーザーがリサイズした場合など）。
// ここでは固定サイズを返しているので、ウィンドウサイズが変わっても
// ゲーム内の解像度は常に 640x480 で、Ebitengineが自動的にスケーリングしてくれる。
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
