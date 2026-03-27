# Ebitengine 初心者ガイド

## Ebitengineとは

Ebitengine（エビテンジン）は、Go言語で書かれたオープンソースの2Dゲームエンジン。
シンプルなAPIでクロスプラットフォーム（Windows, macOS, Linux, Web, モバイル）に対応している。

- 公式サイト: https://ebitengine.org/
- GitHub: https://github.com/hajimehoshi/ebiten

## セットアップ

### 前提条件

- Go 1.22 以上
- macOS の場合: Xcode Command Line Tools (`xcode-select --install`)

### インストール

```bash
go mod init <モジュール名>
go get github.com/hajimehoshi/ebiten/v2
```

## 基本アーキテクチャ

Ebitengineのゲームは `ebiten.Game` インターフェースを実装する構造体を中心に構成される。

### `ebiten.Game` インターフェース

```go
type Game interface {
    Update() error
    Draw(screen *ebiten.Image)
    Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int)
}
```

| メソッド | 呼び出し頻度 | 役割 |
|---------|------------|------|
| `Update()` | 60回/秒 (TPS) | ゲームロジック（入力処理、状態更新、衝突判定など） |
| `Draw(screen)` | 毎フレーム | 画面の描画（Update より頻繁に呼ばれることがある） |
| `Layout(w, h)` | ウィンドウサイズ変更時 | 論理的な画面サイズを返す |

### ゲームループの起動

```go
func main() {
    ebiten.SetWindowSize(640, 480)
    ebiten.SetWindowTitle("My Game")
    if err := ebiten.RunGame(&MyGame{}); err != nil {
        log.Fatal(err)
    }
}
```

`RunGame()` はブロッキング関数で、ウィンドウが閉じられるまで返らない。

## 座標系

```
(0,0) ────────── X+ →
  │
  │
  │
  Y+ ↓
```

- **原点**: 左上 (0, 0)
- **X軸**: 右方向が正
- **Y軸**: 下方向が正
- 単位はピクセル

## 描画 API

### 背景の塗りつぶし

```go
screen.Fill(color.RGBA{0, 0, 0, 255}) // 黒で塗りつぶし
```

### 矩形の描画

```go
import "github.com/hajimehoshi/ebiten/v2/ebitenutil"

// DrawRect(dst, x, y, width, height, color)
ebitenutil.DrawRect(screen, 100, 200, 50, 50, color.RGBA{255, 0, 0, 255})
```

### デバッグテキスト表示

```go
// 左上に表示
ebitenutil.DebugPrint(screen, "Hello, World!")

// 指定位置に表示
ebitenutil.DebugPrintAt(screen, "Score: 10", 100, 50)
```

フォントは組み込みのモノスペースフォント（小さめ）。本格的なテキスト表示には `text/v2` パッケージを使う。

## 入力処理

### キーの押下状態

```go
// キーが押されている間ずっと true
if ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
    // 上キーが押されている
}
```

### キーが押された瞬間の判定

```go
import "github.com/hajimehoshi/ebiten/v2/inpututil"

// キーが押された瞬間だけ true（1フレームだけ）
if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
    // スペースキーが押された瞬間
}
```

### 主なキー定数

| 定数 | キー |
|------|------|
| `ebiten.KeyArrowUp` | ↑ |
| `ebiten.KeyArrowDown` | ↓ |
| `ebiten.KeyArrowLeft` | ← |
| `ebiten.KeyArrowRight` | → |
| `ebiten.KeySpace` | Space |
| `ebiten.KeyEnter` | Enter |
| `ebiten.KeyEscape` | Escape |

## 最小構成のサンプル

```go
package main

import (
    "image/color"
    "log"

    "github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

type Game struct{}

func (g *Game) Update() error {
    return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
    screen.Fill(color.RGBA{0, 0, 0, 255})
    ebitenutil.DebugPrint(screen, "Hello, Ebitengine!")
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
    return 640, 480
}

func main() {
    ebiten.SetWindowSize(640, 480)
    ebiten.SetWindowTitle("Hello")
    if err := ebiten.RunGame(&Game{}); err != nil {
        log.Fatal(err)
    }
}
```

## 参考リンク

- [Ebitengine公式ドキュメント](https://ebitengine.org/en/documents/)
- [Ebitengineサンプル集](https://ebitengine.org/en/examples/)
- [API リファレンス (pkg.go.dev)](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2)
