# 実装マイルストーン

## 概要

Ebitengine を使ったクトゥルフ風スネークゲームの実装マイルストーン。
各マイルストーンは段階的に機能を追加し、各段階で動作確認可能な状態を保つ。

## マイルストーン一覧

### M0: プロジェクトセットアップ
- [x] `go.mod` 作成
- [x] `go mod tidy` で依存解決
- [x] ビルド成功確認

### M1: 蛇の表示と移動
- [x] `Game` 構造体の定義
- [x] 蛇をグリッド上に描画
- [x] 一定間隔で右方向に自動移動

### M2: キー入力
- [x] 矢印キーで方向変更
- [x] 180度反転の防止（右に進んでいるとき左は無視）

### M3: 食べ物と成長
- [x] 食べ物をランダム位置に描画
- [x] 蛇が食べ物に到達したら蛇が1マス伸びる
- [x] スコア加算と画面表示

### M4: 衝突判定
- [x] 壁（画面端）との衝突でゲームオーバー
- [x] 自身の体との衝突でゲームオーバー

### M5: ゲームオーバー & リスタート
- [x] ゲームオーバー画面の表示（半透明オーバーレイ付き）
- [x] Enter キーでゲームリスタート

### M6: スプライト画像の統合
- [x] Gemini で生成したスプライトシートから素材を抽出（128x128px）
- [x] 頭部: 方向別4枚（head_up/down/left/right）
- [x] 胴体: 直線2枚（vertical/horizontal）+ カーブ4枚
- [x] 食べ物: 3種類（ネクロノミコン・脳・エルダーサイン）からランダム出現
- [x] 背景画像（background.png）を減光して描画
- [x] `embed` でアセットをバイナリに埋め込み

### M7: ゲームバランス調整
- [x] グリッドサイズ拡大（20px → 32px）でスプライトの視認性向上
- [x] 移動速度を低下（moveInterval 8 → 15、約4回/秒）
- [x] 背景を30%に減光してスプライトとのコントラスト改善

### M8: 一時停止機能
- [x] Space キーでポーズ/再開をトグル
- [x] ポーズ中にオーバーレイ + "PAUSED" メッセージ表示

### M9: BGM の追加
- [x] BGMファイル（`bgm.mp3`）をアセットに追加
- [x] `audio/mp3` でデコード → `NewInfiniteLoop` で無限ループ再生
- [x] 音量30%で再生

### M10: SE（効果音）の追加
- [x] Pixabay からフリー素材を取得
  - `se_eat.mp3` — 食べ物取得時（[game-eat-sound-83240](https://pixabay.com/sound-effects/game-eat-sound-83240/)）
  - `se_gameover.mp3` — ゲームオーバー時（[videogame-death-sound-43894](https://pixabay.com/sound-effects/videogame-death-sound-43894/)）
- [x] 起動時に MP3 → PCM デコードしてメモリ保持
- [x] 食べ物取得・ゲームオーバー時に SE 再生（音量50%）

### M11: 尻尾スプライトの追加
- [x] 尻尾スプライト（tail_up/down/left/right）を方向別マップで管理
- [x] 前のセグメントの反対方向で尻尾の向きを決定

### M12: 仕上げ
- [x] 日本語コメントの付与（初心者向け）
- [x] ドキュメント整備（ebitengine-guide, README, .gitignore）
- [x] 作業ログ・マイルストーン・セッションログ更新
- [ ] 全機能の最終動作確認

### M13: GitHub Pages 公開
参考: [eihigh/wasmgame](https://github.com/eihigh/wasmgame/blob/main/README_ja.md)

- [x] WASM ビルド確認（`GOOS=js GOARCH=wasm go build` → 16MB）
- [x] `wasm_exec.js` を Go ランタイムからコピー
- [x] `index.html` 作成（4:3 アスペクト比、ダークテーマ）
- [x] GitHub Actions ワークフロー作成（`.github/workflows/deploy.yml`）
- [x] リポジトリを Public に変更
- [x] GitHub Pages 有効化（Source: GitHub Actions）
- [x] デプロイ成功: https://webzawa.github.io/snakegame_with_ebitengine/
- [x] 素材は `embed` で埋め込み済み（WASM 対応済み）

## 今後の拡張案（未実装）
- [ ] ハイスコアの保存
- [ ] 難易度設定（速度の段階的上昇）
- [ ] タイトル画面
