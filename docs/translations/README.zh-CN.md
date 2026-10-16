# Scrawl

**面向 Wayland 下 GNOME 的快速截图与标注工具。** 按下快捷键，在任意显示器上框选区域，画箭头、方框和文字，把隐私内容打码，结果立刻进入剪贴板并保存为文件。

[English](../../README.md) · [Русский](../../README.ru.md) · **简体中文** · [Español](README.es.md) · [Português (Brasil)](README.pt-BR.md) · [日本語](README.ja.md) · [Deutsch](README.de.md) · [Français](README.fr.md) · [한국어](README.ko.md)

![Scrawl 浮层：选中的区域上有方框、箭头、文字说明、荧光笔和模糊处理，下方是工具栏](../images/overlay.png)

*图片是在虚构桌面上真实渲染的浮层。*

> 程序界面目前有英语和俄语两种语言；下文提到的按钮名称均为英文界面中的名称。

## 为什么选择 Scrawl

在 Wayland 下的 GNOME 中，应用不能直接截取屏幕，只能请求桌面门户（portal），而门户**每次截图都会问“要共享哪个屏幕？”**。Scrawl 对每个显示器只问一次并记住答案，之后截图只需按一个键：冻结的屏幕约 150 毫秒内出现。

## 功能

**截图**
- 同时支持所有显示器：浮层会在每个显示器上打开，可在任意一个上框选。
- 虚线框选。拖动角点调整大小；使用“选择”工具时，可拖动边缘调整大小，或拖动内部移动区域。
- 记住上一次的区域，按一个键（`R`）即可恢复。
- 可选：把鼠标指针也截进图片。

**标注**
- 画笔、直线、箭头、方框、荧光笔、**模糊**和文字；八种颜色。
- 鼠标滚轮（或 `−` / `+` 按钮）调整粗细、文字大小或模糊强度。
- `Ctrl+Z` 撤销。

**保存与分享**
- `Enter` 把图片复制到剪贴板，并把 PNG 保存到 `~/Pictures/Screenshots`。`Ctrl+C` 只复制，`Ctrl+S` 只保存。也支持 JPEG。
- **前后对比合集：**用 `A` 把截图先放进合集，再拼成一张带说明文字的图（两张时为“前/后”，更多时为“步骤 1…”），可左右并排或上下排列；也可以把合集作为多个独立文件发送，粘贴到聊天里就是附件。

**个性化**
- 几乎所有选项都在设置窗口里：快捷键、颜色、默认工具、选区外观、文件名和格式、语言（英语、俄语）。
- 带自启动的托盘图标、设置快捷键的按钮，以及可选的更新检查（默认关闭）。

## 状态

个人项目，正在积极开发。**尚未发布正式版本**，目前需要从源码构建（几分钟即可）。仅支持 Wayland 下的 GNOME（Linux）。Windows 版本可以编译，但还没有实现屏幕捕获，详见 [docs/WINDOWS.md](../WINDOWS.md)。

## 系统要求

- 使用 PipeWire 的 Wayland 下 GNOME，`xdg-desktop-portal`（1.21 或更高，需支持应用注册）以及 `xdg-desktop-portal-gnome`。开发和测试环境为 Ubuntu + GNOME 50。
- 剪贴板需要 `wl-clipboard`（`wl-copy` 命令），通知需要 `libnotify`（`notify-send` 命令）：

```bash
sudo apt install wl-clipboard libnotify-bin
```

## 安装

### 1. 获取程序

**从源码构建（现在可用）：**

```bash
sudo apt install gcc pkg-config patch libwayland-dev libx11-dev libx11-xcb-dev \
  libxkbcommon-x11-dev libgles2-mesa-dev libegl1-mesa-dev libffi-dev \
  libxcursor-dev libvulkan-dev libpipewire-0.3-dev
git clone https://github.com/jexme/scrawl.git && cd scrawl
make install            # 构建并安装到 ~/.local（系统级安装用 PREFIX=/usr/local）
```

需要 Go（版本见 `go.mod`）。没有 `sudo`？`scripts/fetch-pipewire-headers.sh` 会把 PipeWire 头文件解压到 `.deps/`，Makefile 会自动使用。

