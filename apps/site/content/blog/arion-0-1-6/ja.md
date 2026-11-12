---
title: Arion 0.1.5〜0.1.6：変更点のまとめ
description: 0.1.5 では2つ目の音声入力キー、整文が使えなかった理由の表示、コピーしたファイルの記録のオプトイン化。0.1.6 では選択したテキストを Claude や ChatGPT に渡すバー、クリップボード履歴の画像サムネイル、Windows と WeChat での貼り付けの安定化が加わりました。
date: 2026-11-12
authors: stvlynn
category: announcement
tags: [リリース, 変更履歴, 音声入力, 選択アクション, クリップボード]
image: ./cover.webp
imageAlt: Arion の暗い色の選択バー。Claude、ChatGPT、Copy、Add to vocabulary と X が並ぶ。その下の Claude のメニューには Translate to Chinese、Explain、Summarize key points、Rewrite、Ask a custom question… がある
draft: true
---

[Arion 0.1.4](/blog/arion-0-1-4/) のあと、Apple シリコン搭載の Mac と Windows x64 向けに 0.1.5 と 0.1.6 の2つのアップデートを出しました。0.1.5 は2つ目の音声入力キーを追加し、マイクを早く開くようにし、整文が使えなかった理由を伝え、候補の文字色を固定できるようにし、コピーしたファイルの記録をオプトインにしました。0.1.6 は選択アクション、初回の Overview、クリップボード履歴の画像サムネイルを追加し、反応の遅い Windows アプリと、macOS で WeChat からコピーしたあとの貼り付けを修正しました。

以前のバージョンをお使いの場合は、Settings › About を開き、**Check for Updates** をクリックしてください。

## 2つ目の音声入力キー（0.1.5）

Settings › Dictation の **Dictation key** の横に **+** が加わりました。クリックして使いたいキーを押すと、どちらのキーでも音声入力を始められます。2つ目のキーも長押しとタップの動きは1つ目と同じで、通常の音声入力を始めます。**−** で削除できます。

<embed-frame src="/blog/arion-0-1-6/stage.ja.html#shortcut" poster="./posters/shortcut.ja.webp" title="デモ：2つ目の音声入力キーを追加する" ratio="16/10" caption="Dictation key の横の + をクリックし、Right Command を押します。これでRight Option とRight Command のどちらでも音声入力を始められます。模擬録画です。"></embed-frame>

追加できるキーは1つだけです。ほかのショートカットで使っているキーは受け付けません。[^shortcut]

## マイクを先に開く（0.1.5）

