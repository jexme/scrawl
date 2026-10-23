# Scrawl

**Capturas de pantalla rápidas con anotaciones para GNOME en Wayland.** Pulsa una tecla, selecciona un área en cualquier monitor, dibuja flechas, recuadros y texto, difumina los datos privados, y el resultado ya está en el portapapeles y guardado como archivo.

[English](../../README.md) · [Русский](README.ru.md) · [简体中文](README.zh-CN.md) · **Español** · [Português (Brasil)](README.pt-BR.md) · [日本語](README.ja.md) · [Deutsch](README.de.md) · [Français](README.fr.md) · [한국어](README.ko.md)

![Superposición de Scrawl: un área seleccionada con un recuadro, una flecha, un texto, un trazo de marcador y una zona difuminada, y la barra de herramientas debajo](../images/overlay.png)

*La imagen es una renderización real de la superposición sobre un escritorio inventado.*

> La interfaz está disponible en inglés, ruso, chino simplificado, español, portugués, japonés, alemán, francés y coreano; los nombres de los botones de abajo son los de la interfaz en español.

## Por qué Scrawl

En GNOME con Wayland, las aplicaciones no pueden capturar la pantalla directamente: se lo piden al portal del escritorio, y el portal **pregunta «¿qué pantalla quieres compartir?» en cada captura**. Scrawl pregunta una sola vez por monitor y recuerda la respuesta; después, una captura es una sola pulsación: la pantalla congelada aparece en unos 150 ms.

## Inicio rápido

En Ubuntu o Debian (Wayland o X11) son tres pasos.

**1. Instala** desde el repositorio apt firmado (las actualizaciones llegan después con `apt upgrade`):

```bash
sudo curl -fsSL https://jexme.github.io/scrawl/scrawl.sources -o /etc/apt/sources.list.d/scrawl.sources
sudo apt update && sudo apt install scrawl
```