**从发布压缩包安装（发布版本后）：** 从 [Releases](https://github.com/jexme/scrawl/releases) 页面下载 `scrawl-VERSION-linux-amd64.tar.gz` 和 `SHA256SUMS`，然后：

```bash
sha256sum -c --ignore-missing SHA256SUMS
tar xzf scrawl-*-linux-amd64.tar.gz && cd scrawl-*-linux-amd64
./install.sh            # 安装到 ~/.local（系统级安装用 PREFIX=/usr/local）
sudo apt install libpipewire-0.3-0 libwayland-client0 libwayland-egl1 libwayland-cursor0 \
  libxkbcommon-x11-0 libegl1 libx11-6 libxcursor1
```

请确认 `~/.local/bin` 在 `PATH` 中。

### 2. 首次运行：三个一次性步骤

打开设置窗口（`scrawl -settings`）点击按钮，或直接执行命令：

1. **选择显示器：**按钮 “Pick monitors again” 或 `scrawl -setup`。GNOME 会**为每个显示器**弹出一个对话框。每次选择一个*不同的*显示器，保持勾选 **“记住此选择”**，然后确认共享。之后不再有对话框。
2. **授予备用权限：**按钮 “Grant permission” 或 `scrawl -grant-screenshot`。在 PipeWire 偶尔没有返回画面时，Scrawl 可借此改用较慢的 Screenshot portal。
3. **设置快捷键：**“Launch hotkey” 输入框和 “Assign” 按钮（默认 `Print`），或 `scripts/gnome-hotkey.sh [按键]`。它会创建一个 GNOME 自定义快捷键，不会覆盖已被占用的组合。如果 `Print` 仍然打开 GNOME 自带的截图工具，请在 *设置 → 键盘 → 查看及自定义快捷键 → 截图* 中关闭它。

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
| `Esc` | 取消 |
| `R` | 恢复上一次截图的区域 |
| `A` | 把截图加入合集 |
| `G` | 把合集（和当前区域）拼成一张图 |
| `D` | 把合集保存为多个独立文件，并把文件列表放入剪贴板 |
| 鼠标滚轮 | 当前工具的粗细、文字大小或模糊强度 |

所有按键都可以在设置窗口（“Overlay keys”）中更改。

### 工具栏

从左到右：选择、画笔、直线、箭头、方框、荧光笔、模糊、文字 · 八种颜色 · `−` 粗细 `+` · 设置（关闭浮层并打开设置窗口）· 加入合集 · 撤销 · 复制 · 保存 · 关闭。

### 前后对比合集

先截第一张并按 `A`。再截下一张时，上一次的区域已被选中，所以可以对同一处再拍一次。按 `G` 把截图拼成一张图（会保存并复制），按 `D` 则得到独立文件。屏幕底部的小面板显示合集内容，并用按钮提供相同的操作。

## 托盘图标与自启动

`scrawl -tray` 会在通知区域保留一个图标：左键单击截图，菜单中有 “Screenshot”、“Settings” 和 “Quit”。要在登录时启动，请打开 *Settings → Startup → Autostart*。在 GNOME 上，图标需要 AppIndicator 扩展（Ubuntu 默认已启用）。

## 设置

![设置窗口](../images/settings-en.png)

可通过工具栏齿轮、`scrawl -settings` 或应用菜单打开。更改会立即保存到 `~/.config/scrawl/config.json`，这是一个普通的 JSON 文件，也可以手动编辑。缺失或无效的值会回退到默认值。每个键的说明见[设置参考](../SETTINGS.md)（英文或[俄文](../SETTINGS.ru.md)）。

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

## 常见问题

- **按快捷键没有反应。** 可能是 `xdg-desktop-portal` 卡住了。重启它：`systemctl --user restart xdg-desktop-portal xdg-desktop-portal-gnome`。
- **GNOME 每次截图都问要共享哪个屏幕。** 没有勾选“记住此选择”；请重做步骤 1。
- **新增或更换了显示器。** 重新选择显示器（步骤 1）。
- **浮层只在一个显示器上打开。** 同上：每个显示器都要完成步骤 1。
- **提示门户拒绝了请求。** 授予权限（步骤 2）。
- **粘贴到某些应用里没有内容。** 检查是否安装了 `wl-copy`（软件包 `wl-clipboard`）。
- **没有通知。** 安装 `libnotify-bin`（`notify-send` 命令），并确认设置中已开启通知。
- **没有托盘图标。** 启用 GNOME 的 AppIndicator 扩展。
- **`Print` 打开了 GNOME 的截图工具。** 在键盘快捷键中关闭它（见步骤 3）。

## 工作原理

Scrawl 通过 `xdg-desktop-portal` 的 ScreenCast 接口，用自带的小型 PipeWire 客户端（`internal/pwcapture`）截取屏幕。为每个显示器保存的恢复令牌省去了对话框。合成器（Mutter）只在显示器重绘时才发送画面，在我们的测试中全屏窗口之上（直接扫描输出）它一帧也没发；如果 250 毫秒内没有画面，Scrawl 会改用 Screenshot portal，速度较慢（约 0.5–1 秒）。浮层是全屏的 [Gio](https://gioui.org) 窗口，每个显示器一个；Gio 带有补丁（`third_party/gio.patch`），可把窗口放到指定的 Wayland 输出上。最终图片由屏幕上绘制它的同一份代码渲染，所以保存的图片与你看到的一致。

## 构建与参与

```bash
make build    # bin/scrawl；首次运行会下载并给 Gio 打补丁
make test     # 单元测试，包括会话 D-Bus 上的托盘图标
make dist     # 在 dist/ 中生成发布压缩包和校验和
make notices  # 重新生成 THIRD_PARTY_NOTICES.md
```

界面文字在 `internal/i18n/en.go`（英文；代码中的源文本为俄文）；`go test ./...` 会检查每条文字都有翻译。计划与想法：[docs/PLAN.md](../PLAN.md)（俄文）。

## 许可证

MIT，见 [LICENSE](../../LICENSE)。第三方组件：[THIRD_PARTY_NOTICES.md](../../THIRD_PARTY_NOTICES.md)（图标形状来自 [Tabler Icons](https://tabler.io/icons)，MIT）。
