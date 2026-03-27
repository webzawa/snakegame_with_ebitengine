---
title: "Claude Codeでゲーム用スプライトシートから素材を自動抽出する"
emoji: "🐙"
type: "tech"
topics: ["claudecode", "gamedev", "ebitengine", "python", "pixelart"]
published: false
---

## はじめに

Ebitengine（Go製の2Dゲームエンジン）でクトゥルフ風スネークゲームを開発しています。

Gemini で生成したスプライトシート画像から、ゲームで使える個別スプライトを抽出する必要がありました。手作業でやると地味に面倒なこの工程を、**Claude Code の `/image-utils` スキル**を使って自動化した記録です。

完成したゲームのアセット構成:

```
assets/
├── background.png              # 背景（640x480）
├── head_up/down/left/right.png # 頭部×4方向
├── body_horizontal.png         # 胴体（水平）
├── body_vertical.png           # 胴体（垂直）
├── body_segment_curved_*.png   # カーブ×4方向
├── tail_up/down/left/right.png # 尻尾×4方向
├── food_necronomicon.png       # 食べ物①
├── food_brain.png              # 食べ物②
└── food_elder_sign.png         # 食べ物③
```

## 元画像: Gemini で生成したスプライトシート

以下のプロンプトで Gemini にスプライトシートを生成させました。

```
Task: Create a Game Sprite Sheet for a top-down 2D snake game.
Character: A 16-bit pixel art eldritch tentacle horror (Cthulhu mythos theme)
           with dark green slimy skin and deep purple suction cups.
Details:
  * Row 1 (Head): 4-frame writhing animation
  * Row 2 (Body): Straight and 90-degree curved tentacle segments
  * Row 3 (Food): Necronomicon, human brain, Elder Sign
Tech Specs: Uniform grid layout, clean pixel edges, transparent background,
            SNES aesthetic, dark cosmic horror color palette.
```

生成された画像は **2760x1504px** のブルーバック。各素材が黒枠で囲まれた状態です。

ここから「黒枠の検出 → 青背景の除去 → 個別PNG保存」を自動でやりたい。

## Claude Code の `/image-utils` スキルで抽出

Claude Code のチャットで `/image-utils` コマンドを呼び出し、画像を添付して指示を出しました。

```
/image-utils 添付画像はゲーム作成用キャラクターシート画像素材です。
ブルーバックにしています。素材は黒枠で囲っています。
素材を抽出して /path/to/assets/ に保存してほしい
```

Claude Code は以下の処理パイプラインを自動で構築・実行してくれました。

### Step 1: 黒枠ボックスの検出（OpenCV）

```python
gray = cv2.cvtColor(img_cv, cv2.COLOR_BGR2GRAY)
_, thresh = cv2.threshold(gray, 40, 255, cv2.THRESH_BINARY_INV)

kernel = np.ones((5,5), np.uint8)
dilated = cv2.dilate(thresh, kernel, iterations=2)

contours, _ = cv2.findContours(
    dilated, cv2.RETR_EXTERNAL, cv2.CHAIN_APPROX_SIMPLE
)
```

黒ピクセルを二値化 → 膨張で枠線を接続 → 輪郭検出で矩形領域を特定。画像右端で途切れているボックスは自動的に除外されました。

### Step 2: 青背景の除去

画像左上隅から背景色をサンプリングし、複数条件で青ピクセルを透明化:

```python
# 背景色との色距離
dist_blue = np.sqrt((r - avg_blue[0])**2 + (g - avg_blue[1])**2
                     + (b - avg_blue[2])**2)

is_blue = (
    (dist_blue < 120) |
    ((b > 100) & (r < 120) & (g < 120) & (b > r + 30) & (b > g + 30)) |
    ((b > r + 60) & (b > g + 60))
)
```

単純な閾値だけでなく、色距離ベースの判定を組み合わせることで、グラデーション部分の青も確実に除去できています。

### Step 3: ノイズ除去と整形

```python
# 最大連結成分のみ保持（テキストラベル等のゴミを除去）
labeled, num = ndimage.label(alpha > 0)
sizes = ndimage.sum(alpha > 0, labeled, range(1, num + 1))
largest = np.argmax(sizes) + 1
sprite_data[labeled != largest, 3] = 0

# エッジフリンジの除去（収縮→膨張）
alpha_eroded = cv2.erode(alpha_clean, kernel_small, iterations=1)
alpha_dilated = cv2.dilate(alpha_eroded, kernel_small, iterations=1)
```

