---
title: Arion 0.1.2〜0.1.4：変更点のまとめ
description: 最初の正式リリースから1週間で3つのアップデートを公開しました。静かな録音に単語帳が書き込まれなくなり、クラウド整文は OpenAI 互換・Anthropic 互換のエンドポイントに接続でき、問題が起きたときはコーディングエージェントに報告を手伝ってもらえるようになりました。
date: 2026-11-08
authors: stvlynn
category: announcement
tags: [リリース, 変更履歴, 音声入力, クラウド整文, フィードバック]
image: ./feedback.en.webp
imageAlt: Arion の Report a problem ウィンドウ。左側の模擬 Codex チャットでは「Dictation stops after a long pause」という GitHub Issue が作成され、右側には Arion 0.1.4 と macOS arm64 を記したプロンプトと Copy ボタンがある
draft: false
---

[Arion 0.1.1](/blog/arion-0-1-1/) の公開から1週間で、Apple シリコン搭載の Mac と Windows x64 向けに3つのアップデートを出しました。0.1.2 は認識の不具合を修正し、0.1.3 はクラウド整文を互換エンドポイントに開放して macOS での次の音声入力の開始を軽くし、0.1.4 は普段使っているコーディングエージェントに問題の報告を任せる入り口を追加しました。

0.1.1 をお使いの場合は、Settings › About を開き、**Check for Updates** をクリックしてください。

## 単語帳が下書きに出なくなりました（0.1.2）

単語帳の項目はホットワードも兼ねています。Arion は録音のたびにそれを音声モデルに渡し、人名や用語をあなたの表記で認識させます。0.1.1 では、モデルがこのリストをそのまま読み返すことがありました。話し始める前に間があいたり、文末で声が小さくなったり、何も話さずにキーを押し続けたりすると、下書きが空のままにならず「Arion MLX Qwen」のような自分の登録語で埋まることがありました。

0.1.2 では2か所で修正しています。

- **静かな音声はモデルに送りません。** Arion は各音声ウィンドウを 20 ms ごとに、区間の終わりを判定するのと同じ −42 dBFS のしきい値で調べます。しきい値に届く部分が1つもないウィンドウは認識リクエストを作らずに空のテキストを返すので、モデルが何かを埋める余地がありません。
- **リストを写した結果は認識し直します。** 認識結果の先頭か末尾に、ホットワードが2つ以上リストの順番どおりに続いて現れた場合、Arion は同じ音声をホットワードなしでもう一度認識し、その結果を使います。実際に話した1語のホットワードは残り、順番が違う語や文の途中に出てくる語もそのままです。キャンセルすればリクエストはすぐに止まります。[^hotwords]

<embed-frame src="/blog/arion-0-1-4/stage.ja.html#hotwords" poster="./posters/hotwords.ja.webp" title="デモ：右 Option を押して少し間をあけてから話す" ratio="16/10" caption="話し始める前に間をあけます。音声が静かなあいだピルは空のままで、話した内容だけが表示されます。単語帳は引き続き Arion と MLX を正しく表記します。模擬録画で、音声は録音していません。"></embed-frame>

## 互換エンドポイントでのクラウド整文（0.1.3）

