# Scrawl

**Wayland의 GNOME을 위한 빠른 주석 스크린샷 도구.** 키를 누르고, 아무 모니터에서나 영역을 선택하고, 화살표·상자·텍스트를 그리고, 민감한 정보는 흐리게 처리하면, 결과가 곧바로 클립보드에 들어가고 파일로도 저장됩니다.

[English](../../README.md) · [Русский](README.ru.md) · [简体中文](README.zh-CN.md) · [Español](README.es.md) · [Português (Brasil)](README.pt-BR.md) · [日本語](README.ja.md) · [Deutsch](README.de.md) · [Français](README.fr.md) · **한국어**

![Scrawl 오버레이: 상자, 화살표, 글자, 형광펜 선, 흐림 처리가 있는 선택 영역과 그 아래의 도구 모음](../images/overlay.png)

*이 그림은 가상의 데스크톱 위에 실제 오버레이를 렌더링한 것입니다.*

> 화면은 영어, 러시아어, 중국어(간체), 스페인어, 포르투갈어, 일본어, 독일어, 프랑스어, 한국어로 제공됩니다. 아래의 버튼 이름은 영어 화면 기준이며, 다른 언어에도 대응하는 이름이 있습니다.

## 왜 Scrawl인가

Wayland의 GNOME에서는 앱이 화면을 직접 캡처할 수 없고 데스크톱 포털에 요청해야 하며, 포털은 **촬영할 때마다 “어느 화면을 공유할까요?”라고 묻습니다**. Scrawl은 모니터마다 한 번만 묻고 답을 기억하므로, 그다음부터는 키 한 번이면 스크린샷이 됩니다. 정지된 화면이 약 150ms 안에 나타납니다.

## 기능

**캡처**
- 모든 모니터에서 동시에 동작합니다. 오버레이가 각 모니터에 열리고 어느 모니터에서든 선택할 수 있습니다.
- 점선 테두리로 영역을 선택합니다. 모서리를 끌어 크기를 바꾸고, *선택* 도구에서는 가장자리를 끌어 크기를 바꾸거나 안쪽을 끌어 이동할 수 있습니다.
- 마지막 영역을 기억하고 키 하나(`R`)로 불러옵니다.
- 선택 사항으로 마우스 커서도 함께 캡처할 수 있습니다.

**주석**
- 펜, 직선, 화살표, 상자, 형광펜, **흐림**, 텍스트. 색은 8가지입니다.
- 마우스 휠(또는 `−` / `+` 버튼)로 두께, 글자 크기, 흐림 강도를 바꿉니다.
- `Ctrl+Z`로 실행 취소합니다.

**저장과 공유**
- `Enter`는 이미지를 클립보드에 복사하고 PNG를 `~/Pictures/Screenshots`에 저장합니다. `Ctrl+C`는 복사만, `Ctrl+S`는 저장만 합니다. JPEG도 지원합니다.
- **전/후 모음:** `A`로 스크린샷을 따로 모아 두었다가, 캡션(“Before”, “After”, 세 장 이상이면 “Step 1…”)이 붙은 한 장의 그림으로 나란히 또는 위아래로 이어 붙입니다. 모음을 개별 파일로 보낼 수도 있으며, 채팅에는 첨부 파일로 붙여 넣어집니다.

**내 취향대로**
- 거의 모든 것을 설정 창에서 바꿀 수 있습니다: 키, 색, 기본 도구, 선택 영역의 모양, 파일 이름과 형식, 화면 언어(9개 언어).
- 자동 시작이 있는 트레이 아이콘, 단축키 지정 버튼, 선택 사항인 업데이트 확인(기본값은 꺼짐).

## 상태

활발히 개발 중인 개인 프로젝트입니다. **아직 릴리스가 없으므로** 지금은 소스에서 빌드해야 합니다(몇 분이면 됩니다). 주요 대상은 Wayland의 GNOME(Linux)입니다. **X11** 세션(Xorg의 GNOME, KDE, XFCE 등)에서도 동작하지만, 이 지원은 새것이고 지금까지 가상 X 서버에서만 테스트했습니다. 실제 기기에서의 제보를 환영합니다. Windows용은 빌드는 되지만 화면 캡처는 아직 작성되지 않았습니다. 자세한 내용은 [docs/WINDOWS.md](../WINDOWS.md)를 참고하세요.

## 요구 사항

- PipeWire를 쓰는 Wayland의 GNOME, `xdg-desktop-portal`(1.21 이상, 앱 등록 지원 필요), `xdg-desktop-portal-gnome`. 개발과 테스트는 Ubuntu와 GNOME 50에서 했습니다.
- 클립보드용 `wl-clipboard`(`wl-copy` 명령)와 알림용 `libnotify`(`notify-send` 명령).
- **X11**에는 포털이 없어 화면을 직접 캡처하며, 클립보드에는 `wl-clipboard` 대신 `xclip`이 필요합니다.

```bash
sudo apt install wl-clipboard libnotify-bin   # Wayland
sudo apt install xclip libnotify-bin          # X11
```

