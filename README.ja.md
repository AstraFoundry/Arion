<div align="center">

# Arion

[MyGo](https://github.com/egoist/mygo) ネイティブ Go UI で構築された、macOS および Windows 向けのローカルファースト音声入力・ピンイン IME。

[English](README.md) · [简体中文](README.zh-CN.md) · [日本語](README.ja.md)

<p align="center">
  <a href="https://x.com/AstraFoundry"><img src="https://img.shields.io/badge/X-%40AstraFoundry-000000?logo=x&logoColor=white" alt="X @AstraFoundry" style="display: inline-block; margin: 2px;" /></a>
  <a href="https://discord.com/invite/72kh465mMm"><img src="https://img.shields.io/badge/Discord-Join_Chat-5865F2?logo=discord&logoColor=white" alt="Discord" style="display: inline-block; margin: 2px;" /></a>
  <a href="https://applink.feishu.cn/client/chat/chatter/add_by_link?link_token=640t6cde-d38e-4a81-875b-7799e44b77ae"><img src="https://img.shields.io/badge/Feishu-Group_Chat-3370FF" alt="Feishu Group" style="display: inline-block; margin: 2px;" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-AGPL--3.0-blue.svg" alt="License: AGPL-3.0" style="display: inline-block; margin: 2px;" /></a>
</p>

<p align="center">
  <a href="go.work"><img src="https://img.shields.io/badge/Go-1.27.1-00ADD8?logo=go&logoColor=white" alt="Go 1.27.1" style="display: inline-block; margin: 2px;" /></a>
  <a href="apps/desktop/go.mod"><img src="https://img.shields.io/badge/MyGo-v0.4.6-4B5563" alt="MyGo v0.4.6" style="display: inline-block; margin: 2px;" /></a>
  <a href="docs/decisions/020-native-qwen-asr.md"><img src="https://img.shields.io/badge/ASR-Qwen3_1.7B-0F766E" alt="Native Qwen3-ASR" style="display: inline-block; margin: 2px;" /></a>
  <a href=".nvmrc"><img src="https://img.shields.io/badge/Node-24-339933?logo=nodedotjs&logoColor=white" alt="Node 24" style="display: inline-block; margin: 2px;" /></a>
  <a href="package.json"><img src="https://img.shields.io/badge/pnpm-10.33.0-F69220?logo=pnpm&logoColor=white" alt="pnpm 10.33.0" style="display: inline-block; margin: 2px;" /></a>
</p>

![Arion の音声ディクテーション動作](docs/readme/demo.gif)

</div>

## 概要

クラウド型音声入力ツールは生のマイク音声やウィンドウのコンテキストを外部サーバーへ送信する必要があり、一方で OS 標準のオフライン音声入力は文中での中英混在発話（コードスイッチング）、専門用語の学習、アプリケーションに応じた整形に課題があります。

[AstraFoundry](https://astrafoundry.si) は、**オープンソース、ローカルファースト、セルフホスト可能**なデスクトップ音声・ピンイン入力環境として **Arion** を開発しています。ストリーミング音声認識、オフライン全ピンイン入力、修正履歴の学習、文脈に応じた推敲をデバイス上で実行し、生の音声データと暗号鍵をユーザーの手元に保持します。

## 主な機能

- **ホールドして話す & ネイティブシステム IME** — どのアプリからでもディクテーションキー（デフォルトは macOS が右 `⌥`、Windows が右 `Alt`）を押したまま中国語（普通話）、英語、またはそれらが混在する技術用語（例：*「把这个 PR merge 到 main 之前，先跑一下 CI」*）を話し、離すと一度で挿入されます。短くタップすると次の押下まで聞き取りを継続します（`Esc` でキャンセル）。また、オフライン簡体字中国語フルピンイン入力と `Shift` 言語切り替えを備えたシステム IME ホスト（macOS `ArionInput.app` / Windows `ArionTSF.dll`）も含まれています。
- **オンデバイスのネイティブ音声認識** — Apple silicon 上ではネイティブ Swift MLX（8-bit）、Windows x64 上ではネイティブ `speech.cpp`（Q8_0、CPU/Vulkan）を介して Qwen3-ASR 1.7B をローカル実行します（[ADR 020](docs/decisions/020-native-qwen-asr.md)）。モデルのダウンロードと SHA-256 検証が完了した後は完全にオフラインで動作します。マイク音声はセッション中のみメモリに保持され、パスワード欄は自動的にスキップされ、小声入力向けの段階的ソフトウェア入力ゲイン（`−6 dB`〜`+12 dB`）も備えています。
- **編集学習とドラフト分離** — 発話中のストリーミングドラフトはフローティングピル内に留まり、キーを離した時点で確定文のみが一度で挿入されます。決定論的ルールと承認済み語彙をローカルで適用し、macOS のネイティブエディタでは挿入後の手動修正を自動学習してワンクリックで元に戻すことができます（[ADR 014](docs/decisions/014-edit-learning.md)）。
- **2段階の文脈推敲** — キーを離した後にオプションの2段階推敲を実行し（[ADR 023](docs/decisions/023-final-only-polishing.md)）、フィラーの除去、文中の言い直しの反映、アクティブウィンドウやカーソル周辺の用語を参照した固有名詞の曖昧性解消（[ADR 013](docs/decisions/013-rewrite-context-on-device.md)）、およびアプリケーションに応じた整形（**アプリに合わせる**、[ADR 012](docs/decisions/012-context-scene-selection.md)）を行います。macOS Apple silicon と Windows x64 上で `llama.cpp` により Qwen3.5-2B（`Q4_K_M` GGUF）をローカル実行できるほか（[ADR 018](docs/decisions/018-staged-gguf-rewriting.md)、[ADR 025](docs/decisions/025-windows-local-polishing.md)）、生の音声をデバイス内に保ったまま明示的オプトインのクラウドモデル（ChatGPT サインインまたは OpenAI、Anthropic、Gemini、DeepSeek、OpenRouter API キー）に接続することも可能です（[ADR 022](docs/decisions/022-cloud-polishing.md)）。
- **ネイティブ Go デスクトップシェル** — Electron や WebView を使わず、[MyGo](https://github.com/egoist/mygo)（`v0.4.6`）と Go における Feature-Sliced Design で構築されています（[ADR 001](docs/decisions/001-desktop-native-boundaries.md)、[ADR 007](docs/decisions/007-sunbeam-design-system.md)）。メニューバー/システムトレイに常駐し、フルスクリーン Space に追従するクリックスルー可能なピルがエディタのフォーカスを奪わずに動作します。英語 / 簡体字中国語 UI のリアルタイム切り替えに対応しています（[i18n](docs/frontend/i18n.md)）。
- **ゼロ知識同期と署名付きアップデート** — 個人語彙は macOS キーチェーンまたは Windows DPAPI に保存された鍵を用いて AES-256-GCM でローカル暗号化されます（[ADR 005](docs/decisions/005-local-encryption.md)）。Docker でセルフホスト可能かつ復号鍵を保持しない Go 暗号文リレー `services/sync`（[ADR 015](docs/decisions/015-hosted-relay-account-authorization.md)）と、Ed25519 署名付きの `stable`/`beta` チャネルを配信する `services/updates`（[ADR 019](docs/decisions/019-center-update-channels.md)）を提供します。

## 対応プラットフォーム

| プラットフォーム | アーキテクチャ | ステータス |
| --- | --- | --- |
| **macOS** | Apple silicon (`arm64`) | メニューバー音声入力、ネイティブ MLX 音声認識、ローカル GGUF 推敲、`ArionInput.app` IMK ホストが動作（Intel macOS はサポート対象外、[ADR 021](docs/decisions/021-apple-silicon-only-macos.md)） |
| **Windows** | 64-bit (`x64` / `amd64`) | 初期プレビュー版：ネイティブ `speech.cpp` 音声認識、ローカル GGUF 推敲、`ArionTSF.dll` IME ホストに対応 |

マイルストーンの進捗と未完了のリリースゲートについては、[リリースステータス](docs/operations/release-status.md)、[エージェントハンドオフ](docs/project/agent-handoff.md)、および[検証レポート](docs/quality/validation.md)を参照してください。

## クイックスタート

### 前提条件

- `cgo` が有効化された **Go 1.27.1**（`go.work` は自動ツールチェーン選択を使用）。
- **macOS（`arm64`）**：完全な **Xcode 26.4.1** アプリケーション（Swift 6.3.1）と **Metal Toolchain** コンポーネント：
  ```sh
  sudo xcode-select -s /Applications/Xcode.app/Contents/Developer
  sudo xcodebuild -license accept && sudo xcodebuild -runFirstLaunch
  xcodebuild -downloadComponent MetalToolchain
  ```
- **Windows（`amd64`）**：ネイティブ C/C++ ツールチェーン、CMake、MinGW-w64 GCC（`CC=x86_64-w64-mingw32-gcc`）、および NSIS。
- **Node 24**（`.nvmrc`）と**バージョン固定された `pnpm@10.33.0`**（`package.json`）。

### ビルドと実行

```sh
make install        # 固定バージョンの pnpm ワークスペースをインストール (--frozen-lockfile)
make check          # ドキュメント、JSON、Go レイヤー境界、core/sync および評価テスト
make desktop-check  # デスクトップテスト、キーボードチェック、go vet (cgo)
make desktop-dev    # ネイティブランタイムをステージングし、開発モードで Arion Dev を起動
make build desktop  # apps/desktop/build/ 配下に Arion.app + DMG (macOS) または NSIS インストーラー (Windows) をパッケージング
make e2e            # 実際のデスクトップセッションでのネイティブウィンドウライフサイクル E2E
make e2e-macos      # パッケージングされた Arion.app に対する製品シナリオテスト
make site-check     # マーケティングサイト (apps/site)：i18n、型、lint、テスト、静的エクスポート
```

初回実行時、`make desktop-dev` はオンボーディングを開き、マイクとアクセシビリティの権限を要求し、デフォルトの認識モデルバンドルを `~/Library/Application Support/Arion/models/`（macOS）または `%AppData%\Arion\models\`（Windows）にダウンロードします。macOS 開発ビルドの `CDHash` 権限挙動とトラブルシューティングについては、[ローカル開発 (`docs/operations/local-dev.md`)](docs/operations/local-dev.md#macos-development-build-and-tcc-permission-diagnostics)を参照してください。

## ドキュメント

| 領域 | エントリポイント |
| --- | --- |
| **概要・ガイドライン** | [ドキュメントインデックス (`docs/README.md`)](docs/README.md) · [プロダクトスコープ (`docs/project/README.md`)](docs/project/README.md) · [Agentic Coding Guidelines (`AGENTS.md`)](AGENTS.md) |
| **アーキテクチャ・仕様** | [アーキテクチャ (`docs/project/architecture.md`)](docs/project/architecture.md) · [リポジトリオーナーシップ (`docs/project/repository.md`)](docs/project/repository.md) · [仕様書 (`specs/README.md`)](specs/README.md) · [ADR (`docs/decisions/README.md`)](docs/decisions/README.md) |
| **フロントエンド・バックエンド** | [フロントエンド FSD 規約 (`docs/frontend/README.md`)](docs/frontend/README.md) · [UI ローカリゼーション (`docs/frontend/i18n.md`)](docs/frontend/i18n.md) · [バックエンド DDD 規約 (`docs/backend/README.md`)](docs/backend/README.md) |
| **運用・デプロイ** | [ローカル開発 (`docs/operations/local-dev.md`)](docs/operations/local-dev.md) · [デプロイメント (`docs/operations/deployment.md`)](docs/operations/deployment.md) · [リリースステータス (`docs/operations/release-status.md`)](docs/operations/release-status.md) · [Arion センター (`deploy/center/README.md`)](deploy/center/README.md) · [同期リレー運用 (`deploy/sync/README.md`)](deploy/sync/README.md) |
| **品質・ベンチマーク** | [品質とテスト (`docs/quality/README.md`)](docs/quality/README.md) · [ネイティブ E2E (`docs/quality/e2e.md`)](docs/quality/e2e.md) · [リサーチエビデンス (`docs/research/README.md`)](docs/research/README.md) · [評価ランナー (`tools/evaluation/README.md`)](tools/evaluation/README.md) · [ベンチマーク結果 (`benchmarks/README.md`)](benchmarks/README.md) |

## コントリビューター

[![AstraFoundry/Arion contributors](https://contrib.rocks/image?repo=AstraFoundry/Arion)](https://github.com/AstraFoundry/Arion/graphs/contributors)

## Star 履歴

[![AstraFoundry/Arion Star History Chart](https://api.star-history.com/svg?repos=AstraFoundry/Arion&type=Date)](https://star-history.com/#AstraFoundry/Arion&Date)

## ライセンス

Arion は [AstraFoundry](https://github.com/AstraFoundry) によって開発され、[AGPL-3.0](LICENSE)（[ADR 009](docs/decisions/009-agpl-license.md)）の下でライセンスされています。[X (@AstraFoundry)](https://x.com/AstraFoundry) のフォロー、[Discord](https://discord.com/invite/72kh465mMm) や [Feishu グループ](https://applink.feishu.cn/client/chat/chatter/add_by_link?link_token=640t6cde-d38e-4a81-875b-7799e44b77ae)への参加をお待ちしています。バグ報告は [GitHub Issues](https://github.com/AstraFoundry/Arion/issues) までお願いします。
