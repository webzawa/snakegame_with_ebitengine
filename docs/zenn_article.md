---
title: "Claude Code × Gemini × Ebitengine：AIだけでクトゥルフ風スネークゲームを作った話"
emoji: "🐙"
type: "tech"
topics: ["go", "ebitengine", "claudecode", "gemini", "gamedev"]
published: false
---

## はじめに

「AIでゲームって作れるの？」という疑問に、実際にやってみた記録です。

**Claude Code**（Anthropic の CLI エージェント）にゲームロジックの実装を任せ、**Gemini** でスプライト画像を生成し、**Pixabay** からフリーの SE を取得。コードを書く手間をほぼゼロにして、Go の 2D ゲームエンジン **Ebitengine** でクトゥルフ神話テーマのスネークゲームを完成させました。

![完成イメージ](https://github.com/user-attachments/assets/placeholder)

普段は Go でバックエンドや SaaS 開発をしていますが、ゲーム開発は完全に未経験。Ebitengine も今回が初めてです。それでも **約2時間** で BGM・SE 付きの完成品ができあがりました。

:::message
この記事のコード・画像・音声はすべて AI で生成したものです。
リポジトリ: https://github.com/webzawa/snakegame_with_ebitengine
:::

## 技術スタック

| 役割 | 技術 |
|------|------|
| ゲームエンジン | [Ebitengine](https://ebitengine.org/) v2 (Go) |
| コード生成 | Claude Code (Opus 4.6) |
| 画像生成 | Gemini Nanobanana2 |
| BGM | AI 生成楽曲 |
| SE | Pixabay（ロイヤリティフリー） |
| 画像加工 | Python (Pillow, OpenCV, scipy) |

## 全体の流れ

```
プラン策定 → 基本実装 → スプライト生成 → 統合 → BGM/SE → 調整
   ↑Claude Code      ↑Claude Code  ↑Gemini     ↑Claude Code  ↑Pixabay+Claude Code
```

## Step 1: Claude Code にプランを立てさせる

まず空のリポジトリで Claude Code の Plan Mode を起動。

```
> /plan このリポジトリはgolangのゲームエンジン、ebitengineを使用した
> snakegameを実装するためのリポジトリです。0>1で実装するためのplanを立てて。
```

Claude Code は内部で **Explore エージェント** と **Plan エージェント** を起動し、以下のプランを提示してきました：

- 2ファイル構成（`go.mod` + `main.go`）
- `ebiten.Game` インターフェースの 3 メソッド実装
- 180度反転防止の `nextDir` バッファ設計
- `ebitenutil.DrawRect` による最小限の描画

ここで重要なのは、Claude Code が **Ebitengine の API を正確に把握している** こと。`Update()` が 60TPS で呼ばれること、`DrawRect` の引数順序、`IsKeyPressed` の使い方など、ドキュメントを読まなくても正しいコードが出てきます。

## Step 2: 一発で動くコードを生成

プラン承認後、Claude Code が `go.mod` と `main.go` を一気に生成。

```go
// Game 構造体 - これだけでスネークゲームの全状態を表現
type Game struct {
    snake     []Point // snake[0] が頭、末尾が尻尾
    direction int
    nextDir   int     // 180度反転防止用バッファ
    food      Point
    score     int
    gameOver  bool
    tickCount int     // 移動速度制御用
}
```

`go mod tidy` → `go build` まで自動実行して、**一度もエラーなく動作するコードが完成**。体感で 1 分程度でした。

### nextDir バッファの設計が秀逸

Claude Code が自発的に提案した `nextDir` パターンは、スネークゲームの定番バグ（1フレーム内に逆方向キーを押して即死）を防ぐもの。ゲーム開発の知見がちゃんと反映されています。

```go
// 入力は nextDir に保存（即座には適用しない）
if ebiten.IsKeyPressed(ebiten.KeyArrowUp) && g.direction != dirDown {
    g.nextDir = dirUp
}

// 移動タイミングで初めて適用
g.direction = g.nextDir
```

## Step 3: Gemini でスプライト画像を生成

ここからがAI全開の面白いところ。Gemini にこんなプロンプトを投げました：

```
Task: Create a Game Sprite Sheet for a top-down 2D snake game.
Character: A 16-bit pixel art eldritch tentacle horror (Cthulhu mythos theme)
with dark green slimy skin and deep purple suction cups.
Action: A sprite set containing the snake entity and collectible items.
Details:
* Row 1 (Head): A 4-frame animation of a grotesque, many-eyed head.
* Row 2 (Body): Straight and 90-degree curved tentacle body segments.
* Row 3 (Food/Items): Glowing Necronomicon, pulsing human brain, Elder Sign.
Tech Specs: Uniform grid layout, transparent background, SNES aesthetic.
```

生成された画像はそのままでは使えません。背景が透過ではなくブルーバックだったり、テキストラベルが入っていたり。ここは **Python + OpenCV** で後処理しました：

1. 黒枠ボックスの輪郭検出で個別スプライトの位置を特定
2. 青背景のピクセルを透明化（色距離ベースのマスキング）
3. scipy の連結成分ラベリングでノイズ除去
4. 128x128px にリサイズ

最終的に **頭部4方向 + 胴体2種 + カーブ4種 + 尻尾4方向 + 食べ物3種 + 背景** の計18枚のスプライトが揃いました。

## Step 4: スプライトをゲームに統合

Claude Code にスプライトファイルの一覧を見せると、自動的に適切なコードを生成してくれます。

```go
//go:embed assets/*.png assets/*.mp3
var assetsFS embed.FS
```

Go の `embed` パッケージでアセットをバイナリに埋め込むことで、**実行ファイル単体で動作する** ゲームになります。

### 体セグメントの自動判定

蛇の体のどの部分に直線/カーブ/尻尾のスプライトを使うかは、前後のセグメントの位置関係から自動判定：

```go
if prev.X == next.X {
    // 前後が同じX座標 → 縦方向の直線
    drawSprite(screen, bodyVertical, px, py)
} else if prev.Y == next.Y {
    // 横方向の直線
    drawSprite(screen, bodyHorizontal, px, py)
} else {
    // カーブ（曲がり角）
    curve := getCurveSprite(p, prev, next)
    drawSprite(screen, curve, px, py)
}
```

## Step 5: BGM と SE を追加

### BGM

BGM は AI で生成した MP3 を使用。Ebitengine の `audio` パッケージで無限ループ再生：

```go
stream, _ := mp3.DecodeWithoutResampling(bytes.NewReader(bgmData))
loop := audio.NewInfiniteLoop(stream, stream.Length())
bgmPlayer, _ = audioCtx.NewPlayer(loop)
bgmPlayer.SetVolume(0.3) // 音量30%
bgmPlayer.Play()
```

### SE（Pixabay から取得）

Claude Code に「Pixabay からフリー素材の SE を取得して」と指示。Pixabay は Cloudflare でブロックされていましたが、Claude Code は **`yt-dlp` で回避** するという判断を自力で行いました：

```bash
python3 -m yt_dlp --extract-audio --audio-format mp3 \
  -o "assets/se_eat.mp3" \
  "https://pixabay.com/sound-effects/game-eat-sound-83240/"
```

SE は起動時に PCM デコードしてメモリに保持し、再生時は即座に鳴る仕組み：

```go
// 起動時にデコード（1回だけ）
seEatData = decodeSE("se_eat")

// 再生時は毎回新しいプレイヤーを作成（重複再生対応）
func playSE(data []byte) {
    player, _ := audioCtx.NewPlayer(bytes.NewReader(data))
    player.SetVolume(0.5)
    player.Play()
}
```

## Step 6: 細かい調整

ここからは「遊んでみて → フィードバック → 修正」のサイクル。Claude Code との対話で即座に反映されます：

| フィードバック | 対応 |
|-------------|------|
| スプライトが小さすぎる | `gridSize` を 20 → 32 に拡大 |
| 背景が明るくてスプライトが見づらい | `ColorScale.Scale(0.3, 0.3, 0.3, 1)` で30%に減光 |
| ゲーム速度が速すぎる | `moveInterval` を 8 → 15 に変更 |
| 一時停止したい | Space キーでポーズ/再開トグル |
| 尻尾の画像が逆 | 向き判定ロジックを反転 |

各修正は **Claude Code に自然言語で指示するだけ** で完了。「背景が明るすぎる」→ 即座に `ColorScale` を提案・実装、のような流れです。

## 最終的なファイル構成

```
snakegame_with_ebitengine/
├── assets/
│   ├── background.png          # 背景 (640x480)
│   ├── bgm.mp3                 # BGM
│   ├── se_eat.mp3              # SE: 食べ物取得
│   ├── se_gameover.mp3         # SE: ゲームオーバー
│   ├── head_up/down/left/right.png    # 頭部 ×4
│   ├── tail_up/down/left/right.png    # 尻尾 ×4
│   ├── body_vertical/horizontal.png   # 胴体 ×2
│   ├── body_segment_curved_*.png      # カーブ ×4
│   └── food_*.png                     # 食べ物 ×3
├── docs/                       # ドキュメント群
├── main.go                     # ゲーム本体（約500行）
├── go.mod
└── README.md
```

**`main.go` 1ファイルに全ロジックが収まっています。** Go + Ebitengine の強みですね。

## Ebitengine の感想

バックエンド Go エンジニアがゲーム開発に入門するなら、Ebitengine は最高の選択肢だと感じました：

- **学習コスト最小**: `Update()` / `Draw()` / `Layout()` の 3 メソッドだけ覚えればいい
- **Go のエコシステムがそのまま使える**: `embed`, `image/png`, テストフレームワーク等
- **WASM 対応**: そのままブラウザゲームとして GitHub Pages に公開可能
- **ドキュメントが充実**: 公式サンプルが豊富

## Claude Code の使い方のコツ

今回の開発で得た知見：

### 1. Plan Mode を活用する

いきなり実装させるより、まず `/plan` でプランを立てさせると精度が上がります。Claude Code は内部で Explore エージェントと Plan エージェントを起動し、既存コードベースを分析したうえで設計を提案してくれます。

### 2. コミットのタイミングを指示する

Claude Code はデフォルトで勝手にコミットしません。「日本語コメントのみコミットして、スプライト変更はコミットしないで」のような細かい指示にも対応します。中間バージョンを復元してコミット → 最新版に戻す、という操作も自動でやってくれました。

### 3. ドキュメントも一緒に作らせる

「初心者向けの Ebitengine ガイドを作って」「マイルストーンファイルを作って」と指示すると、コードと整合性の取れたドキュメントを生成します。作業ログやセッションログも自動で書いてくれるので、後から振り返るのが楽です。

### 4. エラー対応を任せる

ビルドエラーが出ても Claude Code が自力で原因特定 → 修正してくれます。今回も `initBGM()` の閉じ括弧欠落や `go mod tidy` の実行忘れなどを自動で解決しました。

## まとめ

| 項目 | 内容 |
|------|------|
| 開発時間 | 約2時間 |
| 手書きコード量 | ほぼ 0 行 |
| Claude Code への指示回数 | 約 20 回 |
| 最終コード量 | 約 500 行（main.go） |
| アセット数 | 画像 18 枚 + 音声 3 ファイル |

**Go エンジニアがゲーム開発未経験でも、AI ツールを組み合わせれば短時間で完成品が作れる** ことが実証できました。

特に Claude Code は「コードを書く」だけでなく、「プランを立てる」「ドキュメントを作る」「ビルドエラーを直す」「コミット操作をする」まで一貫して対応してくれるので、人間は **意思決定とフィードバック** に集中できます。

ゲーム開発に興味はあるけど手が出なかった Go エンジニアの方、ぜひ Ebitengine × Claude Code で遊んでみてください。

## 参考リンク

- [Ebitengine 公式サイト](https://ebitengine.org/)
- [Ebitengine サンプル集](https://ebitengine.org/en/examples/)
- [Claude Code](https://claude.ai/claude-code)
- [wasmgame テンプレート（GitHub Pages 公開用）](https://github.com/eihigh/wasmgame)
- [Pixabay Sound Effects](https://pixabay.com/sound-effects/)