## 설치

### 1. 프로그램 받기

**apt 저장소에서** (Debian, Ubuntu; 업데이트는 `apt upgrade`로 받습니다):

```bash
curl -fsSL https://jexme.github.io/scrawl/scrawl-archive-keyring.gpg | sudo tee /usr/share/keyrings/scrawl-archive-keyring.gpg >/dev/null
echo "deb [signed-by=/usr/share/keyrings/scrawl-archive-keyring.gpg] https://jexme.github.io/scrawl stable main" | sudo tee /etc/apt/sources.list.d/scrawl.list
sudo apt update && sudo apt install scrawl
```

저장소는 다음 키로 서명되어 있습니다: `E62C DAF5 A1C9 0DB7 A899 D24D FA43 15C7 154A 9EF0`.

**소스에서 (지금 가능):**

```bash
sudo apt install gcc pkg-config patch libwayland-dev libx11-dev libx11-xcb-dev \
  libxkbcommon-x11-dev libgles2-mesa-dev libegl1-mesa-dev libffi-dev \
  libxcursor-dev libvulkan-dev libpipewire-0.3-dev
git clone https://github.com/jexme/scrawl.git && cd scrawl
make install            # 빌드 후 ~/.local에 설치 (시스템 전체에는 PREFIX=/usr/local)
```

Go(`go.mod`에 적힌 버전)가 필요합니다. `sudo`가 없다면 `scripts/fetch-pipewire-headers.sh`가 PipeWire 헤더를 `.deps/`에 풀어 주고 Makefile이 알아서 사용합니다.

