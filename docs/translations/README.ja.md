# Scrawl

**Wayland 上の GNOME 向けの、注釈つきスクリーンショットツール。** キーを押して、好きなモニターで範囲を選び、矢印・枠・テキストを描き、見せたくない部分をぼかせば、結果はすぐクリップボードに入り、ファイルにも保存されます。

[English](../../README.md) · [Русский](README.ru.md) · [简体中文](README.zh-CN.md) · [Español](README.es.md) · [Português (Brasil)](README.pt-BR.md) · **日本語** · [Deutsch](README.de.md) · [Français](README.fr.md) · [한국어](README.ko.md)

![Scrawl のオーバーレイ：選択範囲に枠・矢印・文字・マーカー・ぼかしがあり、その下にツールバーがある](../images/overlay.png)

*画像は、架空のデスクトップの上に実際のオーバーレイを描画したものです。*

> 画面表示は、英語、ロシア語、簡体字中国語、スペイン語、ポルトガル語、日本語、ドイツ語、フランス語、韓国語で利用できます。以下のボタン名は日本語表示のものです。

## Scrawl を作った理由

Wayland 上の GNOME では、アプリは画面を直接取得できず、デスクトップのポータルに頼みます。そのポータルは**撮影のたびに「どの画面を共有しますか？」と尋ねます**。Scrawl はモニターごとに一度だけ尋ねて答えを覚えるので、その後のスクリーンショットはキー 1 回です。固定された画面は約 150 ミリ秒で表示されます。

## クイックスタート

Ubuntu または Debian（Wayland でも X11 でも）なら 3 ステップです。

**1. インストール：**署名つきの apt リポジトリを使います（以後の更新は `apt upgrade` で届きます）：

```bash
sudo curl -fsSL https://jexme.github.io/scrawl/scrawl.sources -o /etc/apt/sources.list.d/scrawl.sources
sudo apt update && sudo apt install scrawl
```

