# Arion

[MyGo](https://github.com/egoist/mygo) ネイティブ Go UI で構築された、macOS および Windows 向けのローカルファースト音声入力ツール。

[English](README.md) · [简体中文](README.zh-CN.md) · [日本語](README.ja.md)

[![X @AstraFoundry](https://img.shields.io/badge/X-%40AstraFoundry-000000?logo=x&logoColor=white)](https://x.com/AstraFoundry)
[![Discord](https://img.shields.io/badge/Discord-Join_Chat-5865F2?logo=discord&logoColor=white)](https://discord.com/invite/72kh465mMm)
[![Feishu Group](https://img.shields.io/badge/Feishu-Group_Chat-3370FF)](https://applink.feishu.cn/client/chat/chatter/add_by_link?link_token=640t6cde-d38e-4a81-875b-7799e44b77ae)
[![License: AGPL-3.0](https://img.shields.io/badge/License-AGPL--3.0-blue.svg)](LICENSE)
[![Go 1.27.1](https://img.shields.io/badge/Go-1.27.1-00ADD8?logo=go&logoColor=white)](go.work)
[![MyGo v0.4.6](https://img.shields.io/badge/MyGo-v0.4.6-4B5563)](apps/desktop/go.mod)
[![Native Qwen3-ASR](https://img.shields.io/badge/ASR-Qwen3_1.7B-0F766E)](docs/decisions/020-native-qwen-asr.md)
[![Node 24](https://img.shields.io/badge/Node-24-339933?logo=nodedotjs&logoColor=white)](.nvmrc)
[![pnpm 10.33.0](https://img.shields.io/badge/pnpm-10.33.0-F69220?logo=pnpm&logoColor=white)](package.json)

![Arion の音声入力ウィンドウとフローティングディクテーションピル](docs/readme/hero.png)

![Arion の音声ディクテーション動作](docs/readme/demo.gif)

## 概要

Arion は、オンデバイスのストリーミング音声認識、決定論的ルールによるテキスト補正、個人語彙学習、コンテキスト認識型の書き換え、および暗号化メモリ同期を組み合わせたローカルファーストのデスクトップ音声入力ツールです。

- **デフォルトでオフライン動作** — 初回のモデルプロビジョニング完了後、音声認識とローカル書き換えはすべてデバイス上で完結します。
- **厳格なプライバシー境界** — 生のマイク音声はデバイス内に留まり、有効なコンテキストはクラウド整形を明示的に選択した場合にのみ送信されます（[005 プライバシー仕様](specs/005-privacy.md)）。
- **主要デスクトップターゲット** — Apple silicon 搭載 macOS（`arm64`）および Windows（`x64`）が主要な `P0` ターゲットです。Intel macOS はサポート対象外であり（[ADR 021](docs/decisions/021-apple-silicon-only-macos.md)）、iOS と Android は将来の `P2` ターゲットです。

## 主な機能

### 音声入力とシステム統合

- **グローバルディクテーションキー** — どのアプリからでもディクテーションキー（デフォルトは macOS が右 `⌥`、Windows が右 `Alt`）を押したまま中国語（普通話）、英語、またはそれらが混在する技術テキストを話し、離すとフォーカスされているエディタに一度で挿入されます。軽くタップすると次のキー押下まで聞き取りを継続し、`Esc` でキャンセルします（[DSK-02](specs/001-desktop-input.md)）。
- **macOS InputMethodKit ホスト（`ArionInput.app`）** — システム入力ソースとして直接統合され、音声コンポジションおよびオフライン簡体字中国語フルピンイン入力をサポートします（[ADR 006](docs/decisions/006-macos-input-method-host.md)、[ADR 017](docs/decisions/017-offline-pinyin-keyboard.md)）。
- **ネイティブ Go デスクトップシェル** — Electron や webview を使用せず、[MyGo](https://github.com/egoist/mygo)（`v0.4.6`）と Sunbeam デザインシステムを用い、Go における Feature-Sliced Design で構築されています（[ADR 001](docs/decisions/001-desktop-native-boundaries.md)、[ADR 007](docs/decisions/007-sunbeam-design-system.md)）。メニューバーポップオーバー、フルスクリーン Space に対応するクリックスルー可能なフローティングピル、初回起動オンボーディング、設定画面、ローカル診断 ZIP エクスポート、およびリアルタイム切り替え可能な英語 / 簡体字中国語 UI ローカリゼーション（[i18n](docs/frontend/i18n.md)）を備えています。

### 音声認識・補正・書き換え

- **ネイティブオンデバイス ASR** — Apple silicon 上ではネイティブ Swift MLX（8-bit）、Windows x64 上ではネイティブ `speech.cpp`（Q8_0）を介して Qwen3-ASR 1.7B をローカル実行し、ダイジェスト検証付きの自動モデルプロビジョニングを提供します（[ADR 004](docs/decisions/004-automatic-model-selection.md)、[ADR 020](docs/decisions/020-native-qwen-asr.md)）。
- **保守的な補正と編集学習** — 認識ドラフトと確定テキストを分離しつつ、決定論的ルール（`arion-rules-v3`）とユーザーが承認した語彙を適用します。macOS では、挿入されたテキストに対する編集がオンデバイスで学習され、ワンクリックで元に戻すことができます（[ADR 014](docs/decisions/014-edit-learning.md)）。
- **段階的ローカル書き換え（macOS Apple silicon）** — ローカルの2段階 GGUF パイプライン（`llama.cpp` による `Qwen3.5-2B`）を使用して発話テキストを整え、シーンに応じたフォーマット（**アプリに合わせる**）およびオンデバイスでのカーソル・画面・直近のディクテーションからのコンテキスト抽出に対応しています（[ADR 011](docs/decisions/011-live-local-rewriting.md)、[ADR 012](docs/decisions/012-context-scene-selection.md)、[ADR 013](docs/decisions/013-rewrite-context-on-device.md)、[ADR 018](docs/decisions/018-staged-gguf-rewriting.md)）。
- **オプションのクラウドポリッシング** — [Republic](https://github.com/AstraFoundry/republic) を介し、ChatGPT サインインまたは単一のプロバイダー API キー（OpenAI、Anthropic、Google Gemini、DeepSeek、OpenRouter）を使用した明示的オプトインの最終テキスト整形に対応しています。ローカルと同じ段階的な整形処理とコンテキスト設定を使用します。有効なコンテキストと想起された語彙を表記補正に使い、音声はデバイス内に留まります（[ADR 022](docs/decisions/022-cloud-polishing.md)、[クラウドポリッシング](docs/project/cloud-polishing.md)）。

### セキュリティ・同期・アップデート

- **ローカル暗号化とセルフホスト可能な同期リレー** — 個人語彙は macOS ログインキーチェーンまたは Windows DPAPI に保存された鍵を使用し、ローカルで AES-256-GCM により暗号化されます（[ADR 005](docs/decisions/005-local-encryption.md)）。`noop` および `oidc` アカウントゲートを備えたセルフホスト可能な Go 暗号文リレー `services/sync`（[ADR 015](docs/decisions/015-hosted-relay-account-authorization.md)）と、デスクトップのループバック PKCE サインインサービス（[アカウントサインイン](docs/project/account-signin.md)）も含まれています。
- **センター配信による署名付きアップデート** — アプリ内アップデーターは Arion センター（`api.arion.astrafoundry.si`、`services/updates`）上の `stable` および `beta` チャネルを確認し、インストール前に Ed25519 パッケージ署名を検証します（[ADR 010](docs/decisions/010-in-app-updates.md)、[ADR 019](docs/decisions/019-center-update-channels.md)、[deploy/center](deploy/center/README.md)）。

## 現在の状態と未完了ゲート

Arion は現在、macOS（`arm64`）で動作するメニューバー音声入力アプリです。Windows（`amd64`）向けコードはコンパイル可能で、ヘッドレステストおよび Windows Server 2022 上での CI ウィンドウライフサイクル、`SendInput`、ホールドキーフックの E2E テストを通過していますが、Windows ホスト上での完全な対話型ディクテーションのエンドツーエンド検証はまだ行われていません。

未実装または進行中の項目：

- **ローカル候補ランキング（`RNK-01`）** — ランキング契約と評価ハーネスは存在しますが、現在の認識器は単一の仮説のみを返すため、候補ランキングは有効化されていません。
- **デスクトップメモリ同期クライアントとデバイス登録（`MEM-01` / `SYNC-10`）** — `services/sync` リレーおよびデスクトップ OIDC サインインフローは実装済みですが、デスクトップ SQLite アウトボックス、デバイス登録、およびエンドツーエンドの鍵同期は未完了です。
- **Windows TSF とローカル書き換えワーカー（`TSF-01` / `RWT-RUN-01`）** — Windows Text Services Framework（TSF）IME 統合、Windows の編集学習、および Windows ローカル書き換えランタイムは未実装です。
- **対話型 OS 権限と IMK 受け入れ検証** — ホールドして話すジェスチャーロジックと Windows 低レベルフックは CI E2E を通過していますが、macOS イベントタップ（「入力監視」権限）およびクリーン環境での IMK 登録は対話型デスクトップ上での検証が必要です。
- **書き換えのブラインド人間評価（`LRW-REVIEW-01` / `COR-01`）** — 開発ベンチマークは [`benchmarks/results`](benchmarks/README.md) に記録されていますが、ブラインド人間評価（`LRW-A06`）は未完了です。また、書き換えパイプライン外のスタンドアロン LLM 補正は採用されていません。
- **署名付きパブリックリリース（`REL-01`）** — アプリ内アップデーターとリリースステージングパイプラインは実装済みですが、本番用 Developer ID コード署名、公証（notarization）、および本番バンドル識別子は保留中です。

正確な検証エビデンスとタスクゲートについては、[エージェントハンドオフ](docs/project/agent-handoff.md)および[検証レポート](docs/quality/validation.md)を参照してください。

## クイックスタート

### 前提条件

- `cgo` が有効化された **Go 1.27.1**（`go.work` は Go の自動ツールチェーン選択を使用します。`packages/core` と `services/sync` は `GOWORK=off` でも単体チェックされます）。
- **macOS（`arm64`）** — Xcode Command Line Tools および macOS SDK。
- **Windows（`amd64`）** — MyGo が必要とするネイティブツールチェーン、および `cgo` のクロスパッケージング用の MinGW-w64（`CC=x86_64-w64-mingw32-gcc`）とインストーラービルド用の NSIS。
- **Node 24**（`.nvmrc`）と**バージョン固定された `pnpm@10.33.0`**（`package.json`）— リポジトリチェック、評価ツール、および静的マーケティングサイト（`apps/site`）で使用します。`npm` と `yarn` は拒否されます。

### 主なコマンド

```sh
make install        # 固定バージョンの pnpm ワークスペースをインストール (--frozen-lockfile)
make check          # ドキュメント、JSON、Go レイヤー境界、core/sync および評価テスト
make desktop-check  # デスクトップテストと go vet (cgo)
make desktop-dev    # ネイティブランタイムをステージングし、開発モードでデスクトップアプリを起動
make build desktop  # apps/desktop/build/ 配下に Arion.app と DMG をパッケージング
make e2e            # 実際のデスクトップセッションでのネイティブウィンドウライフサイクル E2E
make e2e-macos      # パッケージングされた Arion.app に対する製品シナリオテスト
make site-check     # マーケティングサイト (apps/site)：i18n、型、lint、テスト、静的エクスポート
```

初回実行時、`make desktop-dev` はオンボーディングを開き、マイクとアクセシビリティの権限を要求し、デフォルトの認識モデルバンドルを `~/Library/Application Support/Arion/models/`（macOS）または `%AppData%\Arion\models\`（Windows）にダウンロードします。モデルのプロビジョニング完了後、ディクテーションはオフラインで動作します。すべてのベンチマーク、評価、診断ターゲットを一覧表示するには `make help` を実行してください。

## ドキュメントマップ

| 領域 | エントリポイント |
| --- | --- |
| **ガイドライン・索引** | [Agentic Coding Guidelines (`AGENTS.md`)](AGENTS.md) · [ドキュメントインデックス (`docs/README.md`)](docs/README.md) |
| **プロダクト・設計** | [プロダクトスコープ (`docs/project/README.md`)](docs/project/README.md) · [アーキテクチャ (`docs/project/architecture.md`)](docs/project/architecture.md) · [リポジトリオーナーシップ (`docs/project/repository.md`)](docs/project/repository.md) |
| **仕様書・意思決定** | [仕様書 (`specs/README.md`)](specs/README.md) · [アーキテクチャ決定記録 (`docs/decisions/README.md`)](docs/decisions/README.md) · [エージェントハンドオフ (`docs/project/agent-handoff.md`)](docs/project/agent-handoff.md) |
| **フロントエンド・バックエンド** | [フロントエンド FSD 規約 (`docs/frontend/README.md`)](docs/frontend/README.md) · [UI ローカリゼーション (`docs/frontend/i18n.md`)](docs/frontend/i18n.md) · [バックエンド DDD 規約 (`docs/backend/README.md`)](docs/backend/README.md) |
| **運用・デプロイ** | [ローカル開発 (`docs/operations/local-dev.md`)](docs/operations/local-dev.md) · [デプロイメントとリリースゲート (`docs/operations/deployment.md`)](docs/operations/deployment.md) · [同期リレー運用ランブック (`deploy/sync/README.md`)](deploy/sync/README.md) · [センターデプロイ (`deploy/center/README.md`)](deploy/center/README.md) |
| **品質・ベンチマーク** | [品質とテスト (`docs/quality/README.md`)](docs/quality/README.md) · [検証レポート (`docs/quality/validation.md`)](docs/quality/validation.md) · [ネイティブ E2E (`docs/quality/e2e.md`)](docs/quality/e2e.md) · [リサーチエビデンス (`docs/research/README.md`)](docs/research/README.md) · [ベンチマーク手法 (`docs/research/benchmark-methodology.md`)](docs/research/benchmark-methodology.md) · [評価ランナー (`tools/evaluation/README.md`)](tools/evaluation/README.md) · [ベンチマーク結果 (`benchmarks/README.md`)](benchmarks/README.md) |

## コントリビューター

[![AstraFoundry/Arion contributors](https://contrib.rocks/image?repo=AstraFoundry/Arion)](https://github.com/AstraFoundry/Arion/graphs/contributors)

## Star 履歴

[![AstraFoundry/Arion Star History Chart](https://api.star-history.com/svg?repos=AstraFoundry/Arion&type=Date)](https://star-history.com/#AstraFoundry/Arion&Date)

## ライセンス

Arion は [AstraFoundry](https://github.com/AstraFoundry) によって開発され、[AGPL-3.0](LICENSE)（[ADR 009](docs/decisions/009-agpl-license.md)）の下でライセンスされています。[X (@AstraFoundry)](https://x.com/AstraFoundry) のフォロー、[Discord](https://discord.com/invite/72kh465mMm) や [Feishu グループ](https://applink.feishu.cn/client/chat/chatter/add_by_link?link_token=640t6cde-d38e-4a81-875b-7799e44b77ae)への参加をお待ちしています。バグ報告は [GitHub Issues](https://github.com/AstraFoundry/Arion/issues) までお願いします。