**릴리스 압축 파일에서 (버전이 공개된 뒤):** [Releases](https://github.com/jexme/scrawl/releases) 페이지에서 `scrawl-VERSION-linux-amd64.tar.gz`와 `SHA256SUMS`를 받은 다음:

```bash
sha256sum -c --ignore-missing SHA256SUMS
tar xzf scrawl-*-linux-amd64.tar.gz && cd scrawl-*-linux-amd64
./install.sh            # ~/.local에 설치 (시스템 전체에는 PREFIX=/usr/local)
sudo apt install libpipewire-0.3-0 libwayland-client0 libwayland-egl1 libwayland-cursor0 \
  libxkbcommon-x11-0 libegl1 libx11-6 libxcursor1
```

`~/.local/bin`이 `PATH`에 들어 있는지 확인하세요.

### 2. 처음 실행: 한 번만 하는 세 단계

*X11에서는 1, 2단계가 필요 없고(포털이 없습니다) 단축키만 설정하면 됩니다.*

설정 창(`scrawl -settings`, tab “System”)의 버튼을 쓰거나 명령을 실행하세요:

1. **모니터 선택:** 버튼 “Pick monitors again” 또는 `scrawl -setup`. GNOME이 **모니터마다** 대화상자를 보여 줍니다. 매번 *다른* 모니터를 고르고 **“이 선택 기억”**에 체크한 채 공유를 허용하세요. 그 뒤로는 대화상자가 나오지 않습니다.
2. **예비 권한 부여:** 버튼 “Grant permission” 또는 `scrawl -grant-screenshot`. PipeWire가 화면을 주지 못하는 드문 경우에 Scrawl이 더 느린 Screenshot 포털을 쓸 수 있게 해 줍니다.
3. **단축키 지정:** “Launch hotkey” 입력란과 “Assign” 버튼(기본값 `Print`), 또는 `scripts/gnome-hotkey.sh [키]`. GNOME 사용자 지정 단축키를 만들며, 이미 쓰고 있는 조합은 덮어쓰지 않습니다. `Print`가 여전히 GNOME 기본 스크린샷 도구를 연다면 *설정 → 키보드 → 바로 가기 보기 및 사용자 지정 → 스크린샷*에서 끄세요.

### screen-editor에서 업그레이드

Scrawl은 screen-editor의 새 이름입니다. `~/.config/scrawl/config.json`이 생기기 전까지는 `~/.config/screen-editor/config.json`에서 설정을 읽습니다(예전 파일은 그대로 둡니다). 포털은 모니터 선택과 권한을 앱 ID에 묶어 두는데 그 ID가 바뀌었으므로, 1~3단계를 한 번 더 하고 단축키가 `scrawl`을 가리키게 하세요. 마지막 영역과 모음은 새로 시작됩니다.

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
| `Esc` | 취소 |
| `R` | 이전 촬영의 영역 불러오기 |
| `A` | 스크린샷을 모음에 추가 |
| `G` | 모음(과 현재 영역)을 한 장으로 합치기 |
| `D` | 모음을 개별 파일로 저장하고 파일 목록을 클립보드에 넣기 |
| 마우스 휠 | 현재 도구의 두께, 글자 크기 또는 흐림 강도 |

모든 키는 설정 창(“Keys”)에서 바꿀 수 있습니다.

### 도구 모음

왼쪽부터: 선택, 펜, 직선, 화살표, 상자, 형광펜, 흐림, 텍스트 · 8가지 색 · `−` 두께 `+` · 설정(오버레이를 닫고 설정 창을 엽니다) · 모음에 추가 · 실행 취소 · 복사 · 저장 · 닫기.

### 전/후 모음

첫 스크린샷을 찍고 `A`를 누르세요. 다음 것을 찍을 때는 이전 영역이 이미 선택되어 있어서 같은 곳을 다시 찍게 됩니다. `G`를 누르면 한 장의 이미지로 합쳐지고(저장과 복사가 됩니다), `D`를 누르면 개별 파일이 됩니다. 화면 아래의 작은 패널에는 모음에 든 내용이 보이고 같은 동작을 버튼으로 할 수 있습니다.

## 트레이 아이콘과 자동 시작

`scrawl -tray`는 알림 영역에 아이콘을 둡니다. 왼쪽 클릭은 스크린샷, 메뉴에는 “Screenshot”, “Settings”, “Quit”이 있습니다. 로그인할 때 시작하려면 *Settings → System → Autostart*를 켜세요. GNOME에서는 아이콘을 보려면 AppIndicator 확장이 필요합니다(Ubuntu에서는 기본으로 켜져 있습니다).

## 설정

![설정 창](../images/settings-en.png)

도구 모음의 톱니바퀴, `scrawl -settings`, 또는 애플리케이션 메뉴에서 엽니다. 변경 사항은 즉시 `~/.config/scrawl/config.json`(일반 JSON 파일이며 직접 편집해도 됩니다)에 저장됩니다. 없거나 잘못된 값은 기본값으로 돌아갑니다. 각 키는 [설정 참조](../SETTINGS.md)(영어 또는 [러시아어](../SETTINGS.md))에 설명되어 있습니다.

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

## 문제 해결

- **단축키를 눌러도 아무 일도 일어나지 않습니다.** `xdg-desktop-portal`이 멈췄을 수 있습니다. 다시 시작하세요: `systemctl --user restart xdg-desktop-portal xdg-desktop-portal-gnome`.
- **GNOME이 촬영할 때마다 공유할 화면을 묻습니다.** “이 선택 기억”에 체크하지 않았습니다. 1단계를 다시 하세요.
- **모니터를 추가하거나 바꿨습니다.** 모니터를 다시 선택하세요(1단계).
- **오버레이가 모니터 하나에만 열립니다.** 같은 문제입니다. 1단계는 모니터마다 해야 합니다.
- **포털이 거부했다는 메시지가 나옵니다.** 권한을 부여하세요(2단계).
- **어떤 앱에 붙여 넣어도 아무것도 나오지 않습니다.** `wl-copy`(패키지 `wl-clipboard`)가 설치되어 있는지 확인하세요.
- **알림이 없습니다.** `libnotify-bin`(`notify-send` 명령)을 설치하고 설정에서 알림이 켜져 있는지 확인하세요.
- **트레이 아이콘이 없습니다.** GNOME AppIndicator 확장을 켜세요.
- **`Print`가 GNOME 스크린샷 도구를 엽니다.** 키보드 단축키에서 끄세요(3단계 참고).

## 동작 원리

Scrawl은 `xdg-desktop-portal`의 ScreenCast 인터페이스를 통해 자체 PipeWire 클라이언트(`internal/pwcapture`)로 화면을 캡처합니다. 모니터마다 저장된 복원 토큰 덕분에 대화상자가 사라집니다. 컴포지터(Mutter)는 모니터가 다시 그려질 때만 프레임을 주며, 우리 테스트에서는 전체 화면 창 위(다이렉트 스캔아웃)에서는 하나도 주지 않았습니다. 250ms 안에 프레임이 오지 않으면 Scrawl은 더 느린(약 0.5~1초) Screenshot 포털로 전환합니다. 오버레이는 모니터마다 하나씩 있는 전체 화면 [Gio](https://gioui.org) 창이며, Gio에는 창을 지정한 Wayland 출력에 놓기 위한 패치(`third_party/gio.patch`)가 적용되어 있습니다. 결과는 화면에 그리는 것과 같은 코드로 렌더링되므로 저장된 이미지는 본 그대로입니다.

## 빌드와 기여

```bash
make build    # bin/scrawl; 처음 실행 때 Gio를 받아 패치
make test     # 단위 테스트(세션 D-Bus의 트레이 아이콘 포함)
make dist     # dist/에 릴리스 압축 파일과 체크섬 생성
make notices  # THIRD_PARTY_NOTICES.md 다시 생성
```

화면 문구는 `internal/i18n/`(언어마다 파일 하나, 코드 속 원문은 러시아어)에 있으며 `go test ./...`가 모든 문구가 모든 언어로 번역되어 있는지 확인합니다. 계획과 아이디어: [docs/PLAN.md](../PLAN.md)(러시아어).

## 라이선스

MIT, [LICENSE](../../LICENSE)를 참고하세요. 서드파티 구성 요소: [THIRD_PARTY_NOTICES.md](../../THIRD_PARTY_NOTICES.md) (아이콘 모양은 [Tabler Icons](https://tabler.io/icons), MIT).