プロバイダーの一覧に **OpenAI compatible** と **Anthropic compatible** が加わりました。Base URL と API キーを入力すると、Arion がそのエンドポイントのモデル一覧を読み込み、そこから選べます。自前のゲートウェイや地域ミラーなど、どちらかの API に対応したサービスをつなげます。要望を[出してくれた](https://github.com/AstraFoundry/Arion/issues/99) [@harry-xm](https://github.com/harry-xm) さんに感謝します。

<embed-frame src="/blog/arion-0-1-4/stage.ja.html#compatible" poster="./posters/compatible.ja.webp" title="デモ：OpenAI compatible のエンドポイントに接続する" ratio="16/10" caption="OpenAI compatible を選び、Base URL を入力してキーを貼り付けます。Connect のあと、Model メニューにはそのエンドポイント自身のモデル一覧が並びます。模擬録画で、どのプロバイダーにも接続していません。"></embed-frame>

Base URL はキーと一緒に OS の資格情報ストアに保存され、プロバイダーごとに1項目なので、切り替えてもそれぞれのアドレスとキーは残ります。公式の OpenAI と Anthropic の項目は変わりません。プライバシーの境界はほかのクラウドプロバイダーと同じで、Arion が送るのは認識したテキスト、シーン、一致した単語帳の項目だけです。音声、カーソルやウィンドウのコンテキスト、履歴はデバイスに残ります。[^cloud]

エンドポイントはモデル一覧を公開している必要があります。Arion はモデル一覧の取得でキーを確認するため、一覧を公開しないプロキシには引き続き接続できません。

## DeepSeek のキーが再び接続できるように（0.1.3）

有効な DeepSeek のキーでも「The provider refused this key or sign-in.」と表示されて接続できないことがありました。[@huwan](https://github.com/huwan) さんが[原因を突き止めて](https://github.com/AstraFoundry/Arion/issues/101)くれました。モデル一覧のリクエストにボディが付いていたため DeepSeek の CDN が HTTP 403 だけを返し、Arion はそれをキーの拒否と判断していたのです。

モデル一覧はボディのない GET で取得するようになりました。キーの拒否とみなすのは、明示的な認証失敗か HTTP 401 だけです。それ以外の 403、レート制限、サーバーエラー、読み取れない応答は「Couldn't load models.」と表示します。確認に失敗しても保存済みのキーは置き換わらず、更新に失敗しても手元のモデル一覧は残ります。ローカルのログに残るのは失敗コードと HTTP ステータスだけで、キー、URL、応答の中身は記録しません。

<embed-frame src="/blog/arion-0-1-4/stage.ja.html#deepseek" poster="./posters/deepseek.ja.webp" title="デモ：DeepSeek のキーで接続する" ratio="16/10" caption="DeepSeek のキーを貼り付けて Connect を押します。モデル一覧はボディのない GET で取得され、キーが受け入れられてモデルを選べます。模擬録画です。"></embed-frame>

## macOS でマイクの開始が軽く（0.1.3）

これまでは音声入力のたびに macOS の録音デバイスを新しく作り、終わると破棄していました。いまは停止したデバイスを1つ保持して次の録音で再利用し、ウォームスタートではオーディオコンテキストとデバイスの初期化を省きます。

録音と録音のあいだに音を拾い続けることはありません。再利用の前に、バッファーの音声とリサンプラーの状態を消去し、前のセッションを切り離し、マイクの許可と選択中（または既定）の入力デバイスを確認し直します。スリープ、画面ロック、許可の取り消し、終了でデバイスは解放され、入力デバイスの変更やデバイスのエラーがあれば古いデバイスは使いません。Windows ではこれまでどおり録音ごとにデバイスを作ります。[^capture]

## エージェントと一緒に問題を報告（0.1.4）

Codex や Claude Code のようなコーディングエージェントを日常的に使っている方は多いと思います。0.1.4 では、Arion で何かあったときにそのエージェントの手を借りられます。メニューバーのパネルにある **?** ボタン、Settings › About の **Ask an agent**、macOS や Windows の入力メニューにある **Report a problem…** から **Report a problem** ウィンドウを開けます。

<embed-frame src="/blog/arion-0-1-4/stage.ja.html#feedback" poster="./posters/feedback.ja.webp" title="デモ：コーディングエージェントに問題の報告を任せる" ratio="16/10" caption="メニューバーパネルの ? ボタンから Report a problem を開き、プロンプトをコピーしてエージェントに貼り付けると、質問のあと Issue が作成されます。模擬録画で、Issue は作成していません。"></embed-frame>

ウィンドウにはコピーできる1行のプロンプトがあります。Arion のウェブサイトにあるフィードバック用スキルの場所、バージョン、プラットフォームだけを記し、パス、アカウント情報、ログは含みません。これをエージェントに貼り付けると、エージェントは次のように進めます。

- あなたの言語で、1回に1つか2つずつ質問する
- 使い方の質問にはリポジトリをもとに答え、Issue は作らない
- 新しいリリースですでに直っていないか確かめ、既存の Issue を検索する
- ローカルのログから関係する部分だけを読み、個人的な情報を取り除き、添付する前に抜粋を見せる
- 内容をあなたが承認してから Issue を作成し、修正のプルリクエストも提案する

Arion 自体は何も送信せず、エージェントはあなたが承認した情報だけを使います。自分で書きたい場合は、**Open an issue yourself** から直接フォームを開けます。[^feedback]

## まだできていないこと

- マイクの再利用は実機でまだ測定しておらず、何ミリ秒短くなったかは言えません。コールドスタートやデバイス切り替え時の遅延も、ネイティブ環境での測定が必要です。
- モデル一覧を公開しない互換エンドポイントには接続できません。クラウド整文は実サービスでの受け入れ確認も、品質と遅延のベンチマークもまだです。
- [0.1.1 で挙げた未完了の項目](/blog/arion-0-1-1/#まだできていないこと)は引き続き残っています。複数デバイスの同期はまだなく、修正の学習は macOS のネイティブエディターだけで、Intel Mac には対応していません。

各バージョンの詳しい内容は[デスクトップ版の変更履歴](https://github.com/AstraFoundry/Arion/blob/main/apps/desktop/CHANGELOG.md)にあります。問題が起きたら新しい **Report a problem** ウィンドウを試すか、[GitHub Issues](https://github.com/AstraFoundry/Arion/issues) に報告するか、[Discord](https://discord.com/invite/72kh465mMm) や [Feishu](https://applink.feishu.cn/client/chat/chatter/add_by_link?link_token=640t6cde-d38e-4a81-875b-7799e44b77ae) で声をかけてください。

<github-people label="コントリビューター" logins="stvlynn, zacharyhaha"></github-people>

[^hotwords]: [仕様 002：音声パイプライン](https://github.com/AstraFoundry/Arion/blob/main/specs/002-speech-pipeline.md)（`SPH-A12`、`SPH-A13`）と [ADR 020：ネイティブ Qwen3-ASR](https://github.com/AstraFoundry/Arion/blob/main/docs/decisions/020-native-qwen-asr.md)。

[^cloud]: [クラウド整文](https://github.com/AstraFoundry/Arion/blob/main/docs/project/cloud-polishing.md)と [ADR 022](https://github.com/AstraFoundry/Arion/blob/main/docs/decisions/022-cloud-polishing.md)。

[^capture]: [仕様 001：デスクトップ入力](https://github.com/AstraFoundry/Arion/blob/main/specs/001-desktop-input.md)（`DSK-04`）。

[^feedback]: [プルリクエスト #108：エージェントによる問題報告](https://github.com/AstraFoundry/Arion/pull/108)。エージェントがたどる手順はリポジトリの `docs/project` にあるエージェントフィードバックの文書にあります。
