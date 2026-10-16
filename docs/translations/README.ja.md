# Scrawl

**Wayland 上の GNOME 向けの、注釈つきスクリーンショットツール。** キーを押して、好きなモニターで範囲を選び、矢印・枠・テキストを描き、見せたくない部分をぼかせば、結果はすぐクリップボードに入り、ファイルにも保存されます。

[English](../../README.md) · [Русский](../../README.ru.md) · [简体中文](README.zh-CN.md) · [Español](README.es.md) · [Português (Brasil)](README.pt-BR.md) · **日本語** · [Deutsch](README.de.md) · [Français](README.fr.md) · [한국어](README.ko.md)

![Scrawl のオーバーレイ：選択範囲に枠・矢印・文字・マーカー・ぼかしがあり、その下にツールバーがある](../images/overlay.png)

*画像は、架空のデスクトップの上に実際のオーバーレイを描画したものです。*

> プログラムの画面表示は今のところ英語とロシア語です。以下のボタン名は英語表示のものです。

## Scrawl を作った理由

Wayland 上の GNOME では、アプリは画面を直接取得できず、デスクトップのポータルに頼みます。そのポータルは**撮影のたびに「どの画面を共有しますか？」と尋ねます**。Scrawl はモニターごとに一度だけ尋ねて答えを覚えるので、その後のスクリーンショットはキー 1 回です。固定された画面は約 150 ミリ秒で表示されます。

## 機能

**撮影**
- すべてのモニターで同時に動作：オーバーレイは各モニターに開き、どれでも範囲を選べます。
- 破線の枠で範囲を選択。角をドラッグしてサイズ変更。*選択*ツールでは、辺をドラッグしてサイズ変更、内側をドラッグして移動できます。
- 前回の範囲を覚えていて、キー 1 つ（`R`）で呼び戻せます。
- オプションでマウスカーソルも写せます。

**注釈**
- ペン、直線、矢印、枠、マーカー、**ぼかし**、テキスト。色は 8 色。
- マウスホイール（または `−` / `+` ボタン）で、太さ・文字サイズ・ぼかしの強さを変更。
- `Ctrl+Z` で元に戻せます。

**保存と共有**
- `Enter` で画像をクリップボードにコピーし、PNG を `~/Pictures/Screenshots` に保存します。`Ctrl+C` はコピーのみ、`Ctrl+S` は保存のみ。JPEG も使えます。
- **ビフォー/アフター コレクション：** `A` で撮影分をいったん取っておき、キャプション（「Before」「After」、3 枚以上は「Step 1…」）つきの 1 枚に、横並びまたは縦並びで結合できます。コレクションを個別のファイルとして送ることもでき、チャットには添付ファイルとして貼り付けられます。

**自分好みに**
- ほぼすべてを設定ウィンドウで変更可能：キー、色、既定のツール、選択範囲の見た目、ファイル名と形式、言語（英語、ロシア語）。
- 自動起動つきのトレイアイコン、ホットキー設定ボタン、任意の更新確認（初期状態ではオフ）。

## ステータス

個人プロジェクトで、活発に開発中です。**まだリリースはありません**。今のところソースからのビルドが必要です（数分で終わります）。主な対象は Wayland 上の GNOME（Linux）です。**X11** セッション（Xorg 上の GNOME、KDE、XFCE など）でも動作しますが、このサポートは新しく、今のところ仮想 X サーバーでしか試していません。実機での報告を歓迎します。Windows 版はビルドできますが、画面取得はまだ実装されていません。詳しくは [docs/WINDOWS.md](../WINDOWS.md) を参照してください。

## 必要なもの

- PipeWire を使う Wayland 上の GNOME、`xdg-desktop-portal`（1.21 以降、アプリ登録に対応していること）、`xdg-desktop-portal-gnome`。開発とテストは Ubuntu + GNOME 50 で行っています。
- クリップボード用に `wl-clipboard`（`wl-copy` コマンド）、通知用に `libnotify`（`notify-send` コマンド）。
- **X11** ではポータルがなく、画面は直接取得されます。クリップボードには `wl-clipboard` の代わりに `xclip` が必要です。

```bash
sudo apt install wl-clipboard libnotify-bin   # Wayland
sudo apt install xclip libnotify-bin          # X11
```

## インストール

### 1. プログラムを入手する

**ソースから（今すぐ使えます）：**

```bash
sudo apt install gcc pkg-config patch libwayland-dev libx11-dev libx11-xcb-dev \
  libxkbcommon-x11-dev libgles2-mesa-dev libegl1-mesa-dev libffi-dev \
  libxcursor-dev libvulkan-dev libpipewire-0.3-dev
git clone https://github.com/jexme/scrawl.git && cd scrawl
make install            # ビルドして ~/.local にインストール（システム全体なら PREFIX=/usr/local）
```