署名用の鍵は `.sources` ファイルの中に入っているので、キーリングを別に管理する必要はありません。パッケージは **amd64** と **arm64** 向けです（arm64 版はベストエフォートです）。ほかのアーキテクチャでは[ソースからビルド](#インストール)してください。

**2. 一度だけ設定する：**アプリケーションメニューから Scrawl を起動します（または `scrawl` を実行）。初回起動では、タブ「システム」に短いガイドが開きます（いつでも `scrawl -settings` で開けます）。Wayland では「キャプチャを設定」を押します（GNOME はモニターごとに一度尋ねます。ダイアログごとに別のモニターを選び、「この選択を記憶する」にチェックを入れてください）。次にホットキー（既定は `Print`）を入力して「割り当て」を押します。GNOME 標準のスクリーンショットツールも `Print` を使います。「キーを解放」を押すと、その内蔵ショートカットが無効になります（そのキー 1 つだけで、元に戻せます）。X11 ではホットキーだけで済みます。

**3. 撮影する：**ホットキーを押し、範囲をドラッグして、`Enter` を押します。画像はクリップボードと `~/Pictures/Screenshots` に入ります。

そのほかの環境：[Windows](#windows-10-と-11)、ほかの Linux ディストリビューション（リリースのアーカイブまたはソースからのビルド）は、下の[インストール](#インストール)を参照してください。

## 機能

**撮影**
- すべてのモニターで同時に動作：オーバーレイは各モニターに開き、どれでも範囲を選べます。
- 破線の枠で範囲を選択。角をドラッグしてサイズ変更。「選択」ツールでは、辺をドラッグしてサイズ変更、内側をドラッグして移動できます。
- 枠をドラッグしている間、カーソルの横の拡大鏡がピクセルを 6 倍に拡大して表示し、枠の辺をピクセルの格子に合わせるので、ぴったり位置を合わせられます（設定でオフにできます）。
- 前回の範囲を覚えていて、キー 1 つ（`R`）で呼び戻せます。
- オプションでマウスカーソルも写せます（今のところ Wayland のみ）。

**注釈**
- ペン、直線、矢印、枠、マーカー、**ぼかし**、テキスト、**番号スタンプ**（クリックすると 1、2、3…と数字入りの丸が置かれます。flameshot と同じです）。色は 8 色。
- マウスホイール（または `−` / `+` ボタン）で、太さ・文字サイズ・ぼかしの強さを変更。
- **ぼかし**には 6 段階あります（ホイールまたは `−` / `+`）：1〜3 はやわらかいガウスぼかし、4〜6 は四角が大きくなっていくモザイクです。モザイクのマス目は範囲の角から始まり、段階を変えたときに出る通知にモードの名前が表示されます。
- 描画中に **Shift** を押し続けると、ペンとマーカーは直線になり、直線と矢印は 45° に吸着し、枠とぼかしは正方形になります。
- `Ctrl+Z` で元に戻せます。

**保存と共有**
- `Enter` で画像をクリップボードにコピーし、PNG を `~/Pictures/Screenshots` に保存します。`Ctrl+C` はコピーのみ、`Ctrl+S` は保存のみ。JPEG も使えます。
- 選択範囲の中を**ダブルクリック**すると、すぐにコピーされます（設定のタブ「一般」、セクション「動作」で、コピーして保存、または何もしない、に変更することもできます）。
- **ビフォー/アフター コレクション：** `A` で撮影分をいったん取っておき、キャプション（「前」「後」、3 枚以上は「ステップ 1…」）つきの 1 枚に、横並びまたは縦並びで結合できます。コレクションを個別のファイルとして送ることもでき、チャットには添付ファイルとして貼り付けられます。

**自分好みに**
- ほぼすべてを設定ウィンドウで変更できます：キー、色、既定のツール、選択範囲の見た目、ファイル名と形式、表示言語（9 言語：英語、ロシア語、中国語、スペイン語、ポルトガル語、日本語、ドイツ語、フランス語、韓国語）。
- 自動起動つきのトレイアイコン、ホットキー設定ボタン、任意の更新確認（初期状態ではオフ）。

## ステータス

個人プロジェクトで、活発に開発中です。現在のバージョンはプレリリースです（`v0.1.0-rc4`）。すぐ使えるパッケージは [Releases](https://github.com/jexme/scrawl/releases) ページにあります。ソースからビルドすることもできます（数分で終わります）。Linux：主な対象は Wayland 上の GNOME です。**X11** セッション（Xorg 上の GNOME、KDE、XFCE など）でも動作しますが、このサポートは新しく、今のところ仮想 X サーバーでしか試していません。実機での報告を歓迎します。**Windows 10 と 11** でも動作します（画面取得、注釈、クリップボード、トレイアイコン、グローバルホットキー）。この移植版は新しく、100% のスケールでしか試していません。ビルド方法と未対応の点は [docs/WINDOWS.md](../WINDOWS.md) を参照してください。

## 必要なもの

apt または `.deb` でインストールすると、これらは自動で入ります。この一覧はリリースのアーカイブとソースからのビルド向けです。

- PipeWire を使う Wayland 上の GNOME、`xdg-desktop-portal`（1.21 以降、アプリ登録に対応していること）、`xdg-desktop-portal-gnome`。開発とテストは Ubuntu + GNOME 50 で行っています。
- クリップボード用に `wl-clipboard`（`wl-copy` コマンド）、通知用に `libnotify`（`notify-send` コマンド）。`wl-copy` がない場合、`xclip` があれば Scrawl は XWayland 経由でそれを使います。どちらもなければ、何をインストールすればよいかが表示されます。
- **X11** ではポータルがなく、画面は直接取得されます。クリップボードには `wl-clipboard` の代わりに `xclip` が必要です。

```bash
sudo apt install wl-clipboard libnotify-bin   # Wayland
sudo apt install xclip libnotify-bin          # X11
```

## インストール

### 1. プログラムを入手する

**apt リポジトリから** (Debian、Ubuntu。更新は `apt upgrade` で届きます):

```bash
sudo curl -fsSL https://jexme.github.io/scrawl/scrawl.sources -o /etc/apt/sources.list.d/scrawl.sources
sudo apt update && sudo apt install scrawl
```

パッケージは amd64 と arm64 向けです（arm64 版はベストエフォートです。ほかのアーキテクチャではソースからビルドしてください。下を参照）。リポジトリは次の鍵で署名されています: `E62C DAF5 A1C9 0DB7 A899 D24D FA43 15C7 154A 9EF0`。この鍵は `scrawl.sources` に埋め込まれています。

古いシステム (Ubuntu 20.04、Debian 11。apt 2.4 未満では `.sources` ファイル内の鍵を読めません) では、代わりに別個のキーリングと 1 行形式のソースを使います (コマンドとファイル名は英語版 README と同じです):

```bash
curl -fsSL https://jexme.github.io/scrawl/scrawl-archive-keyring.gpg | sudo tee /usr/share/keyrings/scrawl-archive-keyring.gpg >/dev/null
echo "deb [signed-by=/usr/share/keyrings/scrawl-archive-keyring.gpg] https://jexme.github.io/scrawl stable main" | sudo tee /etc/apt/sources.list.d/scrawl.list
sudo apt update && sudo apt install scrawl
```

鍵は ASCII アーマー形式の `scrawl-archive-keyring.asc` としても公開されています。

**.deb パッケージから** (Debian、Ubuntu)：[Releases](https://github.com/jexme/scrawl/releases) ページから `scrawl_VERSION_amd64.deb` をダウンロードし、`sudo apt install ./scrawl_*.deb` を実行します。必要なライブラリが入り、`wl-clipboard`、`xclip`、`libnotify-bin` が推奨パッケージとして加わります。ホットキーはパッケージでは割り当てられません（ユーザーごとの設定のためです）。`scrawl -settings` で設定してください。自分でビルドするには `make deb` を使います。

**リリースのアーカイブから：** [Releases](https://github.com/jexme/scrawl/releases) ページから `scrawl-VERSION-linux-amd64.tar.gz` と `SHA256SUMS` をダウンロードして、次を実行します：

```bash
sha256sum -c --ignore-missing SHA256SUMS   # checks every downloaded file listed in SHA256SUMS
tar xzf scrawl-*-linux-amd64.tar.gz && cd scrawl-*-linux-amd64
./install.sh            # ~/.local にインストール（システム全体なら PREFIX=/usr/local）
sudo apt install libpipewire-0.3-0 libwayland-client0 libwayland-egl1 libwayland-cursor0 \
  libxkbcommon-x11-0 libegl1 libx11-6 libxcursor1 wl-clipboard libnotify-bin
```

`wl-clipboard`（クリップボード）と `libnotify-bin`（通知）は実行時にも必要です。インストーラーがそれらを確認し、足りないものを表示します。X11 では `wl-clipboard` の代わりに `xclip` を使います。

**ソースから：**

```bash
sudo apt install gcc pkg-config patch libwayland-dev libx11-dev libx11-xcb-dev \
  libxkbcommon-x11-dev libgles2-mesa-dev libegl1-mesa-dev libffi-dev \
  libxcursor-dev libvulkan-dev libpipewire-0.3-dev
git clone https://github.com/jexme/scrawl.git && cd scrawl
make install            # ビルドして ~/.local にインストール（システム全体なら PREFIX=/usr/local）
```

Go（バージョンは `go.mod` を参照）が必要です。`sudo` が使えない場合は、`scripts/fetch-pipewire-headers.sh` が PipeWire のヘッダーを `.deps/` に展開し、Makefile が自動で使います。

`~/.local/bin` が `PATH` に入っていることを確認してください。

### Windows 10 と 11

**おすすめ：インストーラー。** [Releases](https://github.com/jexme/scrawl/releases) ページから `scrawl-VERSION-windows-amd64-setup.exe` をダウンロードして実行します。自分のアカウント専用にインストールされ（管理者権限は不要）、スタートメニューに項目が追加されます。サインイン時に Scrawl を起動する設定も選べ、インストールが終わると Scrawl が起動します。更新するには新しいセットアップを実行してください。古い Scrawl を先に終了してくれます。リリースにセットアップファイルがない場合は、下のいずれかの方法を使ってください。

そのほかの方法：

- **ポータブル zip。** `scrawl-VERSION-windows-amd64.zip` とその `.sha256` をダウンロードし、常設のフォルダーに展開して（アーカイブビューアーから直接ではなく）`scrawl.exe` を実行します。`scrawl.exe` をダブルクリックすると、通知領域のアイコンが起動します。単体の `scrawl-VERSION-windows-amd64.exe` も公開されています。更新するには、アイコンのメニューから Scrawl を終了して、ファイルを置き換えます。
- **[Scoop](https://scoop.sh)：** `scoop install https://github.com/jexme/scrawl/releases/download/v0.1.0-rc4/scrawl.json`。このアドレスは 1 つのプレリリースに固定されているため、リリース候補の間で更新するには、`scoop uninstall scrawl` を実行してから、新しいバージョンの URL で `scoop install` を実行します（安定版リリースか bucket ができるまで）。マニフェストを `https://github.com/jexme/scrawl/releases/latest/download/scrawl.json` から取得できるようになれば（安定版リリースがあれば）、`scoop update scrawl` が使えます。

- **アイコン**は通知領域にあります。Windows 11 では時計の横の `^` 矢印の中に隠れることがあるので、外へドラッグしておくと常に表示されます。
- **Print キー**は既定のホットキーです。ほかのプログラム（Greenshot、ShareX、Lightshot）がこのキーを握っていることがよくあります。Windows 自体が握っていることもあります：設定 → アクセシビリティ → キーボード → *Print Screen ボタンを使って画面キャプチャを開く*：これをオフにしてください。または `scrawl.exe -settings` で別のキー（たとえば `<Control><Alt>a`）を選びます。キーが使われているときはアイコンが知らせ、そのまま再試行を続けます。
- **「Windows によって PC が保護されました」**（SmartScreen）が表示されるのは、プログラムにまだコード署名がないためです：*詳細情報*をクリックし、続けて*実行*をクリックしてください。先にファイルを確認するには、`Get-FileHash` の結果を `.sha256` ファイルと比べるか、`gh attestation verify scrawl-*-setup.exe --repo jexme/scrawl` を実行します（ビルドは公開 CI で行われています）。ウイルス対策ソフトは、署名のない新しい Go のプログラムを警告することがあります。そうした場合は issue で報告してください。
- **自動起動**は、`scrawl.exe -settings` のタブ「システム」にあるチェックボックス（`HKCU\...\Run` の値）、またはインストーラーのチェックボックスです。プログラムを移動したり更新したりした場合は、次にアイコンが起動したときに、エントリが新しい場所に書き直されます（オフにした自動起動はオフのままです）。
- **古いバージョンが動作中に新しいバージョンを起動した場合：** Scrawl が、どのバージョンが動作中かを知らせます。先にアイコンのメニューから終了してから、新しいほうを起動してください。

ソースからのビルドと未対応の点：[docs/WINDOWS.md](../WINDOWS.md)。

### 2. 初回起動：一度だけの 3 ステップ

*X11 ではステップ 1 と 2 は不要です（ポータルがないため）。ホットキーだけ設定してください。*

設定ウィンドウ（`scrawl -settings`、タブ「システム」）を開きます。ボタン「キャプチャを設定」でステップ 1 と 2 をまとめて行えます（ごく初回の起動ではウィンドウが自動で開きます）。または個別のボタンを使うか、コマンドを実行します：

1. **モニターを選ぶ：** ボタン「モニターを選び直す」または `scrawl -setup`。GNOME が**モニターごとに**ダイアログを表示します。毎回*別の*モニターを選び、**「この選択を記憶する」**にチェックを入れたまま共有を許可してください。以後ダイアログは出ません。
2. **予備の権限を与える：** ボタン「権限を付与」または `scrawl -grant-screenshot`。PipeWire から映像が来ないごくまれな場合に、Scrawl がより遅い Screenshot ポータルを使えるようになります。
3. **ホットキーを設定する：** GNOME では、アーカイブのインストーラー（`install.sh`）と `make install` が、ほかのカスタムショートカットにすでに使われていない限り、`Print` を自動で割り当てます。apt や `.deb` でインストールした後、またはキーを変更したいときは、「起動ホットキー」欄と「割り当て」ボタン（既定は `Print`）、または `scripts/gnome-hotkey.sh [キー]` を使います。「scrawl」という名前の GNOME カスタムショートカットを作成し（すでにあれば更新するので、再実行しても 2 つ目はできません）、すでに使われている他人のショートカットは上書きしません。`Print` が GNOME 標準のスクリーンショットツールを開いてしまう場合は、*設定 → キーボード → キーボードショートカットを表示およびカスタマイズ → スクリーンショット* で無効にしてください。

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
| `Esc` | キャンセル（開いているメニューがあれば先にそれを閉じる。テキスト入力中はテキストだけを破棄） |
| `Ctrl+A` | そのモニターの画面全体を選択（キーの変更は不可） |
| `R` | 前回の撮影範囲を呼び戻す |
| `V` `P` `L` `W` `B` `M` `U` `T` `N` | ツール：選択、ペン、直線、矢印、枠、マーカー、ぼかし、テキスト、番号スタンプ（範囲を選択しているとき） |
| `+` / `-` | 現在のツールの太さ・文字サイズ・ぼかしの強さ（マウスホイールと同じ） |
| `A` | 撮影をコレクションに追加 |
| `G` | コレクション（と現在の範囲）を 1 枚に結合 |
| `D` | コレクションを個別のファイルとして保存し、ファイル一覧をクリップボードに入れる |
| マウスホイール | 現在のツールの太さ・文字サイズ・ぼかしの強さ |
| 描画中の `Shift` | ペンとマーカー：直線。直線と矢印：45° に吸着。枠とぼかし：正方形。範囲選択中の `Shift`：正方形 |

どのキーも設定ウィンドウ（タブ「キー」）で変更できます。

### ツールバー

メインの行は左から：選択、ペン、直線、矢印、枠、マーカー、ぼかし、テキスト、番号 · 元に戻す · コピー · 保存 · `⋯` メニュー（設定：オーバーレイを閉じて設定ウィンドウを開く、およびコレクションに追加）· 閉じる。

描画ツールを使っているときは、その下に 2 行目が現れます：8 色、自分の色を選ぶための虹色のボタン（パレットが開き、色は記憶されます）、そして `−` 太さ `+`。「選択」ツールにはどちらもないので、この行は隠れます。

キーが割り当てられているボタンには、アイコンの下にそのキーが表示されます（キーは設定から取られます）。ボタンの上にカーソルをしばらく置くと、名前とキーが表示されます。ショートカットはどのキーボード配列でも（たとえばロシア語配列でも）、キーの位置で動作します。

### ビフォー/アフター コレクション

最初の撮影をして `A` を押します。次の撮影では前回の範囲がすでに選択されているので、同じ場所をもう一度撮れます。`G` で撮影分を 1 枚の画像に結合（保存とコピーが行われます）、`D` で個別のファイルになります。画面下の小さなパネルにはコレクションの内容が表示され、同じ操作をボタンで行えます。

## トレイアイコンと自動起動

`scrawl -tray` は通知領域にアイコンを置きます。左クリックでスクリーンショット、メニューには「スクリーンショットを撮る」、「設定」、「終了」があります。ログイン時に起動するには *設定 → システム → 自動起動* をオンにします。GNOME ではアイコンの表示に AppIndicator 拡張機能が必要です（Ubuntu では既定で有効）。

## 設定

![設定ウィンドウ](../images/settings-en.png)

ツールバーの歯車、`scrawl -settings`、またはアプリケーションメニューから開きます。変更はすぐ `~/.config/scrawl/config.json`（通常の JSON ファイルで、手で編集することもできます）に保存されます。欠けている値や無効な値は既定値に戻ります。各キーの説明は[設定リファレンス](../SETTINGS.md)にあります。

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
| `-exit-after DURATION` | 指定した時間がたつと通常どおり終了する。テスト用 (例: `-exit-after 3s`) |

## トラブルシューティング

- **ホットキーを押しても何も起きない。** `xdg-desktop-portal` が固まっている可能性があります。再起動してください：`systemctl --user restart xdg-desktop-portal xdg-desktop-portal-gnome`。
- **撮影のたびに GNOME が共有する画面を尋ねる。** 「この選択を記憶する」にチェックが入っていませんでした。ステップ 1 をやり直してください。
- **モニターを追加または交換した。** モニターを選び直します（ステップ 1）。
- **オーバーレイが 1 つのモニターにしか開かない。** 同じです。ステップ 1 はすべてのモニターで行う必要があります。
- **ポータルに拒否されたと表示される。** 権限を与えてください（ステップ 2）。
- **クリップボードのエラーが出る、または貼り付けても何も出ない。** `wl-copy`（パッケージ `wl-clipboard`）がインストールされているか確認してください。
- **通知が出ない。** `libnotify-bin`（`notify-send` コマンド）をインストールし、設定で通知がオンになっているか確認してください。
- **トレイアイコンが出ない。** GNOME の AppIndicator 拡張機能を有効にしてください。
- **`Print` で GNOME のスクリーンショットが開く。** キーボードショートカットで無効にしてください（ステップ 3 を参照）。

## アンインストール

**apt または `.deb`：**

```bash
sudo apt remove scrawl     # or: sudo apt purge scrawl (the package keeps no files outside /usr, so they are the same)
sudo rm /etc/apt/sources.list.d/scrawl.sources   # optional: forget the repository (scrawl.list for the old form)
```

apt はホームフォルダーには触れられないため、ユーザーごとのものがいくつか残ります：GNOME のホットキー、オンにしていた場合の自動起動の項目、そして設定です。ホットキーは Scrawl の `gnome-hotkey.sh`（リポジトリの `scripts/` とリリースのアーカイブに入っています）で削除するか、*設定 → キーボード → キーボードショートカットを表示およびカスタマイズ → カスタムショートカット* で手動で削除します。残りは普通のファイルです：

```bash
sh gnome-hotkey.sh --remove
rm -f ~/.config/autostart/io.github.jexme.Scrawl.desktop
rm -rf ~/.config/scrawl ~/.cache/scrawl     # optional: settings and cache
```

**リリースのアーカイブまたは `make install`：** 展開したアーカイブで `./uninstall.sh` を実行します（ソースのチェックアウトでは `make uninstall`）。実行ファイル、`.desktop` ファイル、GNOME のショートカット、自動起動の項目が削除されます。`--purge`（`make uninstall PURGE=1`）を付けると、設定（`~/.config/scrawl`）とキャッシュ（`~/.cache/scrawl`）も削除されます。インストール時に `PREFIX` を変えた場合は、同じ `PREFIX` を渡してください。`~/Pictures/Screenshots` のスクリーンショットには一切触れません。

## 仕組み

Scrawl は、`xdg-desktop-portal` の ScreenCast インターフェースを通じて、独自の小さな PipeWire クライアント（`internal/pwcapture`）で画面を取得します。モニターごとに保存した復元トークンでダイアログが不要になります。コンポジター（Mutter）はモニターが再描画されたときだけフレームを渡し、私たちのテストでは全画面ウィンドウの上（ダイレクトスキャンアウト）では 1 枚も渡しませんでした。250 ミリ秒以内にフレームが来なければ、Scrawl はより遅い（約 0.5〜1 秒）Screenshot ポータルに切り替えます。オーバーレイは全画面の [Gio](https://gioui.org) ウィンドウで、モニターごとに 1 つです。Gio には、ウィンドウを指定した Wayland 出力に置くためのパッチ（`third_party/gio.patch`）を当てています。結果は画面に描くのと同じコードで描画されるため、保存される画像は見たものと一致します。

## ビルドと開発への参加

```bash
make build    # bin/scrawl；初回は Gio を取得してパッチを適用
make test     # 単体テスト（セッション D-Bus 上のトレイアイコンを含む）
make dist     # dist/ にリリースのアーカイブとチェックサムを作成
make deb      # .deb package for Debian and Ubuntu in dist/
make notices  # THIRD_PARTY_NOTICES.md を再生成
```

画面の文言は `internal/i18n/`（言語ごとに 1 ファイル。`lang_ru.go`、`lang_de.go` など。コード中の元の文言は英語）にあり、`go test ./...` がすべての文言がすべての言語に翻訳されていることを確認します。バグの報告と変更の送り方：[CONTRIBUTING.md](../../CONTRIBUTING.md)。計画とアイデア：[docs/PLAN.md](../PLAN.md)。

## ライセンス

MIT。[LICENSE](../../LICENSE) を参照。サードパーティのコンポーネント：[THIRD_PARTY_NOTICES.md](../../THIRD_PARTY_NOTICES.md)（アイコンの形状は [Tabler Icons](https://tabler.io/icons)、MIT）。
