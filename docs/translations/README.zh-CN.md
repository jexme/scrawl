# Scrawl

**面向 Wayland 下 GNOME 的快速截图与标注工具。** 按下快捷键，在任意显示器上框选区域，画箭头、方框和文字，把隐私内容打码，结果立刻进入剪贴板并保存为文件。

[English](../../README.md) · [Русский](README.ru.md) · **简体中文** · [Español](README.es.md) · [Português (Brasil)](README.pt-BR.md) · [日本語](README.ja.md) · [Deutsch](README.de.md) · [Français](README.fr.md) · [한국어](README.ko.md)

![Scrawl 浮层：选中的区域上有方框、箭头、文字说明、荧光笔和模糊处理，下方是工具栏](../images/overlay.png)

*图片是在虚构桌面上真实渲染的浮层。*

> 程序界面提供英语、俄语、简体中文、西班牙语、葡萄牙语、日语、德语、法语和韩语；下文的按钮名称按中文界面给出。

## 为什么选择 Scrawl

在 Wayland 下的 GNOME 中，应用不能直接截取屏幕，只能请求桌面门户（portal），而门户**每次截图都会问“要共享哪个屏幕？”**。Scrawl 对每个显示器只问一次并记住答案，之后截图只需按一个键：冻结的屏幕约 150 毫秒内出现。

## 快速开始

在 Ubuntu 或 Debian（Wayland 或 X11）上只需三步。

**1. 安装**，使用带签名的 apt 仓库（之后更新通过 `apt upgrade` 获取）：

```bash
sudo curl -fsSL https://jexme.github.io/scrawl/scrawl.sources -o /etc/apt/sources.list.d/scrawl.sources
sudo apt update && sudo apt install scrawl
```