スプライトシート上の「16x16 pixels」等のテキストラベルが混入する問題は、scipy の連結成分ラベリングで最大領域のみを残すことで解決。

### Step 4: リサイズと保存

```python
sprite_resized = sprite.resize((128, 128), Image.NEAREST)
```

ピクセルアートなので **NEAREST 補間**（最近傍法）でリサイズ。LANCZOS 等を使うとドットがボケるので注意。

## 派生画像の生成

抽出した素材から、ゲームに必要な派生画像も Claude Code に作ってもらいました。

### カーブ胴体: 極座標ワープ

水平ボディを90度の円弧に沿ってワープし、4方向のカーブを生成:

```python
# 出力ピクセルごとに極座標→元画像座標のマッピング
r = np.sqrt(x*x + y*y)
theta = np.arctan2(y, x)

# 角度→元画像のx座標、半径→元画像のy座標
src_x = (theta / (np.pi/2)) * (sw - 1)
src_y = strip_top + ((r_outer - r) / (r_outer - r_inner)) * strip_height
```

基本カーブを1つ作り、左右反転・上下反転で4方向を生成。

### 尻尾: テーパー処理

水平ボディの右端に向かって細くなるテーパー処理を適用:

```python
taper_start = int(w * 0.35)
for x in range(taper_start, w):
    t = (x - taper_start) / (w - taper_start)
    allowed_half = strip_half_h * (1.0 - t * 0.85)  # 85%収束
    # ...
```

回転・反転で4方向分を生成。

### 背景画像: アスペクト比維持リサイズ

ゲームウィンドウ（640x480）に合わせるリサイズで、最初はアスペクト比が崩れてしまいました。

```
# NG: 単純リサイズ → アスペクト比が変わる
resized = img.resize((640, 480), Image.LANCZOS)
```

修正後は高さ基準でリサイズし、左右を中央クロップ:

```python
# OK: アスペクト比維持 → 中央クロップ
ratio_h = 480 / img.height
new_w = int(img.width * ratio_h)       # 860x480
resized = img.resize((new_w, 480), Image.LANCZOS)

left = (new_w - 640) // 2
cropped = resized.crop((left, 0, left + 640, 480))  # 640x480
```

## ゲームコードでの利用

Ebitengine では `embed` ディレクティブでアセットをバイナリに埋め込めます:

```go
//go:embed assets/*.png assets/*.mp3
var assetsFS embed.FS

const (
    spriteSize = 128
    gridSize   = 32
)

var spriteScale = float64(gridSize) / float64(spriteSize) // 0.25
```

128px のスプライトを 32px のグリッドに縮小して描画。`embed` のおかげで実行ファイル単体で動作します。

## 実際のやりとりの流れ

今回の作業は Claude Code との対話で段階的に進めました:

1. **`/image-utils` でスプライト抽出を指示** → 初回は16x16で出力されたが小さすぎた
2. **「128x128にして」と修正指示** → 再抽出、青背景のフリンジが残る
3. **Claude Code が自動で後処理を追加** → 青フリンジ除去パスを追加して解決
4. **「左右反転した画像を作成」** → 画像を添付して指示、即座に生成
5. **「90度回転したものを作成」** → 同様に即座に生成
6. **「背景画像をリサイズして」** → ゲームコードを読んで640x480に自動判断
7. **「アスペクト比がおかしい」** → アスペクト比維持＋中央クロップに修正
8. **「カーブ胴体を作って」** → 極座標ワープで4方向を自動生成
9. **「尻尾も作って」** → テーパー処理で4方向を自動生成

ポイントは、Claude Code がゲームコード（`main.go`）を読んで `screenWidth=640, screenHeight=480` や `gridSize=32` を把握した上で、適切なサイズを自動判断してくれる点です。

## まとめ

| 工程 | 手作業の場合 | Claude Code の場合 |
|------|------------|-------------------|
| スプライト抽出 | 画像エディタで1枚ずつ切り出し | 自動検出・一括抽出 |
| 背景除去 | マジックワンドで選択→削除 | 色距離ベースで自動除去 |
| 派生画像生成 | 回転・反転を1枚ずつ | 指示1回で4方向分を生成 |
| リサイズ | ゲームの解像度を確認して手動 | コードを読んで自動判断 |

スプライトシートからの素材抽出は、ゲーム開発で地味に時間がかかる作業です。Claude Code の `/image-utils` スキルを使うことで、対話的に修正しながら素早くアセットを準備できました。

特に「画像を見せて自然言語で指示 → 即座にコードを生成・実行 → 結果をプレビュー → フィードバックして修正」というループが高速に回せるのが強みだと感じました。
