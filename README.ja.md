# Arion

[MyGo](https://github.com/egoist/mygo) ネイティブ Go UI で構築された、macOS および Windows 向けのローカルファースト音声入力ツール。

[English](README.md) · [简体中文](README.zh-CN.md) · [日本語](README.ja.md)

[![License: AGPL-3.0](https://img.shields.io/badge/License-AGPL--3.0-blue.svg)](LICENSE)
[![Go 1.27.1](https://img.shields.io/badge/Go-1.27.1-00ADD8?logo=go&logoColor=white)](go.work)
[![MyGo v0.4.6](https://img.shields.io/badge/MyGo-v0.4.6-4B5563)](apps/desktop/go.mod)
[![sherpa-onnx 1.13.8](https://img.shields.io/badge/sherpa--onnx-1.13.8-0F766E)](apps/desktop/go.mod)
[![Node 24](https://img.shields.io/badge/Node-24-339933?logo=nodedotjs&logoColor=white)](.nvmrc)
[![pnpm 10.33.0](https://img.shields.io/badge/pnpm-10.33.0-F69220?logo=pnpm&logoColor=white)](package.json)

![Arion の音声入力ウィンドウとフローティングディクテーションピル](docs/readme/hero.png)

![Arion の音声ディクテーション動作](docs/readme/demo.gif)

## 概要

Arion は、オンデバイスのストリーミング音声認識、保守的なルールベース補正、個人語彙学習、オプションのリアルタイムローカル書き換え、ローカル候補ランキング契約、および暗号化メモリ同期を組み合わせています。ユーザーはグローバルショートカットまたはシステム IME 統合を介して音声入力を行えます。モデルのプロビジョニング完了後、ディクテーションはオフラインで動作し、生のマイク音声や一時的な外部ウィンドウのコンテキストがデバイス外へ送信されることはありません。

macOS と Windows が主要なデスクトップターゲット（`P0`）であり、iOS と Android は将来のターゲット（`P2`）です。

## 主な機能

- **グローバルショートカット音声入力と macOS IME ホスト** — どのアプリからでもショートカット（デフォルトは `⌥Space`）を押し、中国語（普通話）、英語、またはそれらが混在する技術テキストを話してもう一度押すと、フォーカスされているエディタに一度で挿入されます（`Esc` でキャンセル）。ホールドして話すモード（[DSK-02](specs/001-desktop-input.md)）および `ArionInput.app` macOS InputMethodKit ホスト（[ADR 006](docs/decisions/006-macos-input-method-host.md)）も含まれています。
- **オンデバイスモデルの自動選択** — 音声認識は `sherpa-onnx`（`v1.13.8`）を通じてローカルで実行されます。レイテンシ基準を満たし、最も高い実測精度を持つ評価済みモデルバンドルが自動的にダウンロード、ダイジェスト検証、読み込みされるため、手動でのモデル選択は不要です（[ADR 004](docs/decisions/004-automatic-model-selection.md)）。
- **保守的な補正と編集学習** — 認識ドラフトと確定テキストを分離しつつ、決定論的ルール（`arion-rules-v1`）とユーザーが承認した語彙を適用します。macOS では、挿入されたテキストに対する編集がオンデバイスで学習され、ワンクリックで元に戻すことができます（[ADR 014](docs/decisions/014-edit-learning.md)）。
- **オプションのリアルタイムローカル書き換え（macOS Apple silicon）** — 分離された `arion-llm` ONNX Runtime GenAI ワーカープロセス内で Qwen3 1.7B を使用し、発話中にテキストをリアルタイム編集します。KV プレフィックス再利用、シーンに応じたフォーマット（**アプリに合わせる**、[ADR 012](docs/decisions/012-context-scene-selection.md)）、およびオンデバイスでのカーソル・画面・履歴からの用語抽出に対応しています（[ADR 011](docs/decisions/011-live-local-rewriting.md)、[ADR 013](docs/decisions/013-rewrite-context-on-device.md)）。
- **ネイティブ Go デスクトップシェル** — Electron や webview を使用せず、Go での Feature-Sliced Design（[ADR 001](docs/decisions/001-desktop-native-boundaries.md)、[ADR 007](docs/decisions/007-sunbeam-design-system.md)）を採用して [MyGo](https://github.com/egoist/mygo)（`v0.4.6`）で構築されています。メニューバーポップオーバー、フルスクリーン Space に対応するクリックスルー可能なフローティングピル、初回起動オンボーディング、設定画面、ローカル診断 ZIP エクスポート、およびリアルタイム切り替え可能な英語 / 簡体字中国語 UI ローカリゼーション（[i18n](docs/frontend/i18n.md)）を備えています。
- **ローカル暗号化とセルフホスト可能な同期リレー** — 個人語彙は macOS ログインキーチェーンまたは Windows DPAPI に保存された鍵を使用し、ローカルで AES-256-GCM により暗号化されます（[ADR 005](docs/decisions/005-local-encryption.md)）。また、`noop` および `oidc` アカウントゲートを備えたセルフホスト可能な Go 暗号文リレー `services/sync`（[ADR 015](docs/decisions/015-hosted-relay-account-authorization.md)）と、デスクトップのループバック PKCE サインインサービス（[アカウントサインイン](docs/project/account-signin.md)）も提供しています。

## 現在の状態と制限事項

Arion は現在、macOS（`arm64`）で動作するメニューバー音声入力アプリです。Windows（`amd64`）向けコードはコンパイル可能で、ヘッドレステストおよび Windows Server 2022 上での CI ウィンドウライフサイクル、`SendInput`、ホールドキーフックの E2E テストを通過していますが、Windows ホスト上での完全な対話型ディクテーションのエンドツーエンド実行はまだ行われていません。

未実装または進行中の項目：

- **ローカル候補ランキング（`RNK-01`）** — ランキング契約と評価ハーネスは存在しますが、現在の認識器は単一の仮説のみを返すため、候補ランキングは有効化されていません。
- **デスクトップメモリ同期クライアントとデバイス登録（`MEM-01` / `SYNC-10`）** — `services/sync` リレーおよびデスクトップ OIDC サインイン画面・サービスは実装済みですが、デスクトップ SQLite アウトボックス、デバイス登録、およびネイティブ OS 資格情報ストアの実行は未完了です。
- **Windows TSF とライブ書き換えワーカー（`TSF-01` / `RWT-RUN-01`）** — Windows Text Services Framework（TSF）IME 統合、Windows の編集学習、および Windows GenAI 書き換えワーカーは未実装です。
- **ホールドして話す機能と同意ガイドの対話型検証** — ホールドして話すジェスチャーロジックと Windows 低レベルフックは CI E2E を通過していますが、macOS イベントタップは対話型デスクトップ上でユーザーが付与する「入力監視」権限を必要とします。
- **書き換えのブラインド人間評価と汎用 LLM 補正（`LRW-REVIEW-01` / `COR-01`）** — 開発ベンチマークは [`benchmarks/results`](benchmarks/README.md) に記録されていますが、ブラインド人間評価（`LRW-A06`）は未完了です。また、ライブ書き換えパイプライン外のスタンドアロン LLM 補正は採用されていません。
- **署名付きパブリックリリース（`REL-01`）** — アプリ内アップデーターは実装済み（[ADR 010](docs/decisions/010-in-app-updates.md)）ですが、Developer ID コード署名、公証（notarization）、および本番用バンドル識別子は保留中です。

正確な検証エビデンスとタスクゲートについては、[エージェントハンドオフ](docs/project/agent-handoff.md)および[検証レポート](docs/quality/validation.md)を参照してください。

## クイックスタート

### 前提条件

- `cgo` が有効化された **Go 1.27.1**（`go.work` は Go の自動ツールチェーン選択を使用します。`packages/core` と `services/sync` は `GOWORK=off` でも単体チェックされます）。
- **macOS（`arm64`）**：Xcode Command Line Tools および macOS SDK。
- **Windows（`amd64`）**：MyGo が必要とするネイティブツールチェーン、および `cgo` のクロスパッケージング用の MinGW-w64（`CC=x86_64-w64-mingw32-gcc`）とインストーラービルド用の NSIS。
- リポジトリチェック、評価ツール、および静的マーケティングサイト（`apps/site`）用の **Node 24**（`.nvmrc`）と**バージョン固定された `pnpm@10.33.0`**（`package.json`）。`npm` と `yarn` は拒否されます。

### ビルドと実行

```sh
make install        # 固定バージョンの pnpm ワークスペースをインストール (--frozen-lockfile)
make check          # ドキュメント、JSON、Go レイヤー境界、core/sync および評価テスト
make desktop-check  # デスクトップテストと go vet (cgo)
make desktop-dev    # ネイティブランタイムをステージングし、開発モードでデスクトップアプリを起動
make desktop-build  # apps/desktop/build/ 配下に Arion.app と DMG をパッケージング
make e2e            # 実際のデスクトップセッションでのネイティブウィンドウライフサイクル E2E
make e2e-macos      # パッケージングされた Arion.app に対する製品シナリオテスト
make site-check     # マーケティングサイト (apps/site)：i18n、型、lint、テスト、静的エクスポート
```

初回実行時、`make desktop-dev` はオンボーディングを開き、マイクとアクセシビリティの権限を要求し、自動選択された認識モデルバンドル（約 270 MB）を `~/Library/Application Support/Arion/models/`（macOS）または `%AppData%\Arion\models\`（Windows）にダウンロードします。モデルのプロビジョニング完了後、ディクテーションはオフラインで動作します。すべてのベンチマーク、評価、診断ターゲットを一覧表示するには `make help` を実行してください。

## ドキュメント

### プロジェクトと仕様

- [Agentic Coding Guidelines (`AGENTS.md`)](AGENTS.md) — コントリビューターおよびコーディングエージェント向けのルール、境界、ワークフロー。
- [ドキュメントインデックス (`docs/README.md`)](docs/README.md) — すべてのドキュメント領域のトップレベルマップ。
- [プロダクトスコープ (`docs/project/README.md`)](docs/project/README.md) — 確定済みスコープ、明示的な境界、およびスタックステータス表。
- [アーキテクチャ (`docs/project/architecture.md`)](docs/project/architecture.md) — データフロー、ライブローカル書き換えパイプライン、ランタイム構成、プライバシー境界。
- [リポジトリオーナーシップ (`docs/project/repository.md`)](docs/project/repository.md) — Go ワークスペースモジュール、FSD/DDD パッケージ境界、作業ルール。
- [仕様書 (`specs/README.md`)](specs/README.md) — 規範的要件（`001`〜`009`）および受け入れシナリオ ID。
- [エージェントハンドオフ (`docs/project/agent-handoff.md`)](docs/project/agent-handoff.md) — 実装済みマイルストーン、未完了の `P0`/`P1`/`P2` タスク、完了ゲート。
- [アーキテクチャ決定記録 (`docs/decisions/README.md`)](docs/decisions/README.md) — ADR `001`〜`016`。

### エンジニアリング、運用、評価

- [フロントエンド規約 (`docs/frontend/README.md`)](docs/frontend/README.md) — ネイティブ Go における Feature-Sliced Design（FSD）と[インターフェースローカリゼーション (`docs/frontend/i18n.md`)](docs/frontend/i18n.md)。
- [バックエンド規約 (`docs/backend/README.md`)](docs/backend/README.md) — `packages/core` および `services/sync` 向けのドメイン駆動設計（DDD）ルール。
- [ローカル開発 (`docs/operations/local-dev.md`)](docs/operations/local-dev.md) — ツールチェーン、データ保存場所、環境変数、トラブルシューティング。
- [デプロイメントとリリースゲート (`docs/operations/deployment.md`)](docs/operations/deployment.md) および[同期リレー運用ランブック (`deploy/sync/README.md`)](deploy/sync/README.md)。
- [品質とテスト (`docs/quality/README.md`)](docs/quality/README.md)、[検証レポート (`docs/quality/validation.md`)](docs/quality/validation.md)、および[ネイティブ E2E (`docs/quality/e2e.md`)](docs/quality/e2e.md)。
- [リサーチエビデンス (`docs/research/README.md`)](docs/research/README.md)、[ベンチマーク手法 (`docs/research/benchmark-methodology.md`)](docs/research/benchmark-methodology.md)、[評価ランナー (`tools/evaluation/README.md`)](tools/evaluation/README.md)、および[ベンチマーク結果 (`benchmarks/README.md`)](benchmarks/README.md)。

## コントリビューター

[![AstraFoundry/Arion contributors](https://contrib.rocks/image?repo=AstraFoundry/Arion)](https://github.com/AstraFoundry/Arion/graphs/contributors)

## Star 履歴

[![AstraFoundry/Arion Star History Chart](https://api.star-history.com/svg?repos=AstraFoundry/Arion&type=Date)](https://star-history.com/#AstraFoundry/Arion&Date)

## ライセンス

Arion は [AstraFoundry](https://github.com/AstraFoundry) によって開発され、[AGPL-3.0](LICENSE)（[ADR 009](docs/decisions/009-agpl-license.md)）の下でライセンスされています。バグ報告は [GitHub Issues](https://github.com/AstraFoundry/Arion/issues) までお願いします。