`.sources` 文件里已经带有签名密钥，不需要另外管理密钥环。软件包提供 **amd64** 和 **arm64** 版本（arm64 版本尽力而为）；其他架构请[从源码构建](#安装)。

**2. 设置一次：**从应用菜单启动 Scrawl（或运行 `scrawl`）：首次启动会在“系统”标签页打开一份简短的引导（任何时候都可以用 `scrawl -settings` 打开）。在 Wayland 上点击“设置截图”（GNOME 会对每个显示器询问一次：在每个对话框里选择不同的显示器，并勾选“记住此选择”）。然后输入快捷键（默认 `Print`），点击“分配”。GNOME 自带的截图工具也使用 `Print`：点击“释放按键”可关闭这个内置快捷键（只关闭这一个键，并且可以撤销）。在 X11 上只需要设置快捷键。

**3. 截图：**按下快捷键，拖出一个区域，按 `Enter`。图片已在剪贴板中，并保存在 `~/Pictures/Screenshots`。

其他系统：[Windows](#windows-10-和-11)，其他 Linux 发行版（发布压缩包或从源码构建），见下面的[安装](#安装)。

## 功能

**截图**
- 同时支持所有显示器：浮层会在每个显示器上打开，可在任意一个上框选。
- 虚线框选。拖动角点调整大小；使用“选择”工具时，可拖动边缘调整大小，或拖动内部移动区域。
- 拖动边框时，光标旁的放大镜会把像素放大 6 倍，并让边框的边缘对齐像素网格，方便精确对准（可在设置中关闭）。
- 记住上一次的区域，按一个键（`R`）即可恢复。
- 可选：把鼠标指针也截进图片（目前仅限 Wayland）。

**标注**
- 画笔、直线、箭头、方框、荧光笔、**模糊**、文字和**编号标记**（点击一下就放上一个带 1、2、3……的圆圈，类似 flameshot）；八种颜色。
- 鼠标滚轮（或 `−` / `+` 按钮）调整粗细、文字大小或模糊强度。
- **模糊**有六个级别（滚轮或 `−` / `+`）：1–3 为柔和的高斯模糊，4–6 为方块逐级变大的马赛克。马赛克的方格从区域的角开始排列，调整级别时弹出的提示会写明当前模式。
- 绘制时按住 **Shift**：画笔和荧光笔画直线，直线或箭头吸附到 45°，方框或模糊变成正方形。
- `Ctrl+Z` 撤销。

**保存与分享**
- `Enter` 把图片复制到剪贴板，并把 PNG 保存到 `~/Pictures/Screenshots`。`Ctrl+C` 只复制，`Ctrl+S` 只保存。也支持 JPEG。
- 在选区内**双击**会立即复制（在设置的“常规”标签页、“操作”部分中，也可以改为复制并保存，或不执行任何操作）。
- **前后对比合集：**用 `A` 把截图先放进合集，再拼成一张带说明文字的图（两张时为“之前”/“之后”，更多时为“步骤 1…”），可左右并排或上下排列；也可以把合集作为多个独立文件发送，粘贴到聊天里就是附件。

**个性化**
- 几乎所有选项都在设置窗口里：快捷键、颜色、默认工具、选区外观、文件名和格式，以及界面语言（9 种：英语、俄语、中文、西班牙语、葡萄牙语、日语、德语、法语、韩语）。
- 带自启动的托盘图标、设置快捷键的按钮，以及可选的更新检查（默认关闭）。

## 状态

个人项目，正在积极开发；当前版本是预发布版本（`v0.1.0-rc4`）。现成的安装包在 [Releases](https://github.com/jexme/scrawl/releases) 页面；也可以从源码构建（几分钟即可）。Linux：主要目标是 Wayland 下的 GNOME。**X11 会话**（Xorg 下的 GNOME、KDE、XFCE 等）也能使用，但这项支持是新的，目前只在虚拟 X 服务器上测试过，欢迎在真实机器上反馈。**Windows 10 和 11** 也能使用（截图、标注、剪贴板、托盘图标和全局快捷键）；这个移植版是新的，目前只在 100% 缩放下试用过，构建方法和尚未实现的部分见 [docs/WINDOWS.md](../WINDOWS.md)。

## 系统要求

使用 apt 或 `.deb` 安装时，这些依赖会自动装上；下面的列表针对发布压缩包和从源码构建。

- 使用 PipeWire 的 Wayland 下 GNOME，`xdg-desktop-portal`（1.21 或更高，需支持应用注册）以及 `xdg-desktop-portal-gnome`。开发和测试环境为 Ubuntu + GNOME 50。
- 剪贴板需要 `wl-clipboard`（`wl-copy` 命令），通知需要 `libnotify`（`notify-send` 命令）。没有 `wl-copy` 时，如果装有 `xclip`，Scrawl 会通过 XWayland 改用它；两者都没有时，会提示该安装什么。
- 在 **X11** 上没有门户：直接截取屏幕，剪贴板需要 `xclip` 而不是 `wl-clipboard`。

```bash
sudo apt install wl-clipboard libnotify-bin   # Wayland
sudo apt install xclip libnotify-bin          # X11
```

## 安装

### 1. 获取程序

**从 apt 仓库安装**（Debian、Ubuntu；更新通过 `apt upgrade` 获取）：

```bash
sudo curl -fsSL https://jexme.github.io/scrawl/scrawl.sources -o /etc/apt/sources.list.d/scrawl.sources
sudo apt update && sudo apt install scrawl
```

软件包提供 amd64 和 arm64 版本（arm64 版本尽力而为；其他架构请从源码构建，见下文）。仓库使用以下密钥签名：`E62C DAF5 A1C9 0DB7 A899 D24D FA43 15C7 154A 9EF0`，该密钥已嵌入 `scrawl.sources`。

在较旧的系统上（Ubuntu 20.04、Debian 11：apt 低于 2.4，无法读取 `.sources` 文件里的密钥），请改用单独的密钥环和单行格式的源（命令和文件名与英文版 README 相同，保持不变）：

```bash
curl -fsSL https://jexme.github.io/scrawl/scrawl-archive-keyring.gpg | sudo tee /usr/share/keyrings/scrawl-archive-keyring.gpg >/dev/null
echo "deb [signed-by=/usr/share/keyrings/scrawl-archive-keyring.gpg] https://jexme.github.io/scrawl stable main" | sudo tee /etc/apt/sources.list.d/scrawl.list
sudo apt update && sudo apt install scrawl
```

密钥也以 ASCII 封装格式发布为 `scrawl-archive-keyring.asc`。

**从 .deb 包安装**（Debian、Ubuntu）：从 [Releases](https://github.com/jexme/scrawl/releases) 页面下载 `scrawl_VERSION_amd64.deb`，然后执行 `sudo apt install ./scrawl_*.deb`。它会装上所需的库，并推荐安装 `wl-clipboard`、`xclip` 和 `libnotify-bin`。快捷键不会由软件包分配（它是按用户设置的）：请在 `scrawl -settings` 中设置。也可以用 `make deb` 自己构建。

**从发布压缩包安装：**从 [Releases](https://github.com/jexme/scrawl/releases) 页面下载 `scrawl-VERSION-linux-amd64.tar.gz` 和 `SHA256SUMS`，然后：

```bash
sha256sum -c --ignore-missing SHA256SUMS   # checks every downloaded file listed in SHA256SUMS
tar xzf scrawl-*-linux-amd64.tar.gz && cd scrawl-*-linux-amd64
./install.sh            # 安装到 ~/.local（系统级安装用 PREFIX=/usr/local）
sudo apt install libpipewire-0.3-0 libwayland-client0 libwayland-egl1 libwayland-cursor0 \
  libxkbcommon-x11-0 libegl1 libx11-6 libxcursor1 wl-clipboard libnotify-bin
```

`wl-clipboard`（剪贴板）和 `libnotify-bin`（通知）运行时同样需要：安装脚本会检查它们，并列出缺少的部分。在 X11 上请用 `xclip` 代替 `wl-clipboard`。

**从源码构建：**

```bash
sudo apt install gcc pkg-config patch libwayland-dev libx11-dev libx11-xcb-dev \
  libxkbcommon-x11-dev libgles2-mesa-dev libegl1-mesa-dev libffi-dev \
  libxcursor-dev libvulkan-dev libpipewire-0.3-dev
git clone https://github.com/jexme/scrawl.git && cd scrawl
make install            # 构建并安装到 ~/.local（系统级安装用 PREFIX=/usr/local）
```

需要 Go（版本见 `go.mod`）。没有 `sudo`？`scripts/fetch-pipewire-headers.sh` 会把 PipeWire 头文件解压到 `.deps/`，Makefile 会自动使用。

请确认 `~/.local/bin` 在 `PATH` 中。

### Windows 10 和 11

**推荐：安装程序。**从 [Releases](https://github.com/jexme/scrawl/releases) 页面下载 `scrawl-VERSION-windows-amd64-setup.exe` 并运行。它只为你的账户安装（不需要管理员权限），会添加开始菜单项，可以在你登录时启动 Scrawl，安装完成后也会启动它。更新时运行更新的安装程序即可：它会先退出旧的 Scrawl。如果某个版本没有安装程序，请使用下面的方式之一。

其他方式：

- **便携 zip。**下载 `scrawl-VERSION-windows-amd64.zip` 和对应的 `.sha256`，解压到一个固定的文件夹（不要直接在压缩包查看器里运行），然后运行 `scrawl.exe`。双击 `scrawl.exe` 会启动通知区域图标。同时还提供单独的 `scrawl-VERSION-windows-amd64.exe`。更新时，请先从图标菜单退出 Scrawl，再替换文件。
- **[Scoop](https://scoop.sh)：**`scoop install https://github.com/jexme/scrawl/releases/download/v0.1.0-rc4/scrawl.json`。这个地址固定指向某一个预发布版本，所以在候选版本之间更新要先执行 `scoop uninstall scrawl`，再用新版本的地址执行 `scoop install`（直到有正式版本或 bucket 为止）。一旦清单可以通过 `https://github.com/jexme/scrawl/releases/latest/download/scrawl.json` 获取（即已有正式版本），`scoop update scrawl` 就可以使用了。

- **图标**在通知区域；在 Windows 11 中它可能藏在时钟旁边的 `^` 箭头后面，把它拖出来就能一直显示。
- **Print 键**是默认快捷键。它常被其他程序占用（Greenshot、ShareX、Lightshot），Windows 本身也可能占用：设置 → 辅助功能 → 键盘 → *使用 Print Screen 按钮打开屏幕截图*：把它关掉。或者在 `scrawl.exe -settings` 中换一个键，例如 `<Control><Alt>a`。如果这个键被占用，图标会告诉你，并持续重试。
- **“Windows 已保护你的电脑”**（SmartScreen）会出现，是因为程序还没有代码签名：点击*更多信息*，然后点击*仍要运行*。如果想先检查文件，可以把 `Get-FileHash` 的结果与 `.sha256` 文件对比，或者运行 `gh attestation verify scrawl-*-setup.exe --repo jexme/scrawl`（构建来自公开的 CI）。杀毒软件可能会把任何新的、未签名的 Go 程序标为可疑；遇到这种情况请提交一个 issue。
- **自启动**是 `scrawl.exe -settings` 中“系统”标签页里的复选框（对应 `HKCU\...\Run` 中的一个值），也可以是安装程序里的一个选项。如果你移动或更新了程序，图标下一次启动时会把这一项改写为新位置（已关闭的自启动保持关闭）。
- **在旧版本运行时启动新版本：**Scrawl 会告诉你正在运行的是哪个版本；请先从图标菜单退出它，再启动新版本。

从源码构建以及尚未实现的部分：[docs/WINDOWS.md](../WINDOWS.md)。

### 2. 首次运行：三个一次性步骤

*在 X11 上不需要步骤 1 和 2（没有门户），只需设置快捷键。*

打开设置窗口（`scrawl -settings`，“系统”标签页）。按钮“设置截图”会一次完成步骤 1 和 2（第一次启动时窗口会自己打开）。也可以使用单独的按钮，或直接执行命令：

1. **选择显示器：**按钮“重新选择显示器”或 `scrawl -setup`。GNOME 会**为每个显示器**弹出一个对话框。每次选择一个*不同的*显示器，保持勾选 **“记住此选择”**，然后确认共享。之后不再有对话框。
2. **授予备用权限：**按钮“授予权限”或 `scrawl -grant-screenshot`。在 PipeWire 偶尔没有返回画面时，Scrawl 可借此改用较慢的 Screenshot portal。
3. **设置快捷键：**在 GNOME 上，发布压缩包的安装脚本（`install.sh`）和 `make install` 会自动分配 `Print`，除非已有其他自定义快捷键使用了它。通过 apt 或 `.deb` 安装之后，或者想更改按键时：使用“启动快捷键”输入框和“分配”按钮（默认 `Print`），或 `scripts/gnome-hotkey.sh [按键]`。它会创建一个名为 “scrawl” 的 GNOME 自定义快捷键（已存在则更新它，所以重复运行不会产生第二个），并且不会覆盖别人已占用的快捷键。如果 `Print` 仍然打开 GNOME 自带的截图工具，请在 *设置 → 键盘 → 查看及自定义快捷键 → 截图* 中关闭它。

### 从 screen-editor 升级

Scrawl 是 screen-editor 的新名称。在 `~/.config/scrawl/config.json` 出现之前，设置会从 `~/.config/screen-editor/config.json` 读取（旧文件保持不变）。门户把显示器选择和权限绑定在应用 ID 上，而 ID 已更改，所以请把步骤 1 到 3 再做一遍，并把快捷键指向 `scrawl`。上次的区域和合集会重新开始。

## 使用方法

1. 按下快捷键，屏幕冻结并变暗。
2. 拖动框选区域（可在任意显示器上）。下方会出现工具栏。
3. 需要的话添加标注：选择工具和颜色，在区域上拖动绘制。
4. 按 `Enter`。图片已在剪贴板中，并保存在 `~/Pictures/Screenshots`，可以随处粘贴。

任何时候按 `Esc` 都可取消。

### 快捷键

| 按键 | 作用 |
| --- | --- |
| `Enter` | 复制到剪贴板**并**保存为文件（具体行为可设置） |
| `Ctrl+C` / `Ctrl+S` | 只复制 / 只保存 |
| `Ctrl+Z` | 撤销上一个标注 |
| `Esc` | 取消（会先关闭已打开的菜单；输入文字时只放弃文字） |
| `Ctrl+A` | 选中该显示器的整个屏幕（不可重新映射） |
| `R` | 恢复上一次截图的区域 |
| `V` `P` `L` `W` `B` `M` `U` `T` `N` | 工具：选择、画笔、直线、箭头、方框、荧光笔、模糊、文字、编号标记（已选中区域时） |
| `+` / `-` | 当前工具的粗细、文字大小或模糊强度（与鼠标滚轮相同） |
| `A` | 把截图加入合集 |
| `G` | 把合集（和当前区域）拼成一张图 |
| `D` | 把合集保存为多个独立文件，并把文件列表放入剪贴板 |
| 鼠标滚轮 | 当前工具的粗细、文字大小或模糊强度 |
| 绘制时按 `Shift` | 画笔和荧光笔：直线；直线和箭头：吸附到 45°；方框和模糊：正方形。选择区域时按 `Shift`：正方形 |

所有按键都可以在设置窗口（“按键”标签页）中更改。

### 工具栏

主行从左到右：选择、画笔、直线、箭头、方框、荧光笔、模糊、文字、编号 · 撤销 · 复制 · 保存 · `⋯` 菜单（设置，即关闭浮层并打开设置窗口；以及加入合集）· 关闭。

使用绘图工具时，下面会出现第二行：八种颜色、一个用于自定义颜色的彩虹按钮（打开调色板，颜色会被记住）以及 `−` 粗细 `+`。“选择”工具两者都没有，所以这一行会隐藏。

有对应按键的按钮会在图标下方标出该按键（按键来自设置）。把光标在按钮上停留片刻，就能看到它的名称和按键。快捷键在任何键盘布局下都有效（例如俄文布局），依据的是按键的位置。

### 前后对比合集

先截第一张并按 `A`。再截下一张时，上一次的区域已被选中，所以可以对同一处再拍一次。按 `G` 把截图拼成一张图（会保存并复制），按 `D` 则得到独立文件。屏幕底部的小面板显示合集内容，并用按钮提供相同的操作。

## 托盘图标与自启动

`scrawl -tray` 会在通知区域保留一个图标：左键单击截图，菜单中有“截图”、“设置”和“退出”。要在登录时启动，请打开 *设置 → 系统 → 自启动*。在 GNOME 上，图标需要 AppIndicator 扩展（Ubuntu 默认已启用）。

## 设置

![设置窗口](../images/settings-en.png)

可通过工具栏齿轮、`scrawl -settings` 或应用菜单打开。更改会立即保存到 `~/.config/scrawl/config.json`，这是一个普通的 JSON 文件，也可以手动编辑。缺失或无效的值会回退到默认值。每个键的说明见[设置参考](../SETTINGS.md)。

## 文件与隐私

- 图片保存在 `~/Pictures/Screenshots`（可在设置中更改），文件名形如 `scrawl-20260131-142501.png`。
- 设置：`~/.config/scrawl/`。显示器令牌、上次区域和合集：`~/.cache/scrawl/`。
- Scrawl 不会自己联网。程序里唯一的网络请求是可选的更新检查（默认关闭）：它向 GitHub 查询最新版本，除常规 HTTP 头外只发送程序版本号。
- 截图不会离开你的电脑，除非你自己把它粘贴到别处。

## 命令行

| 参数 | 作用 |
| --- | --- |
| `-settings` | 打开设置窗口 |
| `-setup` | 重新选择显示器（GNOME 对话框） |
| `-grant-screenshot` | 授予备用截图权限并退出 |
| `-tray` | 在通知区域保留图标 |
| `-fresh` | 不提供上一次截图的区域 |
| `-v` | 打印耗时 |
| `-version` | 打印版本并退出 |
| `-exit-after DURATION` | 在给定时间后正常退出，用于测试（例如 `-exit-after 3s`） |

## 常见问题

- **按快捷键没有反应。** 可能是 `xdg-desktop-portal` 卡住了。重启它：`systemctl --user restart xdg-desktop-portal xdg-desktop-portal-gnome`。
- **GNOME 每次截图都问要共享哪个屏幕。** 没有勾选“记住此选择”；请重做步骤 1。
- **新增或更换了显示器。** 重新选择显示器（步骤 1）。
- **浮层只在一个显示器上打开。** 同上：每个显示器都要完成步骤 1。
- **提示门户拒绝了请求。** 授予权限（步骤 2）。
- **提示剪贴板错误，或粘贴后没有内容。** 检查是否安装了 `wl-copy`（软件包 `wl-clipboard`）。
- **没有通知。** 安装 `libnotify-bin`（`notify-send` 命令），并确认设置中已开启通知。
- **没有托盘图标。** 启用 GNOME 的 AppIndicator 扩展。
- **`Print` 打开了 GNOME 的截图工具。** 在键盘快捷键中关闭它（见步骤 3）。

## 卸载

**apt 或 `.deb`：**

```bash
sudo apt remove scrawl     # or: sudo apt purge scrawl (the package keeps no files outside /usr, so they are the same)
sudo rm /etc/apt/sources.list.d/scrawl.sources   # optional: forget the repository (scrawl.list for the old form)
```

apt 无法处理你的主目录，所以一些按用户保存的内容会留下来：GNOME 快捷键、你开启过的自启动项，以及你的设置。可以用 Scrawl 的 `gnome-hotkey.sh` 删除快捷键（它位于仓库的 `scripts/` 中，也在发布压缩包里），或者在 *设置 → 键盘 → 查看及自定义快捷键 → 自定义快捷键* 中手动删除；其余的都是普通文件：

```bash
sh gnome-hotkey.sh --remove
rm -f ~/.config/autostart/io.github.jexme.Scrawl.desktop
rm -rf ~/.config/scrawl ~/.cache/scrawl     # optional: settings and cache
```

**发布压缩包或 `make install`：**在解压后的压缩包中运行 `./uninstall.sh`（在源码目录中用 `make uninstall`）。它会删除可执行文件、`.desktop` 文件、GNOME 快捷键和自启动项。加上 `--purge`（`make uninstall PURGE=1`）还会删除设置（`~/.config/scrawl`）和缓存（`~/.cache/scrawl`）。如果安装时改过 `PREFIX`，卸载时请传入相同的值。你在 `~/Pictures/Screenshots` 中的截图不会被动。

## 工作原理

Scrawl 通过 `xdg-desktop-portal` 的 ScreenCast 接口，用自带的小型 PipeWire 客户端（`internal/pwcapture`）截取屏幕。为每个显示器保存的恢复令牌省去了对话框。合成器（Mutter）只在显示器重绘时才发送画面，在我们的测试中全屏窗口之上（直接扫描输出）它一帧也没发；如果 250 毫秒内没有画面，Scrawl 会改用 Screenshot portal，速度较慢（约 0.5–1 秒）。浮层是全屏的 [Gio](https://gioui.org) 窗口，每个显示器一个；Gio 带有补丁（`third_party/gio.patch`），可把窗口放到指定的 Wayland 输出上。最终图片由屏幕上绘制它的同一份代码渲染，所以保存的图片与你看到的一致。

## 构建与参与

```bash
make build    # bin/scrawl；首次运行会下载并给 Gio 打补丁
make test     # 单元测试，包括会话 D-Bus 上的托盘图标
make dist     # 在 dist/ 中生成发布压缩包和校验和
make deb      # .deb package for Debian and Ubuntu in dist/
make notices  # 重新生成 THIRD_PARTY_NOTICES.md
```

界面文字放在 `internal/i18n/`（每种语言一个文件，`lang_ru.go`、`lang_de.go` 等；代码中的源文本是英文）；`go test ./...` 会检查每条文字在每种语言中都有翻译。如何报告问题和提交修改：[CONTRIBUTING.md](../../CONTRIBUTING.md)。计划与想法：[docs/PLAN.md](../PLAN.md)。

## 许可证

MIT，见 [LICENSE](../../LICENSE)。第三方组件：[THIRD_PARTY_NOTICES.md](../../THIRD_PARTY_NOTICES.md)（图标形状来自 [Tabler Icons](https://tabler.io/icons)，MIT）。
