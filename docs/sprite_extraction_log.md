# スプライト抽出 作業ログ

## 日付
2026-03-28

## 概要
キャラクターシート画像（`Gemini_Generated_Image_xe5vywxe5vywxe5v.png`）からゲーム用スプライトを抽出し、`assets/` ディレクトリに保存した。
使用Skills `/image-utils`

## プロンプト
Task: Create a Game Sprite Sheet for a top-down 2D snake game.
Character: A 16-bit pixel art eldritch tentacle horror (Cthulhu mythos theme) with dark green slimy skin and deep purple suction cups.
Action: A sprite set containing the snake entity and collectible items, displayed in a grid layout.
Details:
* Row 1 (Head): A 4-frame animation of a grotesque, many-eyed head writhing and opening its maw.
* Row 2 (Body): Straight and 90-degree curved tentacle body segments.
* Row 3 (Food/Items): Collectible items including a glowing Necronomicon (tome), a pulsing human brain, and a mystical Elder Sign.
* Top-down perspective.
Tech Specs: Uniform grid layout, clean pixel edges, transparent background, SNES aesthetic, dark cosmic horror color palette. No anti-aliasing.

## 元画像
- ファイル: `/Users/mkt/Downloads/Gemini_Generated_Image_xe5vywxe5vywxe5v.png`
- サイズ: 2760x1504px (RGBA)
- 背景: ブルーバック
- 各素材は黒枠（黒い矩形ボーダー）で囲まれている

## 抽出手順

### 1. 黒枠ボックスの検出
- OpenCV を使用し、黒ピクセル（閾値 < 40）を二値化
- モルフォロジー演算（膨張）で枠線を接続
- 輪郭検出（`findContours`）で矩形領域を特定
- 画像右端で途切れているボックス（x + w >= 画像幅 - 5）は除外

### 2. 青背景の除去
- 画像左上隅から青背景の平均色をサンプリング（R=27, G=83, B=210）
- 以下の条件で青ピクセルを透明化:
  - 背景色との色距離 < 120
  - B > R+30 かつ B > G+30 かつ B > 80（青優勢ピクセル）
  - B > R+60 かつ B > G+60（強い青優勢）
- 黒ピクセル（R,G,B < 45）も除去（枠線の残留）
- 白/明灰色ピクセル（R,G,B > 170）も除去（テキストラベル除去）

### 3. スプライトの整形
- scipy `ndimage.label` で連結成分を検出し、最大の連結領域のみ保持（ノイズ除去）
- alpha チャンネルに対して収縮→膨張（3x3カーネル）でエッジフリンジを除去
- `getbbox()` で透明部分をトリミング
- 128x128px に NEAREST 補間でリサイズ（ピクセルアート向け）

### 4. 後処理（青フリンジ除去）
- 全スプライトを再スキャンし、残存する青優勢ピクセル（B > R+25 かつ B > G+25 かつ B > 80）を透明化

## 抽出したスプライト一覧

### 頭部（HEAD: WRITHING ANIMATION）
| ファイル | サイズ | 説明 |
|---------|-------|------|
| `head_frame1.png` | 128x128 | 頭部アニメーション フレーム1 |
| `head_frame2.png` | 128x128 | 頭部アニメーション フレーム2 |
| `head_frame3.png` | 128x128 | 頭部アニメーション フレーム3 |
| `head_frame4.png` | 128x128 | 頭部アニメーション フレーム4 |
| `head_frame5.png` | 128x128 | 頭部アニメーション フレーム5 |

※ 6枚目は画像右端で途切れていたためスキップ

### 胴体セグメント（BODY SEGMENTS: STRAIGHT & CURVED）
| ファイル | サイズ | 説明 |
|---------|-------|------|
| `body_horizontal.png` | 128x128 | 水平方向の胴体 |
| `body_vertical.png` | 128x128 | 垂直方向の胴体 |
| `body_half.png` | 128x128 | ハーフサイズの胴体 |
| `body_curve1.png` | 128x128 | カーブ胴体1 |
| `body_curve2.png` | 128x128 | カーブ胴体2 |

※ 6枚目（右端）は途切れていたためスキップ

### 食べ物アイテム（COLLECTIBLE ITEMS）
| ファイル | サイズ | 説明 |
|---------|-------|------|
| `food_necronomicon.png` | 128x128 | 光るネクロノミコン |
| `food_brain.png` | 128x128 | 脈動する人間の脳 |
| `food_elder_sign.png` | 128x128 | 神秘のエルダーサイン |

## 追加作成した派生画像

### 左右反転
| ファイル | 元画像 | 説明 |
|---------|-------|------|
| `body_curve1_flip.png` | `body_curve1.png` | 左右反転 |
| `body_curve2_flip.png` | `body_curve2.png` | 左右反転 |

### 90度回転
| ファイル | 元画像 | 説明 |
|---------|-------|------|
| `body_vertical_rot90.png` | `body_vertical.png` | 時計回り90度回転 |

## 使用ツール・ライブラリ
- Python 3
- OpenCV (`cv2`) 4.11.0 — 輪郭検出・モルフォロジー演算
- Pillow (`PIL`) — 画像の読み込み・加工・保存
- NumPy — ピクセル配列操作
- SciPy (`ndimage`) — 連結成分ラベリング

## 出力先
`/Users/mkt/myApps/snakegame_with_ebitengine/assets/`
