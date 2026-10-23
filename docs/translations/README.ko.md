# Scrawl

**Wayland의 GNOME을 위한 빠른 주석 스크린샷 도구.** 키를 누르고, 아무 모니터에서나 영역을 선택하고, 화살표·상자·텍스트를 그리고, 민감한 정보는 흐리게 처리하면, 결과가 곧바로 클립보드에 들어가고 파일로도 저장됩니다.

[English](../../README.md) · [Русский](README.ru.md) · [简体中文](README.zh-CN.md) · [Español](README.es.md) · [Português (Brasil)](README.pt-BR.md) · [日本語](README.ja.md) · [Deutsch](README.de.md) · [Français](README.fr.md) · **한국어**

![Scrawl 오버레이: 상자, 화살표, 글자, 형광펜 선, 흐림 처리가 있는 선택 영역과 그 아래의 도구 모음](../images/overlay.png)

*이 그림은 가상의 데스크톱 위에 실제 오버레이를 렌더링한 것입니다.*

> 화면은 영어, 러시아어, 중국어(간체), 스페인어, 포르투갈어, 일본어, 독일어, 프랑스어, 한국어로 제공됩니다. 아래의 버튼 이름은 한국어 화면 기준입니다.

## 왜 Scrawl인가

Wayland의 GNOME에서는 앱이 화면을 직접 캡처할 수 없고 데스크톱 포털에 요청해야 하며, 포털은 **촬영할 때마다 “어느 화면을 공유할까요?”라고 묻습니다**. Scrawl은 모니터마다 한 번만 묻고 답을 기억하므로, 그다음부터는 키 한 번이면 스크린샷이 됩니다. 정지된 화면이 약 150ms 안에 나타납니다.

## 빠른 시작

Ubuntu나 Debian(Wayland 또는 X11)에서는 세 단계면 됩니다.

**1. 설치:** 서명된 apt 저장소를 사용합니다(이후 업데이트는 `apt upgrade`로 받습니다):

```bash
sudo curl -fsSL https://jexme.github.io/scrawl/scrawl.sources -o /etc/apt/sources.list.d/scrawl.sources
sudo apt update && sudo apt install scrawl
```