El archivo `.sources` lleva dentro la clave de firma, así que no hay que gestionar ningún llavero aparte. Los paquetes se
compilan para **amd64** y **arm64** (la de arm64 es de mejor esfuerzo); en otras arquitecturas [compila desde el código fuente](#instalación).

**2. Configura una sola vez:** inicia Scrawl desde el menú de aplicaciones (o ejecuta `scrawl`): el primer arranque abre una breve guía
en la pestaña «Sistema» (puedes abrirla cuando quieras con `scrawl -settings`). En Wayland pulsa «Configurar la captura»
(GNOME pregunta una vez por monitor: elige un monitor distinto en cada diálogo y marca «Recordar esta elección»). Después escribe una tecla de acceso rápido
(por defecto `Print`) y pulsa «Asignar». La herramienta de capturas de GNOME también usa `Print`: pulsa «Liberar la tecla» para
desactivar ese atajo integrado (solo esa tecla, y puedes deshacerlo). En X11 solo hace falta la tecla de acceso rápido.

**3. Haz una captura:** pulsa la tecla de acceso rápido, arrastra un área y pulsa `Enter`. La imagen queda en el portapapeles y en
`~/Pictures/Screenshots`.

Otros sistemas: [Windows](#windows-10-y-11), otras distribuciones de Linux (un archivo de versión o una compilación desde el código fuente), consulta
[Instalación](#instalación) más abajo.

## Características

**Captura**
- Funciona en todos los monitores a la vez: la superposición se abre en cada uno y puedes seleccionar en cualquiera.
- Selección con marco discontinuo. Arrastra las esquinas para cambiar el tamaño; con la herramienta *seleccionar* arrastra un borde para cambiar el tamaño o el interior para mover el área.
- Mientras arrastras un marco, una lupa junto al cursor muestra los píxeles 6 veces más grandes, con el borde del marco
  sobre la cuadrícula de píxeles, para que puedas alinearlo con exactitud (se puede desactivar en los ajustes).
- Recuerda la última área y la recupera con una tecla (`R`).
- Opcionalmente, incluye el cursor del ratón en la captura (por ahora solo en Wayland).

**Anotar**
- Lápiz, línea, flecha, recuadro, marcador, **desenfoque**, texto y **sellos numerados** (un clic coloca un círculo con 1, 2, 3 y así sucesivamente, como en flameshot); ocho colores.
- La rueda del ratón (o los botones `−` / `+`) cambia el grosor, el tamaño del texto o la intensidad del desenfoque.
- El **desenfoque** tiene seis niveles (rueda o `−` / `+`): 1-3 desenfoque gaussiano suave, 4-6 mosaico con cuadrados cada vez más grandes. Las celdas del mosaico empiezan en la esquina del área, y la nota que aparece al cambiar de nivel indica el modo.
- Mantén pulsada **Shift** mientras dibujas: el lápiz y el marcador trazan una línea recta, una línea o una flecha se ajusta a 45°, un recuadro o un desenfoque se vuelve un cuadrado.
- Deshacer con `Ctrl+Z`.

**Guardar y compartir**
- `Enter` copia la imagen al portapapeles y guarda un PNG en `~/Pictures/Screenshots`. `Ctrl+C` solo copia, `Ctrl+S` solo guarda. También hay JPEG.
- Un **doble clic** dentro del área seleccionada la copia de inmediato (en los ajustes, pestaña «General», sección «Acciones», también puede copiar y guardar, o no hacer nada).
- **Colecciones antes/después:** aparta capturas con `A` y únelas en una sola imagen con títulos («Before», «After» o «Step 1…» si son más), una al lado de otra o una debajo de otra. O envía la colección como archivos separados que se pegan en un chat como adjuntos.

**A tu manera**
- Una ventana de ajustes para casi todo: teclas, colores, herramientas por defecto, aspecto de la selección, nombre y formato de los archivos e idioma de la interfaz (9 idiomas: inglés, ruso, chino, español, portugués, japonés, alemán, francés, coreano).
- Icono en la bandeja con inicio automático, un botón para asignar la tecla de acceso rápido y una comprobación de actualizaciones opcional (desactivada por defecto).

## Estado

Proyecto personal en desarrollo activo; la versión actual es una versión preliminar (`v0.1.0-rc4`). Los paquetes ya hechos están en la página de
[Releases](https://github.com/jexme/scrawl/releases); también puedes compilarlo desde el código fuente (lleva unos minutos).
Linux: el objetivo principal es GNOME en Wayland. Las sesiones **X11** (GNOME sobre Xorg, KDE,
XFCE y otras) también funcionan, pero ese soporte es nuevo y hasta ahora solo se ha probado en un
servidor X virtual, así que se agradecen informes de máquinas reales. **Windows 10 y 11** también funcionan (captura, anotaciones,
portapapeles, icono en la bandeja y una tecla de acceso rápido global); este port es nuevo y solo se ha probado con una escala del 100 %, consulta
[docs/WINDOWS.md](../WINDOWS.md) para compilar y saber qué falta.

## Requisitos

La instalación con apt o con el `.deb` los incorpora por ti; la lista es para el archivo de versión y para la compilación desde el código fuente.

- GNOME en Wayland con PipeWire, `xdg-desktop-portal` (1.21 o superior, con soporte para registrar aplicaciones) y `xdg-desktop-portal-gnome`. Desarrollado y probado en Ubuntu con GNOME 50.
- `wl-clipboard` (el comando `wl-copy`) para el portapapeles y `libnotify` (el comando `notify-send`) para las notificaciones. Sin `wl-copy`, Scrawl recurre a
  `xclip` a través de XWayland si están ambos; si no hay ninguno, recibes una indicación de qué instalar.
- En **X11** no hay portal: la pantalla se captura directamente y el portapapeles necesita `xclip` en lugar de `wl-clipboard`.

```bash
sudo apt install wl-clipboard libnotify-bin   # Wayland
sudo apt install xclip libnotify-bin          # X11
```

## Instalación

### 1. Obtener el programa

**Desde el repositorio apt** (Debian, Ubuntu; las actualizaciones llegan con `apt upgrade`):

```bash
sudo curl -fsSL https://jexme.github.io/scrawl/scrawl.sources -o /etc/apt/sources.list.d/scrawl.sources
sudo apt update && sudo apt install scrawl
```

Los paquetes se compilan para amd64 y arm64 (la de arm64 es de mejor esfuerzo; en otras arquitecturas: compila desde el código fuente, ver más abajo). El repositorio está firmado con la
clave `E62C DAF5 A1C9 0DB7 A899 D24D FA43 15C7 154A 9EF0`, que va incluida en `scrawl.sources`.

En sistemas más antiguos (Ubuntu 20.04, Debian 11: un apt anterior a 2.4 no puede leer una clave dentro de un archivo `.sources`) usa en su lugar un
llavero aparte y una fuente de una sola línea:

```bash
curl -fsSL https://jexme.github.io/scrawl/scrawl-archive-keyring.gpg | sudo tee /usr/share/keyrings/scrawl-archive-keyring.gpg >/dev/null
echo "deb [signed-by=/usr/share/keyrings/scrawl-archive-keyring.gpg] https://jexme.github.io/scrawl stable main" | sudo tee /etc/apt/sources.list.d/scrawl.list
sudo apt update && sudo apt install scrawl
```

La clave también se publica en formato ASCII-armored como `scrawl-archive-keyring.asc`.

**Desde un paquete .deb** (Debian, Ubuntu): descarga `scrawl_VERSION_amd64.deb`
de la página de [Releases](https://github.com/jexme/scrawl/releases) y luego
`sudo apt install ./scrawl_*.deb`. Instala las bibliotecas necesarias y recomienda `wl-clipboard`, `xclip`
y `libnotify-bin`. El paquete no asigna la tecla de acceso rápido (es por usuario): hazlo en `scrawl -settings`.
Para compilarlo tú mismo: `make deb`.

**Desde un archivo de versión**: descarga
`scrawl-VERSION-linux-amd64.tar.gz` y `SHA256SUMS` de la página de
[Releases](https://github.com/jexme/scrawl/releases) y ejecuta:

```bash
sha256sum -c --ignore-missing SHA256SUMS   # comprueba cada archivo descargado que figura en SHA256SUMS
tar xzf scrawl-*-linux-amd64.tar.gz && cd scrawl-*-linux-amd64
./install.sh            # instala en ~/.local (PREFIX=/usr/local para todo el sistema)
sudo apt install libpipewire-0.3-0 libwayland-client0 libwayland-egl1 libwayland-cursor0 \
  libxkbcommon-x11-0 libegl1 libx11-6 libxcursor1 wl-clipboard libnotify-bin
```

`wl-clipboard` (el portapapeles) y `libnotify-bin` (las notificaciones) también hacen falta en tiempo de ejecución: el
instalador los comprueba e indica lo que falta. En X11 usa `xclip` en lugar de `wl-clipboard`.

**Desde el código fuente**:

```bash
sudo apt install gcc pkg-config patch libwayland-dev libx11-dev libx11-xcb-dev \
  libxkbcommon-x11-dev libgles2-mesa-dev libegl1-mesa-dev libffi-dev \
  libxcursor-dev libvulkan-dev libpipewire-0.3-dev
git clone https://github.com/jexme/scrawl.git && cd scrawl
make install            # compila e instala en ~/.local (PREFIX=/usr/local para todo el sistema)
```

Necesitas Go (la versión de `go.mod`). ¿Sin `sudo`? `scripts/fetch-pipewire-headers.sh` extrae las cabeceras de PipeWire en `.deps/` y el Makefile las usa.

Comprueba que `~/.local/bin` está en tu `PATH`.

### Windows 10 y 11

**Recomendado: el instalador.** Descarga `scrawl-VERSION-windows-amd64-setup.exe` de la página de
[Releases](https://github.com/jexme/scrawl/releases) y ejecútalo. Se instala solo para tu cuenta (sin derechos de administrador),
añade una entrada al menú Inicio, puede iniciar Scrawl al entrar en la sesión y lo inicia al terminar. Para actualizar, ejecuta el
instalador más reciente: cierra primero el Scrawl antiguo. Si una versión no trae archivo de instalación, usa una de las opciones siguientes.

Alternativas:

- **Zip portátil.** Descarga `scrawl-VERSION-windows-amd64.zip` y su `.sha256`, descomprímelo en una carpeta permanente (no
  directamente desde el visor del archivo comprimido) y ejecuta `scrawl.exe`. Al hacer doble clic en `scrawl.exe` se inicia el icono del área de notificación.
  También se publica un único `scrawl-VERSION-windows-amd64.exe`. Para actualizar, cierra Scrawl desde el menú de su icono y sustituye el archivo.
- **[Scoop](https://scoop.sh):** `scoop install https://github.com/jexme/scrawl/releases/download/v0.1.0-rc4/scrawl.json`.
  Esa dirección fija una versión preliminar concreta, así que entre una versión candidata y otra se actualiza con `scoop uninstall scrawl` y `scoop install`
  con la URL de la nueva versión (hasta que haya una versión estable o un bucket). Cuando el manifiesto se sirva desde
  `https://github.com/jexme/scrawl/releases/latest/download/scrawl.json` (exista una versión estable), `scoop update scrawl` funcionará.

- **El icono** está en el área de notificación; en Windows 11 puede esconderse tras la flecha `^` junto al reloj: arrástralo fuera para mantenerlo visible.
- **La tecla Print** es la tecla de acceso rápido por defecto. A menudo otro programa la tiene ocupada (Greenshot, ShareX, Lightshot). Windows mismo también puede:
  Configuración, Accesibilidad, Teclado, *Usar el botón Imprimir pantalla para abrir la captura de pantalla*: desactívalo. O elige otra tecla en
  `scrawl.exe -settings`, por ejemplo `<Control><Alt>a`. El icono avisa cuando la tecla está ocupada y sigue intentándolo.
- **«Windows protegió tu PC»** (SmartScreen) aparece porque el programa aún no está firmado: haz clic en *Más información* y luego en
  *Ejecutar de todas formas*. Para comprobar antes el archivo, compara `Get-FileHash` con el archivo `.sha256`, o ejecuta
  `gh attestation verify scrawl-*-setup.exe --repo jexme/scrawl` (la compilación procede de una CI pública). Los antivirus pueden marcar
  cualquier programa nuevo en Go sin firmar; informa de ese caso como una incidencia (issue).
- **El inicio automático** es la casilla en `scrawl.exe -settings`, pestaña *Sistema* (un valor en `HKCU\...\Run`), o una casilla del instalador.
  Si mueves o actualizas el programa, el siguiente inicio del icono reescribe la entrada con la nueva ubicación (un inicio automático desactivado sigue desactivado).
- **Iniciar una versión más nueva mientras se ejecuta una más antigua:** Scrawl te dice qué versión se está ejecutando; ciérrala primero desde el
  menú del icono y luego inicia la nueva.

Compilar desde el código fuente y qué falta: [docs/WINDOWS.md](../WINDOWS.md).

### 2. Primer uso: tres pasos, una sola vez

*En X11 no hacen falta los pasos 1 y 2 (no hay portal), solo la tecla de acceso rápido.*

Abre la ventana de ajustes (`scrawl -settings`, pestaña «Sistema»). El botón «Configurar la captura» hace los pasos 1 y 2 de una vez (en el primer arranque la ventana se abre sola). O usa los botones por separado o ejecuta los comandos:

1. **Elegir los monitores:** botón «Elegir monitores de nuevo» o `scrawl -setup`. GNOME muestra un diálogo **por monitor**. En cada uno elige un monitor *distinto*, deja marcada la casilla **«Recordar esta elección»** y acepta compartir. Después, no habrá más diálogos.
2. **Conceder el permiso de reserva:** botón «Conceder permiso» o `scrawl -grant-screenshot`. Permite a Scrawl usar el portal de capturas, más lento, en las raras ocasiones en que PipeWire no entrega imagen.
3. **Asignar una tecla de acceso rápido:** en GNOME, el instalador del archivo de versión (`install.sh`) y `make install` asignan `Print` por sí mismos, salvo que otro atajo personalizado ya la use. Tras instalar con apt o con el `.deb`, o para cambiar la tecla: el campo «Tecla de lanzamiento» y el botón «Asignar» (por defecto `Print`), o `scripts/gnome-hotkey.sh [TECLA]`. Crea un atajo personalizado de GNOME llamado «scrawl» (o lo actualiza si existe, así que ejecutarlo de nuevo nunca crea un segundo atajo) y nunca sobrescribe el atajo de otra persona si la combinación está ocupada. Si `Print` sigue abriendo la herramienta de capturas de GNOME, desactívala en *Configuración → Teclado → Ver y personalizar los atajos → Capturas de pantalla*.

### Actualizar desde screen-editor

Scrawl es el nuevo nombre de screen-editor. Tus ajustes se leen de `~/.config/screen-editor/config.json` hasta que exista `~/.config/scrawl/config.json` (el archivo antiguo no se toca). El portal asocia la elección de monitores y los permisos al identificador de la aplicación, que ha cambiado, así que repite una vez los pasos 1 a 3 y apunta tu atajo a `scrawl`. La última área y la colección empiezan de cero.

## Uso

1. Pulsa tu tecla de acceso rápido. La pantalla se congela y se oscurece.
2. Arrastra para seleccionar un área (en cualquier monitor). Debajo aparece una barra de herramientas.
3. Si quieres, anota: elige una herramienta y un color, y dibuja sobre el área.
4. Pulsa `Enter`. La imagen está en el portapapeles y en `~/Pictures/Screenshots`; pégala donde quieras.

`Esc` cancela en cualquier momento.

### Teclas

| Tecla | Acción |
| --- | --- |
| `Enter` | copiar al portapapeles **y** guardar en un archivo (lo que hace es configurable) |
| `Ctrl+C` / `Ctrl+S` | solo copiar / solo guardar |
| `Ctrl+Z` | deshacer la última anotación |
| `Esc` | cancelar (primero cierra un menú abierto; mientras escribes un texto, descarta solo el texto) |
| `Ctrl+A` | seleccionar la pantalla completa del monitor (no se puede reasignar) |
| `R` | recuperar el área de la captura anterior |
| `V` `P` `L` `W` `B` `M` `U` `T` `N` | herramientas: seleccionar, lápiz, línea, flecha, recuadro, marcador, desenfoque, texto, sello numerado (cuando hay un área seleccionada) |
| `+` / `-` | grosor, tamaño del texto o intensidad del desenfoque de la herramienta actual (como la rueda del ratón) |
| `A` | añadir la captura a la colección |
| `G` | unir la colección (y el área actual) en una sola imagen |
| `D` | guardar la colección como archivos separados y poner la lista de archivos en el portapapeles |
| rueda del ratón | grosor, tamaño del texto o intensidad del desenfoque de la herramienta actual |
| `Shift` al dibujar | lápiz y marcador: una línea recta; línea y flecha: se ajustan a 45°; recuadro y desenfoque: un cuadrado. `Shift` al seleccionar un área: un cuadrado |

Todas las teclas se pueden cambiar en la ventana de ajustes (pestaña «Teclas»).

### Barra de herramientas

La fila principal, de izquierda a derecha: seleccionar, lápiz, línea, flecha, recuadro, marcador, desenfoque, texto, sello
numerado · deshacer · copiar · guardar · el menú `⋯` (ajustes, que cierra la superposición y abre la ventana
de ajustes, y añadir a la colección) · cerrar.

Mientras hay una herramienta de dibujo activa, debajo aparece una segunda fila: ocho colores, un botón arcoíris para
tu propio color (abre una paleta; el color se recuerda) y `−` grosor `+`. La herramienta «seleccionar»
no tiene ninguno de los dos, así que la fila se oculta.

Los botones que tienen una tecla la muestran bajo el icono (las teclas vienen de los ajustes). Mantén el cursor un momento sobre un botón para ver su nombre y su tecla. Los atajos funcionan en cualquier distribución de teclado (por ejemplo, una distribución rusa), según la posición de la tecla.

### Colecciones antes/después

Haz la primera captura y pulsa `A`. Haz la siguiente: el área anterior ya está seleccionada, así que fotografías el mismo sitio otra vez. Pulsa `G` para unir las capturas en una imagen (se guarda y se copia) o `D` para obtener archivos separados. El pequeño panel de la parte inferior de la pantalla muestra lo que hay en la colección y ofrece las mismas acciones como botones.

## Icono en la bandeja e inicio automático

`scrawl -tray` mantiene un icono en el área de notificaciones: el clic izquierdo hace una captura y el menú tiene «Hacer una captura», «Ajustes» y «Salir». Para iniciarlo al entrar en la sesión, activa *Ajustes → Sistema → Inicio automático*. En GNOME el icono necesita la extensión AppIndicator (activada por defecto en Ubuntu).

## Ajustes

![La ventana de ajustes](../images/settings-en.png)

Ábrela desde la barra de herramientas (engranaje), con `scrawl -settings` o desde el menú de aplicaciones. Los cambios se guardan al instante en `~/.config/scrawl/config.json`, un archivo JSON normal que también puedes editar a mano. Los valores que faltan o no son válidos vuelven a los valores por defecto. Cada clave está descrita en la [referencia de ajustes](../SETTINGS.md) (en inglés o [ruso](SETTINGS.ru.md)).

## Archivos y privacidad

- Las imágenes van a `~/Pictures/Screenshots` (se cambia en los ajustes) con nombres como `scrawl-20260131-142501.png`.
- Ajustes: `~/.config/scrawl/`. Tokens de los monitores, última área y colección: `~/.cache/scrawl/`.
- Scrawl nunca se conecta a internet por su cuenta. La única petición de red del programa es la comprobación de actualizaciones, opcional y desactivada por defecto: pregunta a GitHub por la última versión y, además de las cabeceras HTTP habituales, solo envía la versión del programa.
- Las capturas no salen de tu ordenador a menos que las pegues en otro sitio.

## Línea de comandos

| Opción | Qué hace |
| --- | --- |
| `-settings` | abre la ventana de ajustes |
| `-setup` | elige de nuevo los monitores (diálogos de GNOME) |
| `-grant-screenshot` | concede el permiso de captura de reserva y sale |
| `-tray` | mantiene el icono en el área de notificaciones |
| `-fresh` | no ofrece el área de la captura anterior |
| `-v` | muestra los tiempos |
| `-version` | muestra la versión y sale |
| `-exit-after DURATION` | sale normalmente tras el tiempo indicado, para pruebas (por ejemplo `-exit-after 3s`) |

## Solución de problemas

- **No pasa nada al pulsar la tecla.** Puede que `xdg-desktop-portal` se haya colgado. Reinícialo: `systemctl --user restart xdg-desktop-portal xdg-desktop-portal-gnome`.
- **GNOME pregunta qué pantalla compartir en cada captura.** No se marcó «Recordar esta elección»; repite el paso 1.
- **Se añadió o cambió un monitor.** Elige los monitores otra vez (paso 1).
- **La superposición solo se abre en un monitor.** Igual: el paso 1 hay que hacerlo para cada monitor.
- **El mensaje dice que el portal se negó.** Concede el permiso (paso 2).
- **«Clipboard error» o al pegar no aparece nada.** Comprueba que `wl-copy` está instalado (paquete `wl-clipboard`).
- **No hay notificaciones.** Instala `libnotify-bin` (el comando `notify-send`) y comprueba que las notificaciones están activadas en los ajustes.
- **No hay icono en la bandeja.** Activa la extensión AppIndicator de GNOME.
- **`Print` abre la herramienta de capturas de GNOME.** Desactívala en los atajos de teclado (consulta el paso 3).

## Desinstalar

**apt o `.deb`:**

```bash
sudo apt remove scrawl     # o: sudo apt purge scrawl (el paquete no deja archivos fuera de /usr, así que es lo mismo)
sudo rm /etc/apt/sources.list.d/scrawl.sources   # opcional: olvidar el repositorio (scrawl.list en la forma antigua)
```

apt no puede tocar tu carpeta personal, así que quedan algunos restos por usuario: el atajo de GNOME, la entrada de inicio automático si
la activaste, y tus ajustes. Elimina el atajo con el `gnome-hotkey.sh` de Scrawl (está en `scripts/` del
repositorio y en el archivo de versión) o a mano en *Configuración → Teclado → Ver y personalizar los atajos → Atajos
personalizados*; lo demás son archivos normales:

```bash
sh gnome-hotkey.sh --remove
rm -f ~/.config/autostart/io.github.jexme.Scrawl.desktop
rm -rf ~/.config/scrawl ~/.cache/scrawl     # opcional: ajustes y caché
```

**Archivo de versión o `make install`:** ejecuta `./uninstall.sh` desde el archivo descomprimido (`make uninstall` en una copia del
código fuente). Elimina el binario, el archivo `.desktop`, el atajo de GNOME y la entrada de inicio automático. Añade `--purge`
(`make uninstall PURGE=1`) para borrar también los ajustes (`~/.config/scrawl`) y la caché (`~/.cache/scrawl`). Pasa
el mismo `PREFIX` que usaste al instalar, si lo cambiaste. Tus capturas en `~/Pictures/Screenshots` nunca se tocan.

## Cómo funciona

Scrawl captura la pantalla mediante la interfaz ScreenCast de `xdg-desktop-portal` con su propio cliente PipeWire (`internal/pwcapture`). Un token de restauración guardado para cada monitor elimina el diálogo. El compositor (Mutter) entrega imágenes solo cuando el monitor se redibuja y, en nuestras pruebas, no entregó ninguna sobre una ventana a pantalla completa (salida directa); si no llega ninguna imagen en 250 ms, Scrawl recurre al portal de capturas, que es más lento (unos 0,5-1 s). Las superposiciones son ventanas de [Gio](https://gioui.org) a pantalla completa, una por monitor; Gio lleva un parche (`third_party/gio.patch`) para colocar una ventana en una salida concreta de Wayland. El resultado se dibuja con el mismo código que lo muestra en pantalla, así que la imagen guardada coincide con lo que viste.

## Compilar y contribuir

```bash
make build    # bin/scrawl; descarga y parchea Gio la primera vez
make test     # pruebas unitarias, incluido el icono de la bandeja en el D-Bus de sesión
make dist     # archivo de versión y suma de comprobación en dist/
make deb      # paquete .deb para Debian y Ubuntu en dist/
make notices  # regenera THIRD_PARTY_NOTICES.md
```

Los textos de la interfaz están en `internal/i18n/` (un archivo por idioma, `lang_ru.go`, `lang_de.go` y así sucesivamente; los textos originales del código están en inglés);
`go test ./...` comprueba que cada texto esté traducido a cada idioma. Cómo informar de un error y enviar un cambio: [CONTRIBUTING.md](../../CONTRIBUTING.md). Plan e ideas: [docs/PLAN.md](../PLAN.md) (en ruso).

## Licencia

MIT, consulta [LICENSE](../../LICENSE). Componentes de terceros: [THIRD_PARTY_NOTICES.md](../../THIRD_PARTY_NOTICES.md) (las formas de los iconos son de [Tabler Icons](https://tabler.io/icons), MIT).
