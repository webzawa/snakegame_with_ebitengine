# Snake Game with Ebitengine

Go言語のゲームエンジン [Ebitengine](https://ebitengine.org/) を使ったシンプルなスネークゲーム。

## スクリーンショット

（TODO: ゲーム画面のスクリーンショットを追加）

## 遊び方

- **矢印キー** で蛇の方向を操作
- **赤いブロック**（食べ物）を食べるとスコアが増え、蛇が伸びる
- **壁**や**自分の体**にぶつかるとゲームオーバー
- **Enter / Space** でリスタート

## 必要環境

- Go 1.22 以上
- macOS の場合: Xcode Command Line Tools

## セットアップ & 実行

```bash
git clone <repository-url>
cd snakegame_with_ebitengine
go mod tidy
go run .
```

## ゲーム仕様

| 項目 | 値 |
|------|-----|
| ウィンドウサイズ | 640 x 480 px |
| グリッドサイズ | 20 x 20 px |
| 移動速度 | 約 7.5 マス/秒 |
| 初期の蛇の長さ | 3 マス |

## プロジェクト構成

```
snakegame_with_ebitengine/
├── docs/
│   ├── ebitengine-guide.md   # Ebitengine 初心者ガイド
│   └── milestones.md         # 実装マイルストーン
├── go.mod
├── go.sum
├── main.go                   # ゲーム本体
└── README.md
```

## 技術スタック

- [Go](https://go.dev/)
- [Ebitengine v2](https://ebitengine.org/)
