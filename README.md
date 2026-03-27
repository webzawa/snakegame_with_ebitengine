# Snake Game - Eldritch Tentacle Horror

Go言語のゲームエンジン [Ebitengine](https://ebitengine.org/) を使ったクトゥルフ神話テーマのスネークゲーム。

**ブラウザで遊ぶ: https://webzawa.github.io/snakegame_with_ebitengine/**

## 遊び方

### デスクトップ（キーボード）
- **矢印キー** で蛇（触手）の方向を操作
- **Space** で一時停止 / 再開
- **Enter** でリスタート

### スマホ・タブレット（タッチ）
- **スワイプ** で方向操作
- **タップ** でリスタート

### 共通ルール
- **食べ物**（ネクロノミコン・脳・エルダーサイン）を食べるとスコアが増え、蛇が伸びる
- **壁**や**自分の体**にぶつかるとゲームオーバー

## デスクトップで実行

```bash
git clone https://github.com/webzawa/snakegame_with_ebitengine.git
cd snakegame_with_ebitengine
go mod tidy
go run .
```

### 必要環境

- Go 1.22 以上
- macOS の場合: Xcode Command Line Tools

## ゲーム仕様

| 項目 | 値 |
|------|-----|
| ウィンドウサイズ | 640 x 480 px |
| グリッドサイズ | 32 x 32 px |
| グリッド数 | 20 x 15 マス |
| 移動速度 | 約 4 マス/秒 |
| 初期の蛇の長さ | 3 マス |
| BGM 音量 | 30% |
| SE 音量 | 50% |

## プロジェクト構成

```
snakegame_with_ebitengine/
├── .github/workflows/
│   └── deploy.yml              # GitHub Pages 自動デプロイ
├── assets/
│   ├── background.png          # 背景画像
│   ├── bgm.mp3                 # BGM
│   ├── se_eat.mp3              # SE: 食べ物取得
│   ├── se_gameover.mp3         # SE: ゲームオーバー
│   ├── head_*.png              # 頭部スプライト (×4方向)
│   ├── tail_*.png              # 尻尾スプライト (×4方向)
│   ├── body_*.png              # 胴体スプライト (直線×2, カーブ×4)
│   └── food_*.png              # 食べ物スプライト (×3種)
├── docs/                       # ドキュメント群
├── main.go                     # ゲーム本体
├── index.html                  # ブラウザ版 HTML
├── wasm_exec.js                # Go WASM ランタイム
├── go.mod
└── README.md
```

## 技術スタック

| 役割 | 技術 |
|------|------|
| 言語 | [Go](https://go.dev/) |
| ゲームエンジン | [Ebitengine v2](https://ebitengine.org/) |
| デプロイ | GitHub Pages (WASM) |
| コード生成 | Claude Code |
| 画像生成 | Gemini |
| SE | [Pixabay](https://pixabay.com/sound-effects/) (ロイヤリティフリー) |

## ライセンス

SE素材は [Pixabay License](https://pixabay.com/service/license-summary/) に基づきます。
