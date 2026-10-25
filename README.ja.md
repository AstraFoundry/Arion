# Arion

[MyGo](https://github.com/egoist/mygo) ネイティブ Go UI で構築された、macOS および Windows 向けのローカルファースト音声入力・ピンイン IME。

[English](README.md) · [简体中文](README.zh-CN.md) · [日本語](README.ja.md)

[![License: AGPL-3.0](https://img.shields.io/badge/License-AGPL--3.0-blue.svg)](LICENSE)
[![Go 1.27.1](https://img.shields.io/badge/Go-1.27.1-00ADD8?logo=go&logoColor=white)](go.work)
[![MyGo v0.4.6](https://img.shields.io/badge/MyGo-v0.4.6-4B5563)](apps/desktop/go.mod)
[![Native Qwen3-ASR](https://img.shields.io/badge/ASR-Qwen3_1.7B-0F766E)](docs/decisions/020-native-qwen-asr.md)
[![Node 24](https://img.shields.io/badge/Node-24-339933?logo=nodedotjs&logoColor=white)](.nvmrc)
[![pnpm 10.33.0](https://img.shields.io/badge/pnpm-10.33.0-F69220?logo=pnpm&logoColor=white)](package.json)

![Arion の音声入力ウィンドウとフローティングディクテーションピル](docs/readme/hero.png)

![Arion の音声ディクテーション動作](docs/readme/demo.gif)

## 背景と解決する課題

入力メソッド（IME）は OS のテキスト入力経路の最下層に位置し、チャット、ドキュメント作成、検索、AI へのプロンプト指示など、日常的な個人のコンテキストに直接関与します。既存のデスクトップ音声入力には主に2つの技術的課題があります。

1. **クラウド型音声入力はリモートサーバーでの認識と後処理に依存する**ため、生のマイク音声、編集履歴、ウィンドウのコンテキストを外部へ送信する必要があります。
2. **OS 標準のオフライン音声入力は個人のコンテキストへの適応が限定的**であり、文中で中国語と英語を切り替える発話（コードスイッチング）、専門用語の学習、フィラー除去、アプリケーションに応じたフォーマット整形に課題があります。

[AstraFoundry](https://astrafoundry.si) は、**オープンソース、ローカルファースト、セルフホスト可能**なデスクトップ音声・ピンイン入力環境として Arion を開発しています。ストリーミング音声認識、オフライン全ピンイン入力、修正履歴の学習、オプションの文脈推敲をデバイス上で実行し、生の音声データと暗号鍵をユーザーの手元に保持します。

### 対象とするユースケース

- **オフライン音声認識と入力フィールドの安全境界**
  モデルのダウンロードと固定 SHA-256 ダイジェスト検証が完了した後は、音声認識はすべてデバイス上でオフライン動作します。マイク音声はアクティブなセッション中のみメモリに保持され、終了時に破棄されます（ディスク保存や外部送信は行いません）。パスワード欄や安全性が不明な入力フィールドは自動的にスキップされます（[DSK-08](specs/001-desktop-input.md)）。
- **距離別に較正された小声入力向けソフトウェアゲイン**
  話す距離に応じたソフトウェア入力ゲイン（15 cm の近接発話向け `−6 dB` から 120 cm 向け `+12 dB` まで 3 dB 刻み、[DSK-12](specs/001-desktop-input.md)）を備えています。OS 全体のマイク音量を変更することなく、Arion のキャプチャ信号のみを増幅してオフィスでの小声入力に対応します。
- **文中での中英混在発話とドラフト分離**
  事前に言語を選択することなく、1つの文中で中国語、英語、技術用語を混在して認識できます（例：*「把这个 PR merge 到 main 之前，先跑一下 CI」*）。発話中のストリーミングドラフトはフローティングピル内に留まり、キーを離した時点で確定文のみが一度で挿入されます。
- **オンデバイスの編集学習とオプションの2段階推敲**
  - **編集学習（Edit Learning）**：`AXValue` を公開する macOS ネイティブエディタにおいて、挿入後の誤認識語を手動で修正すると、その修正ペアをローカルで抽出して暗号化語彙に保存し、ピルからワンクリックで取り消すことができます（[ADR 014](docs/decisions/014-edit-learning.md)）。
  - **2段階の文脈推敲**：有効化すると、キーを離した後に分離された `arion-llm`（`llama.cpp`）ワーカー（または明示的に選択したクラウドモデル）がフィラーを除去し、文中の言い直しを反映し、アクティブウィンドウやカーソル周辺の用語を参照して固有名詞の曖昧性を解消したうえで、シーン（チャット、メール、文書、プロンプト）に合わせて整形します。

## 主な技術的特徴

| 領域 | 実装アプローチ |
| --- | --- |
| **オンデバイス推論とゼロ知識同期** | 認識と語彙学習はローカルで実行。個人語彙は macOS キーチェーンまたは Windows DPAPI の鍵を用いた AES-256-GCM で暗号化されます。同期リレー（`services/sync`）は暗号文とルーティングメタデータのみを保存し、Docker でセルフホスト可能であり、復号鍵を保持しません。 |
| **グローバル音声入力とシステム IME の統合** | グローバルキー（macOS は右 `⌥`、Windows は右 `Alt`）による外部エディタへの直接入力と、システム IME ホスト（macOS `ArionInput.app` / Windows TSF `ArionTSF.dll`）によるオフライン全ピンイン入力・`Shift` 中英切り替えの両方に対応します。 |
| **ネイティブ Go デスクトップシェル** | Electron や WebView を使わず [MyGo](https://github.com/egoist/mygo) ネイティブ Go UI で構築。メニューバー/システムトレイに常駐し、macOS フルスクリーン Space に追従するクリックスルー可能なピルがエディタのフォーカスを奪わずに動作します。 |
| **決定論的なセッション管理とエラー境界** | 認識ドラフトと確定テキストを分離し、キャンセルやフォーカス移動時には進行中の世代（Generation）を即座に無効化します。モデル未準備や権限不足は明示的なエラーコードを返し、無断でクラウド認識へフォールバックしません。 |
| **フルスタック・オープンソース** | [AGPL-3.0](LICENSE) ライセンス。デスクトップアプリ、ネイティブ音声・推敲ワーカー、暗号化同期リレー、アップデート配信サービス、評価ベンチマークを単一リポジトリで提供します。 |

## アーキテクチャと技術機能

Apple silicon 上の macOS と Windows x64 が主要なデスクトップターゲット（`P0`）です。Intel macOS はサポート対象外であり（[ADR 021](docs/decisions/021-apple-silicon-only-macos.md)）、iOS と Android は将来のターゲット（`P2`）です。

- **グローバルディクテーションキーとネイティブ IME ホスト** — ディクテーションキー（macOS はデフォルトで右 `⌥`、Windows は右 `Alt`）を押し続けて話し、離すと一度で挿入されます。短くタップすると次の押下まで聞き取りを継続します（`Esc` でキャンセル、[DSK-02](specs/001-desktop-input.md)）。オフライン全ピンイン入力と暗号化学習を備えた `ArionInput.app` macOS InputMethodKit ホスト（[ADR 006](docs/decisions/006-macos-input-method-host.md)、[ADR 017](docs/decisions/017-offline-pinyin-keyboard.md)）と Windows x64 TSF IME ホスト（[Windows IME](docs/operations/windows-input-method.md)）を含みます。
- **オンデバイスのネイティブ音声認識** — Qwen3-ASR 1.7B は Apple silicon 上ではネイティブ Swift MLX 8-bit、Windows x64 上ではネイティブ `speech.cpp` Q8_0（CPU/Vulkan）を通じて動作します（[ADR 020](docs/decisions/020-native-qwen-asr.md)）。
- **保守的な補正と編集学習** — 決定論的ルール（`arion-rules-v4`）と承認済み暗号化語彙を適用。macOS のネイティブエディタでは挿入テキストへの修正をアクセシビリティ経由で検知してオンデバイス学習します（[ADR 014](docs/decisions/014-edit-learning.md)）。
- **オプションの2段階ローカル/クラウド推敲** — キーを離した後に1回だけ実行され（[ADR 023](docs/decisions/023-final-only-polishing.md)）、共通の2段階セレクタ/フォーマッタパイプライン（[ADR 024](docs/decisions/024-shared-polishing-pipeline.md)）、シーンに応じたフォーマット（**アプリに合わせる**、[ADR 012](docs/decisions/012-context-scene-selection.md)）、およびオンデバイスでのカーソル・画面・履歴からの用語抽出（[ADR 013](docs/decisions/013-rewrite-context-on-device.md)）を使用します。ローカル推敲は分離された `arion-llm`（`llama.cpp`）ワーカープロセス内で Qwen3.5-2B（`Q4_K_M` GGUF）を実行します（[ADR 018](docs/decisions/018-staged-gguf-rewriting.md)、[ADR 025](docs/decisions/025-windows-local-polishing.md)、[ローカル推敲設定](docs/operations/local-rewrite.md)）。オプションのクラウド推敲は生の音声をデバイス内に保ったまま、明示的に選択されたプロバイダーを使用します（[ADR 022](docs/decisions/022-cloud-polishing.md)）。
- **ネイティブ Go デスクトップシェル（FSD）** — Go での Feature-Sliced Design（[ADR 001](docs/decisions/001-desktop-native-boundaries.md)、[ADR 007](docs/decisions/007-sunbeam-design-system.md)）を採用して [MyGo](https://github.com/egoist/mygo)（`v0.4.6`）で構築。初回オンボーディング、ローカル診断 ZIP エクスポート、リアルタイム切り替え可能な英語 / 簡体字中国語 UI ローカリゼーション（[i18n](docs/frontend/i18n.md)）を備えています。
- **ローカル暗号化、セルフホスト同期リレー、署名付きアップデート** — AES-256-GCM による語彙暗号化（[ADR 005](docs/decisions/005-local-encryption.md)）、`noop` / `oidc` ゲートを備えた `services/sync` 暗号文リレー（[ADR 015](docs/decisions/015-hosted-relay-account-authorization.md)）、ループバック PKCE サインイン（[アカウントサインイン](docs/project/account-signin.md)）、および `services/updates` が配信する Ed25519 署名付きアップデートチャネル（[ADR 019](docs/decisions/019-center-update-channels.md)、[deploy/center](deploy/center/README.md)）を提供します。

## 現在の状態と制限事項

Arion は現在、macOS（`arm64`）で動作するメニューバー音声入力アプリであり、Windows（`amd64`）向けの初期プレビューを備えています（[リリースステータス](docs/operations/release-status.md)）。ネイティブ音声認識、ローカル推敲、および IME プリミティブは開発ホストと CI で検証済みですが、クリーンマシンおよび幅広いサードパーティエディタでの受け入れ検証は進行中です。

未実装または進行中の項目：

- **ローカル候補ランキング（`RNK-01`）** — ランキング契約と評価ハーネスは存在しますが、現在の認識器は単一の仮説のみを返すため、候補ランキングは有効化されていません。
- **デスクトップメモリ同期クライアントとデバイス登録（`MEM-01` / `SYNC-10`）** — `services/sync` リレーおよびデスクトップ OIDC サインイン画面・サービスは実装済みですが、デスクトップ SQLite アウトボックスとデバイス登録は未完了です。
- **IME と Windows ホストの広範な受け入れ検証（`IMK-01` / `TSF-01` / `RWT-RUN-01`）** — macOS `ArionInput.app`、Windows `ArionTSF.dll`、および Windows ローカル GGUF 推敲は分離テストと開発ホストチェックを通過していますが、クリーンマシンでのインストール、広範なエディタ互換性マトリクス、および Windows の編集学習は未完了です。
- **書き換えのブラインド人間評価と汎用 LLM 補正（`LRW-REVIEW-01` / `COR-01`）** — 開発ベンチマークは [`benchmarks/results`](benchmarks/README.md) に記録されていますが、ブラインド人間評価（`LRW-A06`）は未完了です。また、推敲パイプライン外のスタンドアロン LLM 補正は採用されていません。
- **Windows パブリッシャー署名と公開チャネルアップグレード検証（`REL-01`）** — macOS `arm64` の Developer ID コード署名、公証（notarization）、および IMK プロビジョニングプロファイル検証は実装済みです（[デプロイメント](docs/operations/deployment.md)）。Windows Authenticode 署名とクリーンホストでの旧バージョンから新バージョンへのチャネルアップグレード検証は未完了です。

正確な検証エビデンスとタスクゲートについては、[エージェントハンドオフ](docs/project/agent-handoff.md)、[リリースステータス](docs/operations/release-status.md)、および[検証レポート](docs/quality/validation.md)を参照してください。

## クイックスタート

### 前提条件

- `cgo` が有効化された **Go 1.27.1**（`go.work` は Go の自動ツールチェーン選択を使用します。`packages/core` と `services/sync` は `GOWORK=off` でも単体チェックされます）。
- **macOS（`arm64`）**：完全な **Xcode 26.4.1** アプリケーション（Swift 6.3.1）と **Metal Toolchain** コンポーネント。Command Line Tools のみでは `ArionASR` をビルドできません：
  ```sh
  sudo xcode-select -s /Applications/Xcode.app/Contents/Developer
  sudo xcodebuild -license accept && sudo xcodebuild -runFirstLaunch
  xcodebuild -downloadComponent MetalToolchain
  ```
- **Windows（`amd64`）**：ネイティブ C/C++ ツールチェーン、CMake、ネイティブキーボード/cgo ビルド用の MinGW-w64 GCC（`CC=x86_64-w64-mingw32-gcc`）、およびインストーラービルド用の NSIS。
- リポジトリチェック、評価ツール、および静的マーケティングサイト（`apps/site`）用の **Node 24**（`.nvmrc`）と**バージョン固定された `pnpm@10.33.0`**（`package.json`）。`npm` と `yarn` は拒否されます。

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

初回実行時、`make desktop-dev` はオンボーディングを開き、マイクとアクセシビリティの権限を要求し、デフォルトの認識モデルバンドルを `~/Library/Application Support/Arion/models/`（macOS）または `%AppData%\Arion\models\`（Windows）にダウンロードします。モデルのプロビジョニング完了後、ディクテーションはオフラインで動作します。すべてのベンチマーク、評価、診断ターゲットを一覧表示するには `make help` を実行してください。

### macOS 開発ビルドと TCC 権限の診断（`make desktop-dev`）

> 詳細な手順は [`docs/operations/local-dev.md` — macOS development build and TCC permission diagnostics](docs/operations/local-dev.md#macos-development-build-and-tcc-permission-diagnostics) を参照してください。

- **初回のコールドビルドは 2〜5 分間出力なしで進行します**：`scripts/stage-asr-runtime.mjs` は `xcodebuild -quiet` を実行して SwiftPM パッケージを取得し、`mlx-swift` と Metal シェーダーをコンパイルします。`Staged native ASR for darwin-arm64.` が表示されるまで `Ctrl+C` で中断せずにお待ちください。
- **バンドルの場所とメニューバー常駐モード**：`make desktop-dev` は `~/Library/Caches/Arion/desktop-dev/<checkout-hash>/darwin-arm64/Arion Dev.app`（起動時に出力）に `Arion Dev`（`si.astrafoundry.arion.dev`）をビルドして起動します。メニューバーアプリ（`LSUIElement`、Dock アイコンやメインウィンドウなし）のため、画面右上のメニューバーアイコンを確認してください。過去の実行で残った `apps/desktop/.mygo/dev/.../Arion Dev.app` は署名ハッシュ（`CDHash`）が異なるため開かないでください。
- **「終了して再度開く」による権限ループの回避**：macOS はアクセシビリティ、入力監視、画面収録の権限をバイナリの `CDHash` に紐付けます。`make desktop-dev` の実行やライブリロードのたびに `Arion Dev.app` がアドホック再署名され、システム設定のスイッチがオンに見えても以前の権限が無効になります。システム設定で権限をオンにした後は、メニューバーから `Arion Dev` を終了し、`make desktop-dev` を再実行するのではなく、**ビルド済みの同じキャッシュバンドル**を `open` で開き直してください：
  ```sh
  open "~/Library/Caches/Arion/desktop-dev/<checkout-hash>/darwin-arm64/Arion Dev.app"
  ```
- **スタックした開発用権限のリセット**：
  - **マイク**（システム設定に `+` ボタンがなく、拒否後は再プロンプトされない）：`tccutil reset Microphone si.astrafoundry.arion.dev` を実行し、`Arion Dev.app` を開き直して **設定 › 一般** のマイク行で **承認** をクリックします。
  - **アクセシビリティ / 入力監視**（スイッチがオンでも未承認になる、または `hold_register PERMISSION_DENIED` が出る）：`tccutil reset Accessibility si.astrafoundry.arion.dev` と `tccutil reset ListenEvent si.astrafoundry.arion.dev` を実行し、再承認後に `open` で開き直します。
  - **画面収録**（ドラッグ用タイルがなく、キャッシュパスが隠しディレクトリにある）：システム設定 › 画面収録とシステムオーディオ録音で **+** をクリックし、`⌘⇧G` を押して起動時に表示された `~/Library/Caches/Arion/desktop-dev/<checkout-hash>/darwin-arm64` を貼り付け、ファイル一覧から `Arion Dev.app` を選択（必要に応じて `⌘2` でリスト表示に切り替え）してスイッチをオンにし、`open` で開き直します。
- **`desktop-dev` におけるアドホック署名の制限**：システム IME（`ArionInput.app`）のインストールはアドホック開発ビルドでは仕様により `ARTIFACT_UNVERIFIED` で拒否されます（`MACOS_IME_PROVISIONING_PROFILE` を設定した Developer ID 署名付き `make build desktop` が必要です）。また、音声入力後の編集学習（[ADR 014](docs/decisions/014-edit-learning.md)）は `AXValue` を読み取るため、ネイティブエディタ（**メモ**、**テキストエディット**）でのみ動作し、Electron アプリ（Cursor、VS Code）やターミナルでは動作しません。

## ドキュメント

### プロジェクトと仕様

- [Agentic Coding Guidelines (`AGENTS.md`)](AGENTS.md) — コントリビューターおよびコーディングエージェント向けのルール、境界、ワークフロー。
- [ドキュメントインデックス (`docs/README.md`)](docs/README.md) — すべてのドキュメント領域のトップレベルマップ。
- [プロダクトスコープ (`docs/project/README.md`)](docs/project/README.md) — 確定済みスコープ、明示的な境界、およびスタックステータス表。
- [アーキテクチャ (`docs/project/architecture.md`)](docs/project/architecture.md) — データフロー、推敲パイプライン、ランタイム構成、プライバシー境界。
- [リポジトリオーナーシップ (`docs/project/repository.md`)](docs/project/repository.md) — Go ワークスペースモジュール、FSD/DDD パッケージ境界、作業ルール。
- [仕様書 (`specs/README.md`)](specs/README.md) — 規範的要件（`001`〜`009`）および受け入れシナリオ ID。
- [エージェントハンドオフ (`docs/project/agent-handoff.md`)](docs/project/agent-handoff.md) — 実装済みマイルストーン、未完了の `P0`/`P1`/`P2` タスク、完了ゲート。
- [アーキテクチャ決定記録 (`docs/decisions/README.md`)](docs/decisions/README.md) — ADR `001`〜`025`。

### エンジニアリング、運用、評価

- [フロントエンド規約 (`docs/frontend/README.md`)](docs/frontend/README.md) — ネイティブ Go における Feature-Sliced Design（FSD）と[インターフェースローカリゼーション (`docs/frontend/i18n.md`)](docs/frontend/i18n.md)。
- [バックエンド規約 (`docs/backend/README.md`)](docs/backend/README.md) — `packages/core` および `services/sync` 向けのドメイン駆動設計（DDD）ルール。
- [ローカル開発 (`docs/operations/local-dev.md`)](docs/operations/local-dev.md) — ツールチェーン、データ保存場所、環境変数、トラブルシューティング。
- [デプロイメントとリリースゲート (`docs/operations/deployment.md`)](docs/operations/deployment.md)、[リリースステータス (`docs/operations/release-status.md`)](docs/operations/release-status.md)、[Arion センター (`deploy/center/README.md`)](deploy/center/README.md)、および[同期リレー運用ランブック (`deploy/sync/README.md`)](deploy/sync/README.md)。
- [品質とテスト (`docs/quality/README.md`)](docs/quality/README.md)、[検証レポート (`docs/quality/validation.md`)](docs/quality/validation.md)、および[ネイティブ E2E (`docs/quality/e2e.md`)](docs/quality/e2e.md)。
- [リサーチエビデンス (`docs/research/README.md`)](docs/research/README.md)、[ベンチマーク手法 (`docs/research/benchmark-methodology.md`)](docs/research/benchmark-methodology.md)、[評価ランナー (`tools/evaluation/README.md`)](tools/evaluation/README.md)、および[ベンチマーク結果 (`benchmarks/README.md`)](benchmarks/README.md)。

## コントリビューター

[![AstraFoundry/Arion contributors](https://contrib.rocks/image?repo=AstraFoundry/Arion)](https://github.com/AstraFoundry/Arion/graphs/contributors)

## Star 履歴

[![AstraFoundry/Arion Star History Chart](https://api.star-history.com/svg?repos=AstraFoundry/Arion&type=Date)](https://star-history.com/#AstraFoundry/Arion&Date)

## ライセンス

Arion は [AstraFoundry](https://github.com/AstraFoundry) によって開発され、[AGPL-3.0](LICENSE)（[ADR 009](docs/decisions/009-agpl-license.md)）の下でライセンスされています。バグ報告は [GitHub Issues](https://github.com/AstraFoundry/Arion/issues) までお願いします。
