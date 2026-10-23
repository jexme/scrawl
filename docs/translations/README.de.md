# Scrawl

**Schnelle Screenshots mit Anmerkungen für GNOME unter Wayland.** Taste drücken, auf einem beliebigen Monitor einen Bereich auswählen, Pfeile, Rahmen und Text zeichnen, Privates unkenntlich machen – und das Ergebnis liegt schon in der Zwischenablage und als Datei auf der Platte.

[English](../../README.md) · [Русский](README.ru.md) · [简体中文](README.zh-CN.md) · [Español](README.es.md) · [Português (Brasil)](README.pt-BR.md) · [日本語](README.ja.md) · **Deutsch** · [Français](README.fr.md) · [한국어](README.ko.md)

![Scrawl-Overlay: ein ausgewählter Bereich mit Rahmen, Pfeil, Beschriftung, Textmarkerstrich und unscharfer Stelle, darunter die Werkzeugleiste](../images/overlay.png)

*Das Bild ist eine echte Darstellung des Overlays über einem erfundenen Desktop.*

> Die Oberfläche gibt es auf Englisch, Russisch, Chinesisch (vereinfacht), Spanisch, Portugiesisch, Japanisch, Deutsch, Französisch und Koreanisch; die unten genannten Schaltflächen sind mit den Namen der deutschen Oberfläche angegeben.

## Warum Scrawl

Unter GNOME mit Wayland können Programme den Bildschirm nicht direkt aufnehmen: Sie bitten das Desktop-Portal darum, und das Portal **fragt bei jedem Screenshot „Welchen Bildschirm möchten Sie freigeben?“**. Scrawl fragt einmal pro Monitor, merkt sich die Antwort, und danach ist ein Screenshot ein einziger Tastendruck: Der eingefrorene Bildschirm erscheint nach etwa 150 ms.

## Schnellstart

Unter Ubuntu oder Debian (Wayland oder X11) sind es drei Schritte.

**1. Installieren** aus dem signierten apt-Repository (Updates kommen dann mit `apt upgrade`):

```bash
sudo curl -fsSL https://jexme.github.io/scrawl/scrawl.sources -o /etc/apt/sources.list.d/scrawl.sources
sudo apt update && sudo apt install scrawl
```