서명 키가 `.sources` 파일 안에 들어 있으므로 키링을 따로 관리할 필요가 없습니다. 패키지는 **amd64**와 **arm64**용이 제공됩니다(arm64 빌드는 최선을 다하는 방식입니다). 다른 아키텍처에서는 [소스에서 빌드](#설치)하세요.

**2. 한 번만 설정:** 애플리케이션 메뉴에서 Scrawl을 실행합니다(또는 `scrawl`을 실행). 처음 실행하면 “시스템” 탭에 짧은 안내가 열립니다(언제든 `scrawl -settings`로 열 수 있습니다). Wayland에서는 “캡처 설정”을 누릅니다(GNOME이 모니터마다 한 번씩 묻습니다. 대화상자마다 다른 모니터를 고르고 “이 선택 기억”에 체크하세요). 그다음 단축키(기본값 `Print`)를 입력하고 “지정”을 누릅니다. GNOME의 기본 스크린샷 도구도 `Print`를 사용합니다. “키 해제”를 누르면 그 기본 단축키가 꺼집니다(그 키 하나만 꺼지며, 되돌릴 수 있습니다). X11에서는 단축키만 있으면 됩니다.

**3. 스크린샷 찍기:** 단축키를 누르고, 영역을 드래그하고, `Enter`를 누릅니다. 이미지가 클립보드와 `~/Pictures/Screenshots`에 들어갑니다.

다른 시스템: [Windows](#windows-10-및-11), 다른 Linux 배포판(릴리스 압축 파일 또는 소스 빌드)은 아래의 [설치](#설치)를 참고하세요.

## 기능

**캡처**
- 모든 모니터에서 동시에 동작합니다. 오버레이가 각 모니터에 열리고 어느 모니터에서든 선택할 수 있습니다.
- 점선 테두리로 영역을 선택합니다. 모서리를 끌어 크기를 바꾸고, “선택” 도구에서는 가장자리를 끌어 크기를 바꾸거나 안쪽을 끌어 이동할 수 있습니다.
- 테두리를 끄는 동안 커서 옆의 돋보기가 픽셀을 6배로 확대해서 보여 주고, 테두리의 가장자리를 픽셀 격자에 맞춰 주므로 정확하게 맞출 수 있습니다(설정에서 끌 수 있습니다).
- 마지막 영역을 기억하고 키 하나(`R`)로 불러옵니다.
- 선택 사항으로 마우스 커서도 함께 캡처할 수 있습니다(지금은 Wayland만 해당).

**주석**
- 펜, 직선, 화살표, 상자, 형광펜, **흐림**, 텍스트, **번호 스탬프**(클릭하면 1, 2, 3… 숫자가 든 원이 놓입니다. flameshot과 같습니다). 색은 8가지입니다.
- 마우스 휠(또는 `−` / `+` 버튼)로 두께, 글자 크기, 흐림 강도를 바꿉니다.
- **흐림**은 6단계입니다(휠 또는 `−` / `+`): 1–3은 부드러운 가우시안 흐림, 4–6은 사각형이 점점 커지는 모자이크입니다. 모자이크의 칸은 영역의 모서리에서 시작하며, 단계를 바꿀 때 나타나는 알림에 모드 이름이 표시됩니다.
- 그리는 동안 **Shift**를 누르고 있으면 펜과 형광펜은 직선이 되고, 직선과 화살표는 45°에 맞춰지며, 상자와 흐림은 정사각형이 됩니다.
- `Ctrl+Z`로 실행 취소합니다.

**저장과 공유**
- `Enter`는 이미지를 클립보드에 복사하고 PNG를 `~/Pictures/Screenshots`에 저장합니다. `Ctrl+C`는 복사만, `Ctrl+S`는 저장만 합니다. JPEG도 지원합니다.
- 선택 영역 안을 **더블 클릭**하면 즉시 복사됩니다(설정의 “일반” 탭, “동작” 섹션에서 복사하고 저장하기, 또는 아무것도 하지 않기로 바꿀 수도 있습니다).
- **전/후 모음:** `A`로 스크린샷을 따로 모아 두었다가, 캡션(“전”, “후”, 세 장 이상이면 “단계 1…”)이 붙은 한 장의 그림으로 나란히 또는 위아래로 이어 붙입니다. 모음을 개별 파일로 보낼 수도 있으며, 채팅에는 첨부 파일로 붙여 넣어집니다.

**내 취향대로**
- 거의 모든 것을 설정 창에서 바꿀 수 있습니다: 키, 색, 기본 도구, 선택 영역의 모양, 파일 이름과 형식, 화면 언어(9개 언어: 영어, 러시아어, 중국어, 스페인어, 포르투갈어, 일본어, 독일어, 프랑스어, 한국어).
- 자동 시작이 있는 트레이 아이콘, 단축키 지정 버튼, 선택 사항인 업데이트 확인(기본값은 꺼짐).

## 상태

활발히 개발 중인 개인 프로젝트이며, 현재 버전은 사전 릴리스입니다(`v0.1.0-rc4`). 바로 쓸 수 있는 패키지는 [Releases](https://github.com/jexme/scrawl/releases) 페이지에 있으며, 소스에서 빌드할 수도 있습니다(몇 분이면 됩니다). Linux: 주요 대상은 Wayland의 GNOME입니다. **X11** 세션(Xorg의 GNOME, KDE, XFCE 등)에서도 동작하지만, 이 지원은 새것이고 지금까지 가상 X 서버에서만 테스트했습니다. 실제 기기에서의 제보를 환영합니다. **Windows 10과 11**에서도 동작합니다(캡처, 주석, 클립보드, 트레이 아이콘, 전역 단축키). 이 포팅은 새것이며 100% 배율에서만 시험해 보았습니다. 빌드 방법과 아직 없는 기능은 [docs/WINDOWS.md](../WINDOWS.md)를 참고하세요.

## 요구 사항

apt나 `.deb`로 설치하면 이 항목들이 자동으로 설치됩니다. 이 목록은 릴리스 압축 파일과 소스 빌드를 위한 것입니다.

- PipeWire를 쓰는 Wayland의 GNOME, `xdg-desktop-portal`(1.21 이상, 앱 등록 지원 필요), `xdg-desktop-portal-gnome`. 개발과 테스트는 Ubuntu와 GNOME 50에서 했습니다.
- 클립보드용 `wl-clipboard`(`wl-copy` 명령)와 알림용 `libnotify`(`notify-send` 명령). `wl-copy`가 없으면 `xclip`이 있을 때 Scrawl이 XWayland를 통해 `xclip`을 대신 사용하며, 둘 다 없으면 무엇을 설치해야 하는지 안내가 나옵니다.
- **X11**에는 포털이 없어 화면을 직접 캡처하며, 클립보드에는 `wl-clipboard` 대신 `xclip`이 필요합니다.

```bash
sudo apt install wl-clipboard libnotify-bin   # Wayland
sudo apt install xclip libnotify-bin          # X11
```

## 설치

### 1. 프로그램 받기

**apt 저장소에서** (Debian, Ubuntu; 업데이트는 `apt upgrade`로 받습니다):

```bash
sudo curl -fsSL https://jexme.github.io/scrawl/scrawl.sources -o /etc/apt/sources.list.d/scrawl.sources
sudo apt update && sudo apt install scrawl
```

패키지는 amd64와 arm64용이 제공됩니다(arm64 빌드는 최선을 다하는 방식이며, 다른 아키텍처에서는 아래를 참고해 소스에서 빌드하세요). 저장소는 다음 키로 서명되어 있습니다: `E62C DAF5 A1C9 0DB7 A899 D24D FA43 15C7 154A 9EF0`. 이 키는 `scrawl.sources`에 들어 있습니다.

오래된 시스템(Ubuntu 20.04, Debian 11: apt 2.4 미만은 `.sources` 파일 안의 키를 읽지 못합니다)에서는 대신 별도의 키링과 한 줄짜리 소스를 사용하세요(명령과 파일 이름은 영어 README와 같습니다):

```bash
curl -fsSL https://jexme.github.io/scrawl/scrawl-archive-keyring.gpg | sudo tee /usr/share/keyrings/scrawl-archive-keyring.gpg >/dev/null
echo "deb [signed-by=/usr/share/keyrings/scrawl-archive-keyring.gpg] https://jexme.github.io/scrawl stable main" | sudo tee /etc/apt/sources.list.d/scrawl.list
sudo apt update && sudo apt install scrawl
```

키는 ASCII 아머 형식의 `scrawl-archive-keyring.asc`로도 공개되어 있습니다.

**.deb 패키지에서** (Debian, Ubuntu): [Releases](https://github.com/jexme/scrawl/releases) 페이지에서 `scrawl_VERSION_amd64.deb`를 받은 다음 `sudo apt install ./scrawl_*.deb`를 실행합니다. 필요한 라이브러리가 함께 설치되고 `wl-clipboard`, `xclip`, `libnotify-bin`이 권장 패키지로 추가됩니다. 단축키는 패키지가 지정하지 않습니다(사용자마다 따로 설정하기 때문입니다). `scrawl -settings`에서 지정하세요. 직접 빌드하려면 `make deb`를 사용합니다.

**릴리스 압축 파일에서:** [Releases](https://github.com/jexme/scrawl/releases) 페이지에서 `scrawl-VERSION-linux-amd64.tar.gz`와 `SHA256SUMS`를 받은 다음:

```bash
sha256sum -c --ignore-missing SHA256SUMS   # checks every downloaded file listed in SHA256SUMS
tar xzf scrawl-*-linux-amd64.tar.gz && cd scrawl-*-linux-amd64
./install.sh            # ~/.local에 설치 (시스템 전체에는 PREFIX=/usr/local)
sudo apt install libpipewire-0.3-0 libwayland-client0 libwayland-egl1 libwayland-cursor0 \
  libxkbcommon-x11-0 libegl1 libx11-6 libxcursor1 wl-clipboard libnotify-bin
```

`wl-clipboard`(클립보드)와 `libnotify-bin`(알림)은 실행할 때도 필요합니다. 설치 스크립트가 이를 확인하고 빠진 것을 알려 줍니다. X11에서는 `wl-clipboard` 대신 `xclip`을 사용하세요.

**소스에서:**

```bash
sudo apt install gcc pkg-config patch libwayland-dev libx11-dev libx11-xcb-dev \
  libxkbcommon-x11-dev libgles2-mesa-dev libegl1-mesa-dev libffi-dev \
  libxcursor-dev libvulkan-dev libpipewire-0.3-dev
git clone https://github.com/jexme/scrawl.git && cd scrawl
make install            # 빌드 후 ~/.local에 설치 (시스템 전체에는 PREFIX=/usr/local)
```

Go(`go.mod`에 적힌 버전)가 필요합니다. `sudo`가 없다면 `scripts/fetch-pipewire-headers.sh`가 PipeWire 헤더를 `.deps/`에 풀어 주고 Makefile이 알아서 사용합니다.

`~/.local/bin`이 `PATH`에 들어 있는지 확인하세요.

### Windows 10 및 11

**권장: 설치 프로그램.** [Releases](https://github.com/jexme/scrawl/releases) 페이지에서 `scrawl-VERSION-windows-amd64-setup.exe`를 받아 실행하세요. 내 계정에만 설치되고(관리자 권한이 필요 없습니다) 시작 메뉴 항목이 추가되며, 로그인할 때 Scrawl을 시작하도록 할 수 있고, 설치가 끝나면 Scrawl이 시작됩니다. 업데이트하려면 더 새로운 설치 프로그램을 실행하세요. 먼저 이전 Scrawl을 종료해 줍니다. 릴리스에 설치 파일이 없다면 아래 방법 중 하나를 사용하세요.

다른 방법:

- **휴대용 zip.** `scrawl-VERSION-windows-amd64.zip`과 해당 `.sha256`을 받아 고정된 폴더에 풀고(압축 파일 뷰어에서 바로 실행하지 마세요) `scrawl.exe`를 실행합니다. `scrawl.exe`를 더블 클릭하면 알림 영역 아이콘이 시작됩니다. 단독 파일인 `scrawl-VERSION-windows-amd64.exe`도 공개됩니다. 업데이트하려면 아이콘 메뉴에서 Scrawl을 종료한 다음 파일을 교체하세요.
- **[Scoop](https://scoop.sh):** `scoop install https://github.com/jexme/scrawl/releases/download/v0.1.0-rc4/scrawl.json`. 이 주소는 사전 릴리스 하나에 고정되어 있으므로, 릴리스 후보 사이에는 `scoop uninstall scrawl`을 실행한 뒤 새 버전의 URL로 `scoop install`을 실행해 업데이트합니다(정식 릴리스나 bucket이 생길 때까지). 매니페스트를 `https://github.com/jexme/scrawl/releases/latest/download/scrawl.json`에서 받을 수 있게 되면(정식 릴리스가 생기면) `scoop update scrawl`이 동작합니다.

- **아이콘**은 알림 영역에 있습니다. Windows 11에서는 시계 옆의 `^` 화살표 뒤에 숨을 수 있으니, 밖으로 끌어내면 계속 보입니다.
- **Print 키**가 기본 단축키입니다. 다른 프로그램(Greenshot, ShareX, Lightshot)이 이 키를 쥐고 있는 경우가 많습니다. Windows 자체가 쥐고 있을 수도 있습니다: 설정 → 접근성 → 키보드 → *Print Screen 단추를 사용하여 화면 캡처 열기*: 이것을 끄세요. 또는 `scrawl.exe -settings`에서 다른 키(예: `<Control><Alt>a`)를 고르세요. 키가 이미 사용 중이면 아이콘이 알려 주고 계속 다시 시도합니다.
- **“Windows의 PC 보호”**(SmartScreen) 화면은 프로그램에 아직 코드 서명이 없어서 나타납니다: *추가 정보*를 클릭한 다음 *실행*을 클릭하세요. 먼저 파일을 확인하려면 `Get-FileHash` 결과를 `.sha256` 파일과 비교하거나, `gh attestation verify scrawl-*-setup.exe --repo jexme/scrawl`을 실행하세요(빌드는 공개 CI에서 만들어집니다). 백신 프로그램은 서명되지 않은 새로운 Go 프로그램을 경고할 수 있습니다. 그런 경우에는 이슈로 알려 주세요.
- **자동 시작**은 `scrawl.exe -settings`의 “시스템” 탭에 있는 체크박스(`HKCU\...\Run`의 값)이거나 설치 프로그램의 체크박스입니다. 프로그램을 옮기거나 업데이트하면 다음에 아이콘이 시작될 때 항목이 새 위치로 다시 기록됩니다(꺼 둔 자동 시작은 꺼진 채로 유지됩니다).
- **이전 버전이 실행 중일 때 새 버전을 시작하는 경우:** Scrawl이 어느 버전이 실행 중인지 알려 줍니다. 먼저 아이콘 메뉴에서 종료한 다음 새 버전을 시작하세요.

소스에서 빌드하는 방법과 아직 없는 기능: [docs/WINDOWS.md](../WINDOWS.md).

### 2. 처음 실행: 한 번만 하는 세 단계

*X11에서는 1, 2단계가 필요 없고(포털이 없습니다) 단축키만 설정하면 됩니다.*

설정 창(`scrawl -settings`, “시스템” 탭)을 엽니다. “캡처 설정” 버튼이 1단계와 2단계를 한 번에 해 줍니다(맨 처음 실행할 때는 창이 저절로 열립니다). 또는 개별 버튼을 쓰거나 명령을 실행하세요:

1. **모니터 선택:** 버튼 “모니터 다시 선택” 또는 `scrawl -setup`. GNOME이 **모니터마다** 대화상자를 보여 줍니다. 매번 *다른* 모니터를 고르고 **“이 선택 기억”**에 체크한 채 공유를 허용하세요. 그 뒤로는 대화상자가 나오지 않습니다.
2. **예비 권한 부여:** 버튼 “권한 부여” 또는 `scrawl -grant-screenshot`. PipeWire가 화면을 주지 못하는 드문 경우에 Scrawl이 더 느린 Screenshot 포털을 쓸 수 있게 해 줍니다.
3. **단축키 지정:** GNOME에서는 압축 파일의 설치 스크립트(`install.sh`)와 `make install`이, 다른 사용자 지정 단축키가 이미 쓰고 있지 않다면 `Print`를 알아서 지정합니다. apt나 `.deb`로 설치한 뒤, 또는 키를 바꾸고 싶을 때는 “실행 단축키” 입력란과 “지정” 버튼(기본값 `Print`), 또는 `scripts/gnome-hotkey.sh [키]`를 사용하세요. “scrawl”이라는 이름의 GNOME 사용자 지정 단축키를 만들며(이미 있으면 갱신하므로 다시 실행해도 두 번째 것이 생기지 않습니다), 다른 사람이 이미 쓰고 있는 단축키는 덮어쓰지 않습니다. `Print`가 여전히 GNOME 기본 스크린샷 도구를 연다면 *설정 → 키보드 → 바로 가기 보기 및 사용자 지정 → 스크린샷*에서 끄세요.

### screen-editor에서 업그레이드

Scrawl은 screen-editor의 새 이름입니다. `~/.config/scrawl/config.json`이 생기기 전까지는 `~/.config/screen-editor/config.json`에서 설정을 읽습니다(예전 파일은 그대로 둡니다). 포털은 모니터 선택과 권한을 앱 ID에 묶어 두는데 그 ID가 바뀌었으므로, 1–3단계를 한 번 더 하고 단축키가 `scrawl`을 가리키게 하세요. 마지막 영역과 모음은 새로 시작됩니다.

## 사용 방법

1. 단축키를 누릅니다. 화면이 멈추고 어두워집니다.
2. 드래그해서 영역을 선택합니다(아무 모니터에서나). 아래에 도구 모음이 나타납니다.
3. 원하면 주석을 답니다: 도구와 색을 고르고 영역 위에 그립니다.
4. `Enter`를 누릅니다. 이미지가 클립보드와 `~/Pictures/Screenshots`에 들어가니 어디든 붙여 넣으세요.

`Esc`는 언제든 취소합니다.

### 키

| 키 | 동작 |
| --- | --- |
| `Enter` | 클립보드에 복사**하고** 파일로 저장 (동작은 설정 가능) |
| `Ctrl+C` / `Ctrl+S` | 복사만 / 저장만 |
| `Ctrl+Z` | 마지막 주석 취소 |
| `Esc` | 취소 (열린 메뉴가 있으면 먼저 그것을 닫음. 텍스트 입력 중에는 텍스트만 버림) |
| `Ctrl+A` | 해당 모니터의 화면 전체 선택 (키 변경 불가) |
| `R` | 이전 촬영의 영역 불러오기 |
| `V` `P` `L` `W` `B` `M` `U` `T` `N` | 도구: 선택, 펜, 직선, 화살표, 상자, 형광펜, 흐림, 텍스트, 번호 스탬프 (영역이 선택되어 있을 때) |
| `+` / `-` | 현재 도구의 두께, 글자 크기 또는 흐림 강도 (마우스 휠과 같음) |
| `A` | 스크린샷을 모음에 추가 |
| `G` | 모음(과 현재 영역)을 한 장으로 합치기 |
| `D` | 모음을 개별 파일로 저장하고 파일 목록을 클립보드에 넣기 |
| 마우스 휠 | 현재 도구의 두께, 글자 크기 또는 흐림 강도 |
| 그리는 동안 `Shift` | 펜과 형광펜: 직선. 직선과 화살표: 45°에 맞춤. 상자와 흐림: 정사각형. 영역을 선택하는 동안 `Shift`: 정사각형 |

모든 키는 설정 창(“키” 탭)에서 바꿀 수 있습니다.

### 도구 모음

기본 줄은 왼쪽부터: 선택, 펜, 직선, 화살표, 상자, 형광펜, 흐림, 텍스트, 번호 · 실행 취소 · 복사 · 저장 · `⋯` 메뉴(설정: 오버레이를 닫고 설정 창을 엽니다, 그리고 모음에 추가) · 닫기.

그리기 도구를 쓰는 동안에는 그 아래에 두 번째 줄이 나타납니다: 8가지 색, 나만의 색을 위한 무지개 버튼(팔레트가 열리며 색은 기억됩니다), 그리고 `−` 두께 `+`. “선택” 도구에는 둘 다 없으므로 이 줄은 숨겨집니다.

키가 있는 버튼에는 아이콘 아래에 그 키가 표시됩니다(키는 설정에서 가져옵니다). 버튼 위에 커서를 잠시 올려 두면 이름과 키가 보입니다. 단축키는 어떤 키보드 배열에서도(예: 러시아어 배열) 키의 위치를 기준으로 동작합니다.

### 전/후 모음

첫 스크린샷을 찍고 `A`를 누르세요. 다음 것을 찍을 때는 이전 영역이 이미 선택되어 있어서 같은 곳을 다시 찍게 됩니다. `G`를 누르면 한 장의 이미지로 합쳐지고(저장과 복사가 됩니다), `D`를 누르면 개별 파일이 됩니다. 화면 아래의 작은 패널에는 모음에 든 내용이 보이고 같은 동작을 버튼으로 할 수 있습니다.

## 트레이 아이콘과 자동 시작

`scrawl -tray`는 알림 영역에 아이콘을 둡니다. 왼쪽 클릭은 스크린샷, 메뉴에는 “스크린샷 찍기”, “설정”, “종료”가 있습니다. 로그인할 때 시작하려면 *설정 → 시스템 → 자동 시작*을 켜세요. GNOME에서는 아이콘을 보려면 AppIndicator 확장이 필요합니다(Ubuntu에서는 기본으로 켜져 있습니다).

## 설정

![설정 창](../images/settings-en.png)

도구 모음의 톱니바퀴, `scrawl -settings`, 또는 애플리케이션 메뉴에서 엽니다. 변경 사항은 즉시 `~/.config/scrawl/config.json`(일반 JSON 파일이며 직접 편집해도 됩니다)에 저장됩니다. 없거나 잘못된 값은 기본값으로 돌아갑니다. 각 키는 [설정 참조](../SETTINGS.md)에 설명되어 있습니다.

## 파일과 개인정보

- 이미지는 `~/Pictures/Screenshots`(설정에서 변경 가능)에 `scrawl-20260131-142501.png` 같은 이름으로 저장됩니다.
- 설정: `~/.config/scrawl/`. 모니터 토큰, 마지막 영역, 모음: `~/.cache/scrawl/`.
- Scrawl은 스스로 인터넷에 접속하지 않습니다. 프로그램의 유일한 네트워크 요청은 선택 사항인 업데이트 확인(기본값은 꺼짐)으로, GitHub에 최신 릴리스를 묻고 일반적인 HTTP 헤더 외에는 프로그램 버전만 보냅니다.
- 스크린샷은 직접 어딘가에 붙여 넣지 않는 한 컴퓨터를 벗어나지 않습니다.

## 명령줄

| 옵션 | 동작 |
| --- | --- |
| `-settings` | 설정 창을 연다 |
| `-setup` | 모니터를 다시 고른다(GNOME 대화상자) |
| `-grant-screenshot` | 예비 캡처 권한을 부여하고 종료한다 |
| `-tray` | 알림 영역에 아이콘을 둔다 |
| `-fresh` | 이전 촬영의 영역을 제안하지 않는다 |
| `-v` | 소요 시간을 출력한다 |
| `-version` | 버전을 출력하고 종료한다 |
| `-exit-after DURATION` | 지정한 시간이 지나면 정상 종료한다. 테스트용 (예: `-exit-after 3s`) |

## 문제 해결

- **단축키를 눌러도 아무 일도 일어나지 않습니다.** `xdg-desktop-portal`이 멈췄을 수 있습니다. 다시 시작하세요: `systemctl --user restart xdg-desktop-portal xdg-desktop-portal-gnome`.
- **GNOME이 촬영할 때마다 공유할 화면을 묻습니다.** “이 선택 기억”에 체크하지 않았습니다. 1단계를 다시 하세요.
- **모니터를 추가하거나 바꿨습니다.** 모니터를 다시 선택하세요(1단계).
- **오버레이가 모니터 하나에만 열립니다.** 같은 문제입니다. 1단계는 모니터마다 해야 합니다.
- **포털이 거부했다는 메시지가 나옵니다.** 권한을 부여하세요(2단계).
- **클립보드 오류가 나거나 붙여 넣어도 아무것도 나오지 않습니다.** `wl-copy`(패키지 `wl-clipboard`)가 설치되어 있는지 확인하세요.
- **알림이 없습니다.** `libnotify-bin`(`notify-send` 명령)을 설치하고 설정에서 알림이 켜져 있는지 확인하세요.
- **트레이 아이콘이 없습니다.** GNOME AppIndicator 확장을 켜세요.
- **`Print`가 GNOME 스크린샷 도구를 엽니다.** 키보드 단축키에서 끄세요(3단계 참고).

## 제거

**apt 또는 `.deb`:**

```bash
sudo apt remove scrawl     # or: sudo apt purge scrawl (the package keeps no files outside /usr, so they are the same)
sudo rm /etc/apt/sources.list.d/scrawl.sources   # optional: forget the repository (scrawl.list for the old form)
```

apt는 홈 폴더를 건드릴 수 없으므로 사용자별로 남는 것이 몇 가지 있습니다: GNOME 단축키, 켜 두었다면 자동 시작 항목, 그리고 설정입니다. 단축키는 Scrawl의 `gnome-hotkey.sh`(저장소의 `scripts/`와 릴리스 압축 파일에 들어 있습니다)로 제거하거나 *설정 → 키보드 → 바로 가기 보기 및 사용자 지정 → 사용자 지정 바로 가기*에서 직접 지우세요. 나머지는 일반 파일입니다:

```bash
sh gnome-hotkey.sh --remove
rm -f ~/.config/autostart/io.github.jexme.Scrawl.desktop
rm -rf ~/.config/scrawl ~/.cache/scrawl     # optional: settings and cache
```

**릴리스 압축 파일 또는 `make install`:** 풀어 놓은 압축 파일에서 `./uninstall.sh`를 실행하세요(소스 체크아웃에서는 `make uninstall`). 실행 파일, `.desktop` 파일, GNOME 단축키, 자동 시작 항목이 삭제됩니다. `--purge`(`make uninstall PURGE=1`)를 붙이면 설정(`~/.config/scrawl`)과 캐시(`~/.cache/scrawl`)도 삭제됩니다. 설치할 때 `PREFIX`를 바꿨다면 같은 `PREFIX`를 넘기세요. `~/Pictures/Screenshots`의 스크린샷은 절대 건드리지 않습니다.

## 동작 원리

Scrawl은 `xdg-desktop-portal`의 ScreenCast 인터페이스를 통해 자체 PipeWire 클라이언트(`internal/pwcapture`)로 화면을 캡처합니다. 모니터마다 저장된 복원 토큰 덕분에 대화상자가 사라집니다. 컴포지터(Mutter)는 모니터가 다시 그려질 때만 프레임을 주며, 우리 테스트에서는 전체 화면 창 위(다이렉트 스캔아웃)에서는 하나도 주지 않았습니다. 250ms 안에 프레임이 오지 않으면 Scrawl은 더 느린(약 0.5–1초) Screenshot 포털로 전환합니다. 오버레이는 모니터마다 하나씩 있는 전체 화면 [Gio](https://gioui.org) 창이며, Gio에는 창을 지정한 Wayland 출력에 놓기 위한 패치(`third_party/gio.patch`)가 적용되어 있습니다. 결과는 화면에 그리는 것과 같은 코드로 렌더링되므로 저장된 이미지는 본 그대로입니다.

## 빌드와 기여

```bash
make build    # bin/scrawl; 처음 실행 때 Gio를 받아 패치
make test     # 단위 테스트(세션 D-Bus의 트레이 아이콘 포함)
make dist     # dist/에 릴리스 압축 파일과 체크섬 생성
make deb      # .deb package for Debian and Ubuntu in dist/
make notices  # THIRD_PARTY_NOTICES.md 다시 생성
```

화면 문구는 `internal/i18n/`(언어마다 파일 하나, `lang_ru.go`, `lang_de.go` 등. 코드 속 원문은 영어)에 있으며 `go test ./...`가 모든 문구가 모든 언어로 번역되어 있는지 확인합니다. 버그를 신고하고 변경을 보내는 방법: [CONTRIBUTING.md](../../CONTRIBUTING.md). 계획과 아이디어: [docs/PLAN.md](../PLAN.md).

## 라이선스

MIT, [LICENSE](../../LICENSE)를 참고하세요. 서드파티 구성 요소: [THIRD_PARTY_NOTICES.md](../../THIRD_PARTY_NOTICES.md) (아이콘 모양은 [Tabler Icons](https://tabler.io/icons), MIT).