キーを押してすぐ話し始めると、最初の言葉が準備中に埋もれることがありました。いまの Arion はまずマイクを開き、入力欄を探して認識器を準備しているあいだの発話をためておき、準備ができたら順番に認識器へ渡します。[@earmer](https://github.com/earmer) さん、[この変更](https://github.com/AstraFoundry/Arion/pull/117)をありがとうございます。

<embed-frame src="/blog/arion-0-1-6/stage.ja.html#capture" poster="./posters/capture.ja.webp" title="デモ：キーを押してすぐ話す" ratio="16/10" caption="Right Option を押してすぐ話します。下書きは最初の言葉から始まり、長い下書きの末尾はピルの右端に収まったままです。模擬録画です。音声は録音していません。"></embed-frame>

モデルがない場合は、これまでどおりマイクを開く前にセッションが終わります。準備に失敗したときやキャンセルしたときは、ためた音声を認識せずに破棄します。パスワード欄が対象のときも同じです。[^capture]

同じ変更で、ピルの長い下書きも直りました。日本語や中国語とラテン文字が混ざった長い下書きでは、末尾が行の端から左へずれていきました。Arion が1文字ずつの幅と、行全体を組んだときの幅を混ぜて計算していたためです。いまは下書きの末尾が端に収まります。

## 整文が使えなかった理由を伝える（0.1.5）

クラウド整文が使えず、Arion が整文前のテキストを挿入したときは、音声入力が終わったあとにピルが短いメモを表示します。何が起きたか、次に何をすればよいかです。メモは、Arion の起動中、プロバイダー、モデル、問題の種類ごとに1回だけ出ます。

<embed-frame src="/blog/arion-0-1-6/stage.ja.html#polish" poster="./posters/polish.ja.webp" title="デモ：整文のメモと Allow reasoning when required" ratio="16/10" caption="モデルが推論なしでの整文を拒否します。テキストは整文前のまま貼り付けられ、ピルに Model needs reasoning と表示され、クラウドモデルの設定で Allow reasoning when required をオンにします。模擬録画です。プロバイダーには接続していません。"></embed-frame>

推論をオフにしたリクエストを拒否するモデルがあります。新しい Gemini モデルや、OpenRouter の gpt-5-mini などです。クラウドモデルのページに、既定でオフの **Allow reasoning when required** スイッチが加わりました。オンにすると、モデルに拒否されたときに Arion は minimal、次に low の推論で再試行し、そのモデルが受け入れたレベルを覚えます。これによって遅くなるのは、そうしたモデルの整文だけです。[@earmer](https://github.com/earmer) さん、[こちらの変更](https://github.com/AstraFoundry/Arion/pull/107)もありがとうございます。[^polish]

## 候補の文字色（0.1.5）

キーボードの候補バーに、Settings › Keyboard の **Text color** が加わりました。**Automatic** は、**Highlight color** に合わせて選択中の候補の文字を白か黒にします。塗りに対して 3:1 のコントラストを保てるときは白、そうでなければ黒です。メニューには **Automatic (White)** のように、どちらを選んだかが表示されます。

<embed-frame src="/blog/arion-0-1-6/stage.ja.html#textcolor" poster="./posters/textcolor.ja.webp" title="デモ：ハイライト色を選び、文字色を固定する" ratio="16/10" caption="ハイライト色に Purple を選ぶと、自動の文字色が白になります。Black を選ぶと固定され、選択中の候補もそれに従います。模擬録画です。"></embed-frame>

**Black** と **White** は、ライトとダークのどちらの外観でも色を固定します。**Automatic** やほかのハイライト色を選ぶと固定は解除されます。[@cyrus-cai](https://github.com/cyrus-cai) さん、[追加してくれて](https://github.com/AstraFoundry/Arion/pull/121)ありがとうございます。[^textcolor]

## コピーしたファイルの記録はオプトインに（0.1.5）

クリップボード履歴がコピーしたファイルを記録するのは、Settings › Clipboard で **Record copied files** をオンにしたときだけになりました。このスイッチは既定でオフで、以前のバージョンのプロファイルでも同じです。オンにするときは **Enable file history** で一度だけ確認します。

<embed-frame src="/blog/arion-0-1-6/stage.ja.html#files" poster="./posters/files.ja.webp" title="デモ：Record copied files をオンにする" ratio="16/10" caption="Record copied files をオンにして一度だけ確認します。履歴にはコピーしたファイルへの参照が残り、ファイルの中身はアップロードされません。模擬録画です。"></embed-frame>

スイッチがオフのあいだ、すでにあるファイルの記録は非表示になるだけで、削除はされません。履歴が持つのはファイルのコピーではなく参照で、macOS はファイルのあるフォルダーへのアクセスを求めることがあります。[@gavi102803](https://github.com/gavi102803) さん、[この変更](https://github.com/AstraFoundry/Arion/commit/9fba8413)をありがとうございます。[^files]

## Windows の Ctrl+V をすっきり送る（0.1.5）

[@gialoc668](https://github.com/gialoc668) さんから、Windows 11 の Feishu と WeChat で音声入力したテキストが入力欄に出ないと[報告](https://github.com/AstraFoundry/Arion/issues/110)がありました。原因の1つは、貼り付けのときに押されたままのキーでした。Windows は送信された Ctrl+V のために Shift、Alt、Windows キーを離さないため、余分な V が入力されたり、何も貼り付けられなかったりしました。いまの Arion はそれらのキーを先に離してから、左 Ctrl+V を1回送ります。送信した貼り付けについて、ピルは **Pasted** と表示します。アプリが受け取ったかどうかは Arion には確かめられないためです。[^paste]

## 選択アクション（0.1.6）

どのアプリでもテキストを選択すると、その横に小さなバーが出ます。有効にしたアシスタント、**Copy**、**Add to vocabulary**、X が並びます。Claude か ChatGPT にポインターを置くとメニューが開き、**Translate to Chinese**、**Explain**、**Summarize key points**、**Rewrite**、**Ask a custom question…** が並びます。ChatGPT の下には **Ask ChatGPT** もあります。

<embed-frame src="/blog/arion-0-1-6/stage.ja.html#selection" poster="./posters/selection.ja.webp" title="デモ：選択したテキストの説明を Claude に頼む" ratio="16/10" caption="文を選択し、Claude にポインターを置いて Explain を選びます。Claude アプリが開き、プロンプトと選択したテキストが入力済みで、送信を待っています。模擬録画です。Claude には何も送っていません。"></embed-frame>

プロンプトを選ぶと、Arion は `claude://` で Claude を、`codex://` で ChatGPT を開き、ローカルのチャットにプロンプトと選択したテキストを入れておきます。送信するのはあなたで、Arion 自身は何も送りません。プロンプトは組み込みの英語の指示で、Claude に渡す内容は最大 14,000 文字です。選択アクションは既定でオフです。セットアップに加わった最後のステップ **Act on selected text** か、Settings › General でオンにでき、そこで Claude、ChatGPT、またはその両方を選べます。

Arion は選択されたテキストをクリップボードを使わずに読み取ります。macOS ではアクセシビリティ、Windows では UI Automation を使い、Command-C を送ることはありません。パスワード欄ではバーは出ず、選択したテキストはログに書き込まれません。[^selection]

X を押すと、バーを止めるメニューが開きます。**Disable for 1 hour**、いまのサイトで無効にする、いまのアプリで無効にする、**Turn off selection actions** です。

<embed-frame src="/blog/arion-0-1-6/stage.ja.html#disable" poster="./posters/disable.ja.webp" title="デモ：1つのサイトでバーを無効にする" ratio="16/10" caption="X で無効化のメニューを開きます。Disable on docs.example.com を選ぶと、そのサイトでテキストを選択してもバーは出ません。模擬録画です。"></embed-frame>

## 初回の Overview（0.1.6）

最初の音声入力までは、**Overview** ページに合計、ヒートマップ、シーンの代わりに、マイクの絵と長押しするキーを載せたカードが1枚出て、その下に単語帳のカードが続きます。

![最初の音声入力前の Arion の Overview ページ。「No dictations yet. Say your first sentence.」の下にマイクのイラスト、「Speaking time, words and polishing scenes add up here as you dictate.」の一文、Option キーの横に「Hold in any app and speak」。その下の Vocabulary カードには「No terms yet. Add names and terms Arion should spell your way.」とある](./overview.en.webp "インストール直後の Overview ページ。デスクトップ版 UI のレンダリングです。")

0.1.6 では、設定、メニューバーのパネル、セットアップの見た目もよりフラットになりました。カードは細い境界線だけで影がなく、浮いて見えるのはメニューとポップアップだけです。[^overview]

## クリップボード履歴の画像サムネイル（0.1.6）

PNG、JPEG、GIF、WebP、BMP のファイルをコピーすると、クリップボード履歴でファイル名の横にサムネイルが出て、ポインターを置くと大きな画像が表示されます。[@gavi102803](https://github.com/gavi102803) さん、[追加してくれて](https://github.com/AstraFoundry/Arion/commit/73b4f0b2)ありがとうございます。

<embed-frame src="/blog/arion-0-1-6/stage.ja.html#thumbnails" poster="./posters/thumbnails.ja.webp" title="デモ：クリップボード履歴の画像ファイル" ratio="16/10" caption="Command-Shift-V でクリップボード履歴を開きます。コピーした beach.jpg にサムネイルが付き、ポインターを置くと大きく表示されます。模擬録画です。"></embed-frame>

項目はこれまでどおりファイルへの参照で、選ぶとファイルが貼り付けられます。読み込めない画像にはサムネイルが出ず、エラーも出ません。[^thumbnails]

## 反応の遅い Windows アプリを待って貼り付ける（0.1.6）

同じ [Windows の報告](https://github.com/AstraFoundry/Arion/issues/110)には、もう1つ原因がありました。Arion は貼り付けの 400 ms 後に元のクリップボードを戻していたため、WeChat のように遅れてクリップボードを読むアプリは古い内容を読んでいました。入力欄は空なのに、ピルには **Pasted** と出ていたのです。

いまの Arion は貼り付けの前に入力欄の状態を測り、300 ms 後にもう一度確かめて、600 ms でクリップボードを戻します。入力欄が変わっていなければ、500 ms 後にもう一度確かめて 1100 ms で戻します。どの確認でもテキストが入ったことを確かめられなかった場合は、音声入力したテキストをクリップボードに残し、ピルに **Copied** と表示します。Ctrl+V を押せば貼り付けられます。この Issue はまだ開いたままです。[^hold]

## macOS で WeChat からコピーしたあとの貼り付け（0.1.6）

WeChat の入力欄からテキストをコピーしたあと、次の音声入力が「Could not paste」「The clipboard changed or was unavailable.」で失敗することがありました。[@huwan](https://github.com/huwan) さんが[原因を突き止めてくれました](https://github.com/AstraFoundry/Arion/issues/122)。WeChat は macOS が名前を読めない2種類のリッチテキストのクリップボード形式を追加し、Arion はどれか1つの形式からデータが取れないとクリップボードのスナップショットをあきらめていました。いまはそれらの形式を飛ばし、貼り付けは最後まで進みます。[^wechat]

## そのほかの修正（0.1.6）

- 保存済みのアップデートがインストール済みのアプリと合わなくなったとき（たとえばディスクイメージから入れ直す前に準備されていたもの）でも、音声入力キーとメニューバーのボタンが効かなくなることはなくなりました。Arion はそれを忘れ、音声入力を使える状態に保ちます。[^update]
- すべて正常なとき、メニューバーのパネルに「Ready」と表示しなくなりました。状態を表示するのは、Arion の準備ができていないときだけです。

## まだできていないこと

- Windows では、選択したテキストがどのアプリやサイトのものかをまだ判別できないため、無効化のメニューにサイトやアプリの項目はありません。
- Windows の貼り付けの変更はプラットフォームに依存しないテストを通っていますが、Windows 11 の WeChat、QQ、Feishu ではまだ手で確かめていないため、[Issue #110](https://github.com/AstraFoundry/Arion/issues/110) は開いたままです。
- [0.1.1 で挙げた未完了の項目](/blog/arion-0-1-1/#まだできていないこと)は引き続き残っています。複数デバイスの同期はまだなく、修正の学習は macOS のネイティブエディターだけで、Intel Mac には対応していません。

各バージョンの詳しい内容は[デスクトップ版の変更履歴](https://github.com/AstraFoundry/Arion/blob/main/apps/desktop/CHANGELOG.md)にあります。問題が起きたら **Report a problem** を使うか、[GitHub Issues](https://github.com/AstraFoundry/Arion/issues) に報告するか、[Discord](https://discord.com/invite/72kh465mMm) や [Feishu](https://applink.feishu.cn/client/chat/chatter/add_by_link?link_token=e05vef49-cce2-404f-9740-2fb99d17d444) で声をかけてください。

<github-people label="コントリビューター" logins="stvlynn, earmer, cyrus-cai, gavi102803"></github-people>

<github-people label="Issue を報告してくれた方" logins="gialoc668, huwan"></github-people>

[^shortcut]: [音声入力のショートカット](https://github.com/AstraFoundry/Arion/blob/main/docs/project/dictation-shortcuts.md)と[仕様 001：デスクトップ入力](https://github.com/AstraFoundry/Arion/blob/main/specs/001-desktop-input.md)。

[^capture]: [仕様 001：デスクトップ入力](https://github.com/AstraFoundry/Arion/blob/main/specs/001-desktop-input.md)（`DSK-08`、`DSK-A01`）と[アーキテクチャの概要](https://github.com/AstraFoundry/Arion/blob/main/docs/project/architecture.md)の Start ステップ。

[^polish]: [クラウド整文](https://github.com/AstraFoundry/Arion/blob/main/docs/project/cloud-polishing.md)。メモが伝えるのは問題の種類だけで、あなたのテキストは含みません。

[^textcolor]: [ADR 017：共有のオフライン拼音キーボード](https://github.com/AstraFoundry/Arion/blob/main/docs/decisions/017-offline-pinyin-keyboard.md)。

[^files]: [クリップボード履歴](https://github.com/AstraFoundry/Arion/blob/main/docs/project/clipboard.md#file-access-guidance)、[ADR 047：クリップボードのファイルアクセスの案内](https://github.com/AstraFoundry/Arion/blob/main/docs/decisions/047-clipboard-file-access-guidance.md)、[仕様 010：クリップボード履歴](https://github.com/AstraFoundry/Arion/blob/main/specs/010-clipboard.md)（`CLP-09`）。

[^paste]: [ADR 003：ショートカットモードの挿入経路](https://github.com/AstraFoundry/Arion/blob/main/docs/decisions/003-insertion-routes.md)と [Pull Request #118](https://github.com/AstraFoundry/Arion/pull/118)。

[^selection]: [仕様 011：選択アクション](https://github.com/AstraFoundry/Arion/blob/main/specs/011-selection-actions.md)（`SEL-02`、`SEL-03`）と [ADR 048：選択アクションの URL](https://github.com/AstraFoundry/Arion/blob/main/docs/decisions/048-selection-action-urls.md)。

[^overview]: [デザインシステム](https://github.com/AstraFoundry/Arion/blob/main/docs/frontend/design-system.md#overview)の Overview の節、[Pull Request #127](https://github.com/AstraFoundry/Arion/pull/127)、[ADR 049：フラットな素材と浮き上がる素材](https://github.com/AstraFoundry/Arion/blob/main/docs/decisions/049-sunbeam-package-and-materials.md)。

[^thumbnails]: [仕様 010：クリップボード履歴](https://github.com/AstraFoundry/Arion/blob/main/specs/010-clipboard.md)（`CLP-06`）。

[^hold]: [ADR 003：ショートカットモードの挿入経路](https://github.com/AstraFoundry/Arion/blob/main/docs/decisions/003-insertion-routes.md)と [Pull Request #126](https://github.com/AstraFoundry/Arion/pull/126)。入力欄の変化はクリップボードを戻すタイミングを決めるだけで、貼り付けが成功した証拠にはなりません。

[^wechat]: [Pull Request #123](https://github.com/AstraFoundry/Arion/pull/123)。

[^update]: [アップデートのライフサイクル](https://github.com/AstraFoundry/Arion/blob/main/docs/project/update-lifecycle.md)と [Pull Request #137](https://github.com/AstraFoundry/Arion/pull/137)。