Die `.sources`-Datei enthält den Signaturschlüssel selbst, einen separaten Schlüsselbund muss man also nicht verwalten. Die Pakete
gibt es nur für **amd64**; unter arm64 [aus dem Quellcode bauen](#installation).

**2. Einmal einrichten:** Starten Sie Scrawl aus dem Anwendungsmenü (oder führen Sie `scrawl` aus): Beim ersten Start öffnet sich eine kurze Anleitung
im Tab „System“ (Sie können sie jederzeit mit `scrawl -settings` öffnen). Drücken Sie unter Wayland „Aufnahme einrichten“
(GNOME fragt einmal pro Monitor: Wählen Sie in jedem Dialog einen anderen Monitor und haken Sie „Diese Auswahl merken“ an). Geben Sie dann ein Tastenkürzel ein
(standardmäßig `Print`) und drücken Sie „Zuweisen“. Auch das Screenshot-Werkzeug von GNOME nutzt `Print`: Drücken Sie „Taste freigeben“,
um dieses eingebaute Kürzel abzuschalten (nur diese eine Taste, und es lässt sich rückgängig machen). Unter X11 braucht man nur das Tastenkürzel.

**3. Einen Screenshot machen:** Kürzel drücken, einen Bereich aufziehen, `Enter` drücken. Das Bild liegt in der Zwischenablage und in
`~/Pictures/Screenshots`.

Andere Systeme: [Windows](#windows-10-und-11), andere Linux-Distributionen (ein Release-Archiv oder ein Bau aus dem Quellcode), siehe
unten [Installation](#installation).

## Funktionen

**Aufnahme**
- Funktioniert auf allen Monitoren gleichzeitig: Das Overlay öffnet sich auf jedem, und man kann auf jedem auswählen.
- Auswahl mit gestricheltem Rahmen. Ecken ziehen ändert die Größe; mit dem Werkzeug *Auswählen* ändert das Ziehen an einer Kante die Größe, das Ziehen im Inneren verschiebt den Bereich.
- Beim Ziehen eines Rahmens zeigt eine Lupe neben dem Mauszeiger die Pixel sechsfach vergrößert, mit der Kante des Rahmens auf dem Pixelraster, sodass man ihn genau ausrichten kann (sie lässt sich in den Einstellungen abschalten).
- Merkt sich den letzten Bereich und holt ihn mit einer Taste (`R`) zurück.
- Optional kommt der Mauszeiger mit ins Bild (vorerst nur unter Wayland).

**Anmerkungen**
- Stift, Linie, Pfeil, Rahmen, Textmarker, **Weichzeichner**, Text und **nummerierte Stempel** (ein Klick setzt einen Kreis mit 1, 2, 3 und so weiter, wie in flameshot); acht Farben.
- Das Mausrad (oder die Tasten `−` / `+`) ändert Dicke, Textgröße oder Stärke der Unschärfe.
- Der **Weichzeichner** hat sechs Stufen (Mausrad oder `−` / `+`): 1–3 weicher Gaußscher Weichzeichner, 4–6 Mosaik mit wachsenden Quadraten. Die Mosaikzellen beginnen an der Ecke des Bereichs, und der Hinweis, der beim Ändern der Stufe erscheint, nennt den Modus.
- **Shift** beim Zeichnen gedrückt halten: Stift und Textmarker ziehen eine gerade Linie, eine Linie oder ein Pfeil rastet in 45°-Schritten ein, ein Rahmen oder Weichzeichner wird zum Quadrat.
- Rückgängig mit `Strg+Z` (`Ctrl+Z`).

**Speichern und teilen**
- `Enter` kopiert das Bild in die Zwischenablage und speichert ein PNG in `~/Pictures/Screenshots`. `Ctrl+C` kopiert nur, `Ctrl+S` speichert nur. JPEG gibt es auch.
- Ein **Doppelklick** im ausgewählten Bereich kopiert ihn sofort (in den Einstellungen, Tab „Allgemein“, Abschnitt „Aktionen“, kann er auch kopieren und speichern oder nichts tun).
- **Vorher/Nachher-Sammlungen:** Mit `A` Aufnahmen beiseitelegen und dann zu einem Bild mit Beschriftungen („Before“, „After“ oder „Step 1…“ bei mehr) zusammenfügen – nebeneinander oder untereinander. Oder die Sammlung als einzelne Dateien senden, die sich in einem Chat als Anhänge einfügen lassen.

**Nach eigenem Geschmack**
- Ein Einstellungsfenster für fast alles: Tasten, Farben, Standardwerkzeuge, Aussehen der Auswahl, Dateinamen und -format sowie Sprache der Oberfläche (9 Sprachen: Englisch, Russisch, Chinesisch, Spanisch, Portugiesisch, Japanisch, Deutsch, Französisch, Koreanisch).
- Ein Tray-Symbol mit Autostart, eine Schaltfläche zum Festlegen des Tastenkürzels und eine optionale Update-Prüfung (standardmäßig aus).

## Status

Ein persönliches Projekt in aktiver Entwicklung; die aktuelle Version ist eine Vorabversion (`v0.1.0-rc4`). Fertige Pakete gibt es auf der Seite
[Releases](https://github.com/jexme/scrawl/releases); man kann es auch aus dem Quellcode bauen (ein paar Minuten).
Linux: Hauptziel ist GNOME unter Wayland. **X11**-Sitzungen (GNOME unter Xorg, KDE,
XFCE und andere) funktionieren ebenfalls, doch diese Unterstützung ist neu und bisher nur auf einem
virtuellen X-Server getestet; Rückmeldungen von echten Rechnern sind willkommen. **Windows 10 und 11** funktionieren ebenfalls (Aufnahme, Anmerkungen,
Zwischenablage, Tray-Symbol und ein globales Tastenkürzel); diese Portierung ist neu und wurde nur bei 100 % Skalierung ausprobiert, zum Bauen und
dazu, was fehlt, siehe [docs/WINDOWS.md](../WINDOWS.md).

## Voraussetzungen

Bei der Installation mit apt oder der `.deb`-Datei werden diese Pakete automatisch mitinstalliert; die Liste ist für das Release-Archiv und den Bau aus dem Quellcode gedacht.

- GNOME unter Wayland mit PipeWire, `xdg-desktop-portal` (ab 1.21, mit Unterstützung für die Registrierung von Anwendungen) und `xdg-desktop-portal-gnome`. Entwickelt und getestet unter Ubuntu mit GNOME 50.
- `wl-clipboard` (der Befehl `wl-copy`) für die Zwischenablage und `libnotify` (der Befehl `notify-send`) für Benachrichtigungen. Ohne `wl-copy` weicht Scrawl über XWayland auf `xclip` aus, wenn beide vorhanden sind; ist keines von beiden da, erhalten Sie einen Hinweis, was zu installieren ist.
- Unter **X11** gibt es kein Portal: Der Bildschirm wird direkt aufgenommen, und für die Zwischenablage braucht man `xclip` statt `wl-clipboard`.

```bash
sudo apt install wl-clipboard libnotify-bin   # Wayland
sudo apt install xclip libnotify-bin          # X11
```

## Installation

### 1. Das Programm besorgen

**Aus dem apt-Repository** (Debian, Ubuntu; Updates kommen mit `apt upgrade`):

```bash
sudo curl -fsSL https://jexme.github.io/scrawl/scrawl.sources -o /etc/apt/sources.list.d/scrawl.sources
sudo apt update && sudo apt install scrawl
```

Die Pakete gibt es nur für amd64 (unter arm64 aus dem Quellcode bauen, siehe unten). Das Repository ist mit dem
Schlüssel `E62C DAF5 A1C9 0DB7 A899 D24D FA43 15C7 154A 9EF0` signiert, der in `scrawl.sources` eingebettet ist.

Auf älteren Systemen (Ubuntu 20.04, Debian 11: apt älter als 2.4 kann einen Schlüssel in einer `.sources`-Datei nicht lesen) stattdessen einen
separaten Schlüsselbund und eine einzeilige Quelle verwenden:

```bash
curl -fsSL https://jexme.github.io/scrawl/scrawl-archive-keyring.gpg | sudo tee /usr/share/keyrings/scrawl-archive-keyring.gpg >/dev/null
echo "deb [signed-by=/usr/share/keyrings/scrawl-archive-keyring.gpg] https://jexme.github.io/scrawl stable main" | sudo tee /etc/apt/sources.list.d/scrawl.list
sudo apt update && sudo apt install scrawl
```

Der Schlüssel wird auch ASCII-armored als `scrawl-archive-keyring.asc` veröffentlicht.

**Aus einem .deb-Paket** (Debian, Ubuntu): `scrawl_VERSION_amd64.deb`
von der Seite [Releases](https://github.com/jexme/scrawl/releases) herunterladen, dann
`sudo apt install ./scrawl_*.deb`. Es zieht die Bibliotheken nach und empfiehlt `wl-clipboard`, `xclip`
und `libnotify-bin`. Das Tastenkürzel wird vom Paket nicht gesetzt (es gilt pro Benutzer): Tun Sie das in `scrawl -settings`.
Selbst bauen: `make deb`.

**Aus einem Release-Archiv**: `scrawl-VERSION-linux-amd64.tar.gz` und `SHA256SUMS` von der Seite
[Releases](https://github.com/jexme/scrawl/releases) herunterladen, dann:

```bash
sha256sum -c --ignore-missing SHA256SUMS   # prüft jede heruntergeladene Datei, die in SHA256SUMS steht
tar xzf scrawl-*-linux-amd64.tar.gz && cd scrawl-*-linux-amd64
./install.sh            # installiert nach ~/.local (PREFIX=/usr/local für das ganze System)
sudo apt install libpipewire-0.3-0 libwayland-client0 libwayland-egl1 libwayland-cursor0 \
  libxkbcommon-x11-0 libegl1 libx11-6 libxcursor1 wl-clipboard libnotify-bin
```

`wl-clipboard` (die Zwischenablage) und `libnotify-bin` (Benachrichtigungen) werden auch zur Laufzeit gebraucht: Das
Installationsskript prüft sie und gibt aus, was fehlt. Unter X11 `xclip` statt `wl-clipboard` verwenden.

**Aus dem Quellcode**:

```bash
sudo apt install gcc pkg-config patch libwayland-dev libx11-dev libx11-xcb-dev \
  libxkbcommon-x11-dev libgles2-mesa-dev libegl1-mesa-dev libffi-dev \
  libxcursor-dev libvulkan-dev libpipewire-0.3-dev
git clone https://github.com/jexme/scrawl.git && cd scrawl
make install            # baut und installiert nach ~/.local (PREFIX=/usr/local für das ganze System)
```

Sie brauchen Go (die Version aus `go.mod`). Kein `sudo`? `scripts/fetch-pipewire-headers.sh` entpackt die PipeWire-Header nach `.deps/`, und das Makefile nutzt sie.

Stellen Sie sicher, dass `~/.local/bin` in Ihrem `PATH` liegt.

### Windows 10 und 11

**Empfohlen: das Installationsprogramm.** Laden Sie `scrawl-VERSION-windows-amd64-setup.exe` von der Seite
[Releases](https://github.com/jexme/scrawl/releases) herunter und führen Sie es aus. Es installiert nur für Ihr Konto (keine Administratorrechte),
legt einen Eintrag im Startmenü an, kann Scrawl bei der Anmeldung starten und startet es nach Abschluss. Zum Aktualisieren führen Sie das neuere
Setup aus: Es beendet zuerst das alte Scrawl. Hat ein Release keine Setup-Datei, nehmen Sie eine der folgenden Möglichkeiten.

Alternativen:

- **Portables zip.** Laden Sie `scrawl-VERSION-windows-amd64.zip` und die zugehörige `.sha256` herunter, entpacken Sie es in einen dauerhaften Ordner (nicht
  direkt aus dem Archivbetrachter) und führen Sie `scrawl.exe` aus. Ein Doppelklick auf `scrawl.exe` startet das Symbol im Infobereich.
  Auch eine einzelne `scrawl-VERSION-windows-amd64.exe` wird veröffentlicht. Zum Aktualisieren beenden Sie Scrawl über das Menü des Symbols und ersetzen die Datei.
- **[Scoop](https://scoop.sh):** `scoop install https://github.com/jexme/scrawl/releases/download/v0.1.0-rc4/scrawl.json`.
  Diese Adresse ist an eine Vorabversion gebunden; zwischen Release-Kandidaten aktualisieren Sie daher mit `scoop uninstall scrawl` und `scoop install`
  mit der URL der neuen Version (bis es ein stabiles Release oder einen Bucket gibt). Sobald das Manifest unter
  `https://github.com/jexme/scrawl/releases/latest/download/scrawl.json` ausgeliefert wird (ein stabiles Release existiert), funktioniert `scoop update scrawl`.

- **Das Symbol** liegt im Infobereich; unter Windows 11 kann es sich hinter dem Pfeil `^` neben der Uhr verstecken – ziehen Sie es heraus, damit es sichtbar bleibt.
- **Die Taste Print** ist das Standard-Tastenkürzel. Oft belegt sie ein anderes Programm (Greenshot, ShareX, Lightshot). Auch Windows selbst kann sie belegen:
  Einstellungen, Barrierefreiheit, Tastatur, Option *Use the Print screen button to open screen capture* (in der deutschen Oberfläche entsprechend benannt): ausschalten. Oder wählen Sie in
  `scrawl.exe -settings` eine andere Taste, zum Beispiel `<Control><Alt>a`. Das Symbol meldet, wenn die Taste belegt ist, und versucht es weiter.
- **„Der Computer wurde durch Windows geschützt“** („Windows protected your PC“, SmartScreen) erscheint, weil das Programm noch nicht code-signiert ist: Klicken Sie auf *Weitere Informationen*, dann
  auf *Trotzdem ausführen*. Um die Datei vorher zu prüfen, vergleichen Sie `Get-FileHash` mit der `.sha256`-Datei oder führen Sie
  `gh attestation verify scrawl-*-setup.exe --repo jexme/scrawl` aus (der Build stammt aus öffentlicher CI). Virenscanner können
  jedes neue unsignierte Go-Programm markieren; melden Sie einen solchen Fall als Issue.
- **Autostart** ist das Kontrollkästchen in `scrawl.exe -settings`, Tab *System* (ein Wert unter `HKCU\...\Run`), oder ein Kästchen im Installationsprogramm.
  Wenn Sie das Programm verschieben oder aktualisieren, schreibt der nächste Start des Symbols den Eintrag auf den neuen Ort um (ein ausgeschalteter Autostart bleibt aus).
- **Eine neuere Version starten, während eine ältere läuft:** Scrawl sagt Ihnen, welche Version läuft; beenden Sie sie zuerst über das
  Menü des Symbols und starten Sie dann die neue.

Bauen aus dem Quellcode und was fehlt: [docs/WINDOWS.md](../WINDOWS.md).

### 2. Erster Start: drei einmalige Schritte

*Unter X11 sind die Schritte 1 und 2 nicht nötig (es gibt kein Portal), nur das Tastenkürzel.*

Öffnen Sie das Einstellungsfenster (`scrawl -settings`, Tab „System“). Die Schaltfläche „Aufnahme einrichten“ erledigt die Schritte 1 und 2 in einem Zug (beim allerersten Start öffnet sich das Fenster von selbst). Oder nutzen Sie die einzelnen Schaltflächen oder führen Sie die Befehle aus:

1. **Monitore auswählen:** Schaltfläche „Monitore neu auswählen“ oder `scrawl -setup`. GNOME zeigt **pro Monitor** einen Dialog. Wählen Sie in jedem einen *anderen* Monitor, lassen Sie **„Diese Auswahl merken“** angehakt und bestätigen Sie die Freigabe. Danach gibt es keine Dialoge mehr.
2. **Ersatz-Berechtigung erteilen:** Schaltfläche „Berechtigung erteilen“ oder `scrawl -grant-screenshot`. Damit darf Scrawl in den seltenen Fällen, in denen PipeWire kein Bild liefert, das langsamere Screenshot-Portal verwenden.
3. **Tastenkürzel festlegen:** Unter GNOME weisen das Installationsskript des Archivs (`install.sh`) und `make install` `Print` selbst zu, sofern es nicht schon ein anderes benutzerdefiniertes Kürzel belegt. Nach der Installation mit apt oder der `.deb`-Datei oder um die Taste zu ändern: das Feld „Tastenkürzel zum Starten“ und die Schaltfläche „Zuweisen“ (Standard `Print`) oder `scripts/gnome-hotkey.sh [TASTE]`. Es legt ein benutzerdefiniertes GNOME-Kürzel namens „scrawl“ an (oder aktualisiert es, falls es existiert, sodass ein erneuter Aufruf nie ein zweites erzeugt) und überschreibt nie das Kürzel eines anderen, das bereits belegt ist. Öffnet `Print` weiterhin das GNOME-eigene Screenshot-Werkzeug, schalten Sie es unter *Einstellungen → Tastatur → Tastenkürzel anzeigen und anpassen → Bildschirmfotos* ab.

### Umstieg von screen-editor

Scrawl ist der neue Name von screen-editor. Ihre Einstellungen werden aus `~/.config/screen-editor/config.json` gelesen, bis `~/.config/scrawl/config.json` existiert (die alte Datei bleibt unangetastet). Das Portal bindet Monitorauswahl und Berechtigungen an die App-ID, und die hat sich geändert – wiederholen Sie also die Schritte 1 bis 3 einmal und richten Sie Ihr Tastenkürzel auf `scrawl`. Der letzte Bereich und die Sammlung beginnen von vorn.

## Benutzung

1. Drücken Sie Ihr Tastenkürzel. Der Bildschirm friert ein und wird abgedunkelt.
2. Ziehen Sie einen Bereich auf (auf einem beliebigen Monitor). Darunter erscheint eine Werkzeugleiste.
3. Wenn Sie möchten, anmerken: Werkzeug und Farbe wählen und auf dem Bereich zeichnen.
4. Drücken Sie `Enter`. Das Bild liegt in der Zwischenablage und in `~/Pictures/Screenshots` – fügen Sie es beliebig ein.

`Esc` bricht jederzeit ab.

### Tasten

| Taste | Aktion |
| --- | --- |
| `Enter` | in die Zwischenablage kopieren **und** als Datei speichern (einstellbar, was genau passiert) |
| `Ctrl+C` / `Ctrl+S` | nur kopieren / nur speichern |
| `Ctrl+Z` | letzte Anmerkung rückgängig machen |
| `Esc` | abbrechen (schließt zuerst ein offenes Menü; beim Eintippen eines Textes verwirft es nur den Text) |
| `Ctrl+A` | den ganzen Bildschirm des Monitors auswählen (nicht umbelegbar) |
| `R` | den Bereich der vorherigen Aufnahme zurückholen |
| `V` `P` `L` `W` `B` `M` `U` `T` `N` | Werkzeuge: Auswählen, Stift, Linie, Pfeil, Rahmen, Textmarker, Unschärfe, Text, Nummer (wenn ein Bereich ausgewählt ist) |
| `+` / `-` | Dicke, Textgröße oder Unschärfestärke des aktuellen Werkzeugs (wie das Mausrad) |
| `A` | die Aufnahme zur Sammlung hinzufügen |
| `G` | die Sammlung (und den aktuellen Bereich) zu einem Bild zusammenfügen |
| `D` | die Sammlung als einzelne Dateien speichern und die Dateiliste in die Zwischenablage legen |
| Mausrad | Dicke, Textgröße oder Unschärfestärke des aktuellen Werkzeugs |
| `Shift` beim Zeichnen | Stift und Textmarker: eine gerade Linie; Linie und Pfeil: Einrasten in 45°-Schritten; Rahmen und Unschärfe: ein Quadrat. `Shift` beim Auswählen eines Bereichs: ein Quadrat |

Jede Taste lässt sich im Einstellungsfenster (Tab „Tasten“) ändern.

### Werkzeugleiste

Die Hauptzeile von links nach rechts: Auswählen, Stift, Linie, Pfeil, Rahmen, Textmarker, Unschärfe, Text, Nummer ·
rückgängig · kopieren · speichern · das Menü `⋯` (Einstellungen, das das Overlay schließt und das
Einstellungsfenster öffnet, und Zur Sammlung) · schließen.

Ist ein Zeichenwerkzeug aktiv, erscheint darunter eine zweite Zeile: acht Farben, eine Regenbogen-Schaltfläche für
eine eigene Farbe (öffnet eine Palette; die Farbe wird gemerkt) und `−` Dicke `+`. Das Werkzeug „Auswählen“
hat beides nicht, daher ist die Zeile ausgeblendet.

Die Schaltflächen, die eine Taste haben, zeigen sie unter dem Symbol (die Tasten stammen aus den Einstellungen). Halten Sie den Mauszeiger kurz auf einer Schaltfläche, um ihren Namen und ihre Taste zu sehen. Die Kürzel funktionieren in jeder Tastaturbelegung (zum Beispiel einer russischen), und zwar nach der Position der Taste.

### Vorher/Nachher-Sammlungen

Machen Sie die erste Aufnahme und drücken Sie `A`. Machen Sie die nächste: Der vorherige Bereich ist schon ausgewählt, Sie fotografieren also dieselbe Stelle noch einmal. Mit `G` fügen Sie die Aufnahmen zu einem Bild zusammen (es wird gespeichert und kopiert), mit `D` erhalten Sie einzelne Dateien. Das kleine Feld am unteren Bildschirmrand zeigt, was in der Sammlung liegt, und bietet dieselben Aktionen als Schaltflächen.

## Tray-Symbol und Autostart

`scrawl -tray` hält ein Symbol im Benachrichtigungsbereich: Linksklick macht einen Screenshot, das Menü bietet „Screenshot aufnehmen“, „Einstellungen“ und „Beenden“. Damit es beim Anmelden startet, schalten Sie *Einstellungen → System → Autostart* ein. Unter GNOME braucht das Symbol die AppIndicator-Erweiterung (unter Ubuntu standardmäßig aktiv).

## Einstellungen

![Das Einstellungsfenster](../images/settings-en.png)

Öffnen Sie sie über die Werkzeugleiste (Zahnrad), mit `scrawl -settings` oder über das Anwendungsmenü. Änderungen werden sofort in `~/.config/scrawl/config.json` gespeichert, einer gewöhnlichen JSON-Datei, die Sie auch von Hand bearbeiten können. Fehlende oder ungültige Werte fallen auf die Standardwerte zurück. Jeder Schlüssel ist in der [Einstellungsreferenz](../SETTINGS.md) beschrieben (auf Englisch oder [Russisch](SETTINGS.ru.md)).

## Dateien und Datenschutz

- Bilder landen in `~/Pictures/Screenshots` (in den Einstellungen änderbar) und heißen etwa `scrawl-20260131-142501.png`.
- Einstellungen: `~/.config/scrawl/`. Monitor-Token, letzter Bereich und Sammlung: `~/.cache/scrawl/`.
- Scrawl geht nie von selbst ins Internet. Die einzige Netzwerkanfrage des Programms ist die optionale Update-Prüfung (standardmäßig aus): Sie fragt GitHub nach dem neuesten Release und sendet außer den üblichen HTTP-Headern nur die Programmversion.
- Screenshots verlassen Ihren Rechner nicht, solange Sie sie nicht selbst irgendwo einfügen.

## Kommandozeile

| Option | Wirkung |
| --- | --- |
| `-settings` | öffnet das Einstellungsfenster |
| `-setup` | wählt die Monitore neu aus (GNOME-Dialoge) |
| `-grant-screenshot` | erteilt die Ersatz-Aufnahmeberechtigung und beendet das Programm |
| `-tray` | hält das Symbol im Benachrichtigungsbereich |
| `-fresh` | bietet den Bereich der vorherigen Aufnahme nicht an |
| `-v` | gibt Zeitmessungen aus |
| `-version` | gibt die Version aus und beendet das Programm |
| `-exit-after DURATION` | beendet sich nach der angegebenen Zeit normal, für Tests (zum Beispiel `-exit-after 3s`) |

## Fehlerbehebung

- **Beim Drücken des Tastenkürzels passiert nichts.** Möglicherweise hängt `xdg-desktop-portal`. Starten Sie es neu: `systemctl --user restart xdg-desktop-portal xdg-desktop-portal-gnome`.
- **GNOME fragt bei jedem Screenshot, welcher Bildschirm freigegeben werden soll.** „Diese Auswahl merken“ war nicht angehakt; wiederholen Sie Schritt 1.
- **Ein Monitor wurde hinzugefügt oder ausgetauscht.** Wählen Sie die Monitore neu aus (Schritt 1).
- **Das Overlay öffnet sich nur auf einem Monitor.** Dasselbe: Schritt 1 muss für jeden Monitor durchgeführt werden.
- **Die Meldung sagt, das Portal habe abgelehnt.** Erteilen Sie die Berechtigung (Schritt 2).
- **„Clipboard error“ oder das Einfügen liefert nichts.** Prüfen Sie, ob `wl-copy` installiert ist (`wl-clipboard`).
- **Keine Benachrichtigungen.** Installieren Sie `libnotify-bin` (der Befehl `notify-send`) und prüfen Sie, ob Benachrichtigungen in den Einstellungen eingeschaltet sind.
- **Kein Tray-Symbol.** Aktivieren Sie die GNOME-Erweiterung AppIndicator.
- **`Print` öffnet das GNOME-Screenshot-Werkzeug.** Schalten Sie es in den Tastenkürzeln ab (siehe Schritt 3).

## Deinstallation

**apt oder `.deb`:**

```bash
sudo apt remove scrawl     # oder: sudo apt purge scrawl (das Paket hält keine Dateien außerhalb von /usr, beides ist also gleich)
sudo rm /etc/apt/sources.list.d/scrawl.sources   # optional: das Repository vergessen (scrawl.list bei der alten Form)
```

apt kann Ihr Home-Verzeichnis nicht anfassen, daher bleiben einige Reste pro Benutzer zurück: das GNOME-Tastenkürzel, der Autostart-Eintrag, falls Sie
ihn eingeschaltet hatten, und Ihre Einstellungen. Entfernen Sie das Tastenkürzel mit `gnome-hotkey.sh` von Scrawl (es liegt in `scripts/` des
Repositorys und im Release-Archiv) oder von Hand unter *Einstellungen → Tastatur → Tastenkürzel anzeigen und anpassen → Benutzerdefinierte
Tastenkürzel*; der Rest sind gewöhnliche Dateien:

```bash
sh gnome-hotkey.sh --remove
rm -f ~/.config/autostart/io.github.jexme.Scrawl.desktop
rm -rf ~/.config/scrawl ~/.cache/scrawl     # optional: Einstellungen und Cache
```

**Release-Archiv oder `make install`:** Führen Sie `./uninstall.sh` aus dem entpackten Archiv aus (`make uninstall` in einem Quellcode-
Checkout). Es entfernt die Programmdatei, die `.desktop`-Datei, das GNOME-Kürzel und den Autostart-Eintrag. Mit `--purge`
(`make uninstall PURGE=1`) löschen Sie zusätzlich die Einstellungen (`~/.config/scrawl`) und den Cache (`~/.cache/scrawl`). Geben Sie
dasselbe `PREFIX` an wie bei der Installation, falls Sie es geändert hatten. Ihre Screenshots in `~/Pictures/Screenshots` werden nie angetastet.

## So funktioniert es

Scrawl nimmt den Bildschirm über die ScreenCast-Schnittstelle von `xdg-desktop-portal` mit einem eigenen kleinen PipeWire-Client auf (`internal/pwcapture`). Ein gespeichertes Wiederherstellungs-Token pro Monitor erspart den Dialog. Der Compositor (Mutter) liefert Frames nur, wenn der Monitor neu zeichnet, und in unseren Tests lieferte er über einem Vollbildfenster (Direct Scanout) keinen; kommt innerhalb von 250 ms kein Frame, weicht Scrawl auf das langsamere Screenshot-Portal aus (etwa 0,5–1 s). Die Overlays sind Vollbildfenster von [Gio](https://gioui.org), eines pro Monitor; Gio ist gepatcht (`third_party/gio.patch`), damit ein Fenster auf einer bestimmten Wayland-Ausgabe erscheint. Das Ergebnis wird mit demselben Code gezeichnet, der es auf dem Bildschirm darstellt, daher entspricht das gespeicherte Bild dem, was Sie gesehen haben.

## Bauen und mitmachen

```bash
make build    # bin/scrawl; lädt Gio beim ersten Lauf herunter und patcht es
make test     # Unit-Tests, einschließlich des Tray-Symbols auf dem Sitzungs-D-Bus
make dist     # Release-Archiv und Prüfsumme in dist/
make deb      # .deb-Paket für Debian und Ubuntu in dist/
make notices  # THIRD_PARTY_NOTICES.md neu erzeugen
```

Die Oberflächentexte stehen in `internal/i18n/` (eine Datei pro Sprache, `lang_ru.go`, `lang_de.go` und so weiter; die Originaltexte im Code sind englisch);
`go test ./...` prüft, dass jeder Text in jede Sprache übersetzt ist. Wie man einen Fehler meldet und eine Änderung einreicht: [CONTRIBUTING.md](../../CONTRIBUTING.md). Plan und Ideen: [docs/PLAN.md](../PLAN.md) (auf Russisch).

## Lizenz

MIT, siehe [LICENSE](../../LICENSE). Komponenten Dritter: [THIRD_PARTY_NOTICES.md](../../THIRD_PARTY_NOTICES.md) (die Symbolformen stammen von [Tabler Icons](https://tabler.io/icons), MIT).