Go（バージョンは `go.mod` を参照）が必要です。`sudo` が使えない場合は、`scripts/fetch-pipewire-headers.sh` が PipeWire のヘッダーを `.deps/` に展開し、Makefile が自動で使います。

**リリースのアーカイブから（公開後）：** [Releases](https://github.com/jexme/scrawl/releases) ページから `scrawl-VERSION-linux-amd64.tar.gz` と `SHA256SUMS` をダウンロードして、次を実行します：

```bash
sha256sum -c --ignore-missing SHA256SUMS
tar xzf scrawl-*-linux-amd64.tar.gz && cd scrawl-*-linux-amd64
./install.sh            # ~/.local にインストール（システム全体なら PREFIX=/usr/local）
sudo apt install libpipewire-0.3-0 libwayland-client0 libwayland-egl1 libwayland-cursor0 \
  libxkbcommon-x11-0 libegl1 libx11-6 libxcursor1
```

`~/.local/bin` が `PATH` に入っていることを確認してください。

### 2. 初回起動：一度だけの 3 ステップ

*X11 ではステップ 1 と 2 は不要です（ポータルがないため）。ホットキーだけ設定してください。*

設定ウィンドウ（`scrawl -settings`）のボタンを使うか、コマンドを実行します：

1. **モニターを選ぶ：** ボタン “Pick monitors again” または `scrawl -setup`。GNOME が**モニターごとに**ダイアログを表示します。毎回*別の*モニターを選び、**「この選択を記憶する」**にチェックを入れたまま共有を許可してください。以後ダイアログは出ません。
2. **予備の権限を与える：** ボタン “Grant permission” または `scrawl -grant-screenshot`。PipeWire から映像が来ないごくまれな場合に、Scrawl がより遅い Screenshot ポータルを使えるようになります。
3. **ホットキーを設定する：** “Launch hotkey” 欄と “Assign” ボタン（既定は `Print`）、または `scripts/gnome-hotkey.sh [キー]`。GNOME のカスタムショートカットを作成し、すでに使われている組み合わせは上書きしません。`Print` が GNOME 標準のスクリーンショットを開いてしまう場合は、*設定 → キーボード → キーボードショートカットを表示およびカスタマイズ → スクリーンショット* で無効にしてください。

### screen-editor からの移行

Scrawl は screen-editor の新しい名前です。`~/.config/scrawl/config.json` ができるまでは `~/.config/screen-editor/config.json` から設定を読み込みます（古いファイルは変更されません）。ポータルはモニターの選択と権限をアプリ ID に結び付けており、その ID が変わったので、ステップ 1〜3 をもう一度行い、ホットキーを `scrawl` に向け直してください。前回の範囲とコレクションは新しく始まります。

## 使い方

1. ホットキーを押します。画面が固定され、暗くなります。
2. ドラッグして範囲を選びます（どのモニターでも可）。下にツールバーが現れます。
3. 必要なら注釈を付けます：ツールと色を選び、範囲の上にドラッグして描きます。
4. `Enter` を押します。画像はクリップボードと `~/Pictures/Screenshots` に入るので、好きな場所に貼り付けられます。

`Esc` でいつでもキャンセルできます。

### キー

| キー | 動作 |
| --- | --- |
| `Enter` | クリップボードにコピー**して**ファイルに保存（動作は設定で変更可能） |
| `Ctrl+C` / `Ctrl+S` | コピーのみ / 保存のみ |
| `Ctrl+Z` | 直前の注釈を取り消す |
| `Esc` | キャンセル |
| `R` | 前回の撮影範囲を呼び戻す |
| `A` | 撮影をコレクションに追加 |
| `G` | コレクション（と現在の範囲）を 1 枚に結合 |
| `D` | コレクションを個別のファイルとして保存し、ファイル一覧をクリップボードに入れる |
| マウスホイール | 現在のツールの太さ・文字サイズ・ぼかしの強さ |

どのキーも設定ウィンドウ（“Overlay keys”）で変更できます。

### ツールバー

左から：選択、ペン、直線、矢印、枠、マーカー、ぼかし、テキスト · 8 色 · `−` 太さ `+` · 設定（オーバーレイを閉じて設定ウィンドウを開く）· コレクションに追加 · 元に戻す · コピー · 保存 · 閉じる。

### ビフォー/アフター コレクション

最初の撮影をして `A` を押します。次の撮影では前回の範囲がすでに選択されているので、同じ場所をもう一度撮れます。`G` で撮影分を 1 枚の画像に結合（保存とコピーが行われます）、`D` で個別のファイルになります。画面下の小さなパネルにはコレクションの内容が表示され、同じ操作をボタンで行えます。

## トレイアイコンと自動起動

`scrawl -tray` は通知領域にアイコンを置きます。左クリックでスクリーンショット、メニューには “Screenshot”、“Settings”、“Quit” があります。ログイン時に起動するには *Settings → Startup → Autostart* をオンにします。GNOME ではアイコンの表示に AppIndicator 拡張機能が必要です（Ubuntu では既定で有効）。

## 設定

![設定ウィンドウ](../images/settings-en.png)

ツールバーの歯車、`scrawl -settings`、またはアプリケーションメニューから開きます。変更はすぐ `~/.config/scrawl/config.json`（通常の JSON ファイルで、手で編集することもできます）に保存されます。欠けている値や無効な値は既定値に戻ります。各キーの説明は[設定リファレンス](../SETTINGS.md)（英語、または[ロシア語](../SETTINGS.ru.md)）にあります。

## ファイルとプライバシー

- 画像は `~/Pictures/Screenshots`（設定で変更可能）に、`scrawl-20260131-142501.png` のような名前で保存されます。
- 設定：`~/.config/scrawl/`。モニターのトークン、前回の範囲、コレクション：`~/.cache/scrawl/`。
- Scrawl が自分からインターネットに接続することはありません。プログラム内の唯一のネットワーク通信は、任意の更新確認（初期状態ではオフ）で、GitHub に最新リリースを問い合わせ、通常の HTTP ヘッダーのほかにはプログラムのバージョンだけを送ります。
- スクリーンショットは、あなたがどこかに貼り付けない限り、パソコンの外には出ません。

## コマンドライン

| オプション | 内容 |
| --- | --- |
| `-settings` | 設定ウィンドウを開く |
| `-setup` | モニターを選び直す（GNOME のダイアログ） |
| `-grant-screenshot` | 予備の撮影権限を与えて終了する |
| `-tray` | 通知領域にアイコンを置く |
| `-fresh` | 前回の範囲を提案しない |
| `-v` | 所要時間を表示する |
| `-version` | バージョンを表示して終了する |

## トラブルシューティング

- **ホットキーを押しても何も起きない。** `xdg-desktop-portal` が固まっている可能性があります。再起動してください：`systemctl --user restart xdg-desktop-portal xdg-desktop-portal-gnome`。
- **撮影のたびに GNOME が共有する画面を尋ねる。** 「この選択を記憶する」にチェックが入っていませんでした。ステップ 1 をやり直してください。
- **モニターを追加または交換した。** モニターを選び直します（ステップ 1）。
- **オーバーレイが 1 つのモニターにしか開かない。** 同じです。ステップ 1 はすべてのモニターで行う必要があります。
- **ポータルに拒否されたと表示される。** 権限を与えてください（ステップ 2）。
- **一部のアプリに貼り付けても何も出ない。** `wl-copy`（パッケージ `wl-clipboard`）がインストールされているか確認してください。
- **通知が出ない。** `libnotify-bin`（`notify-send` コマンド）をインストールし、設定で通知がオンになっているか確認してください。
- **トレイアイコンが出ない。** GNOME の AppIndicator 拡張機能を有効にしてください。
- **`Print` で GNOME のスクリーンショットが開く。** キーボードショートカットで無効にしてください（ステップ 3 を参照）。

## 仕組み

Scrawl は、`xdg-desktop-portal` の ScreenCast インターフェースを通じて、独自の小さな PipeWire クライアント（`internal/pwcapture`）で画面を取得します。モニターごとに保存した復元トークンでダイアログが不要になります。コンポジター（Mutter）はモニターが再描画されたときだけフレームを渡し、私たちのテストでは全画面ウィンドウの上（ダイレクトスキャンアウト）では 1 枚も渡しませんでした。250 ミリ秒以内にフレームが来なければ、Scrawl はより遅い（約 0.5〜1 秒）Screenshot ポータルに切り替えます。オーバーレイは全画面の [Gio](https://gioui.org) ウィンドウで、モニターごとに 1 つです。Gio には、ウィンドウを指定した Wayland 出力に置くためのパッチ（`third_party/gio.patch`）を当てています。結果は画面に描くのと同じコードで描画されるため、保存される画像は見たものと一致します。

## ビルドと開発への参加

```bash
make build    # bin/scrawl；初回は Gio を取得してパッチを適用
make test     # 単体テスト（セッション D-Bus 上のトレイアイコンを含む）
make dist     # dist/ にリリースのアーカイブとチェックサムを作成
make notices  # THIRD_PARTY_NOTICES.md を再生成
```

画面の文言は `internal/i18n/en.go`（英語。コード中の元の文言はロシア語）にあり、`go test ./...` がすべての文言に翻訳があることを確認します。計画とアイデア：[docs/PLAN.md](../PLAN.md)（ロシア語）。

## ライセンス

MIT。[LICENSE](../../LICENSE) を参照。サードパーティのコンポーネント：[THIRD_PARTY_NOTICES.md](../../THIRD_PARTY_NOTICES.md)（アイコンの形状は [Tabler Icons](https://tabler.io/icons)、MIT）。
