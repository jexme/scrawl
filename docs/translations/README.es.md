# Scrawl

**Capturas de pantalla rápidas con anotaciones para GNOME en Wayland.** Pulsa una tecla, selecciona un área en cualquier monitor, dibuja flechas, recuadros y texto, difumina los datos privados, y el resultado ya está en el portapapeles y guardado como archivo.

[English](../../README.md) · [Русский](../../README.ru.md) · [简体中文](README.zh-CN.md) · **Español** · [Português (Brasil)](README.pt-BR.md) · [日本語](README.ja.md) · [Deutsch](README.de.md) · [Français](README.fr.md) · [한국어](README.ko.md)

![Superposición de Scrawl: un área seleccionada con un recuadro, una flecha, un texto, un trazo de marcador y una zona difuminada, y la barra de herramientas debajo](../images/overlay.png)

*La imagen es una renderización real de la superposición sobre un escritorio inventado.*

> La interfaz está disponible en inglés, ruso, chino simplificado, español, portugués, japonés, alemán, francés y coreano; los nombres de los botones de abajo son los de la interfaz en inglés, con sus equivalentes en cada idioma.

## Por qué Scrawl

En GNOME con Wayland, las aplicaciones no pueden capturar la pantalla directamente: se lo piden al portal del escritorio, y el portal **pregunta «¿qué pantalla quieres compartir?» en cada captura**. Scrawl pregunta una sola vez por monitor y recuerda la respuesta; después, una captura es una sola pulsación: la pantalla congelada aparece en unos 150 ms.

## Características

**Captura**
- Funciona en todos los monitores a la vez: la superposición se abre en cada uno y puedes seleccionar en cualquiera.
- Selección con marco discontinuo. Arrastra las esquinas para cambiar el tamaño; con la herramienta *seleccionar* arrastra un borde para cambiar el tamaño o el interior para mover el área.
- Recuerda la última área y la recupera con una tecla (`R`).
- Opcionalmente, incluye el cursor del ratón en la captura.

**Anotar**
- Lápiz, línea, flecha, recuadro, marcador, **desenfoque** y texto; ocho colores.
- La rueda del ratón (o los botones `−` / `+`) cambia el grosor, el tamaño del texto o la intensidad del desenfoque.
- Deshacer con `Ctrl+Z`.

**Guardar y compartir**
- `Enter` copia la imagen al portapapeles y guarda un PNG en `~/Pictures/Screenshots`. `Ctrl+C` solo copia, `Ctrl+S` solo guarda. También hay JPEG.
- **Colecciones antes/después:** aparta capturas con `A` y únelas en una sola imagen con títulos («Before», «After» o «Step 1…» si son más), una al lado de otra o una debajo de otra. O envía la colección como archivos separados que se pegan en un chat como adjuntos.

**A tu manera**
- Una ventana de ajustes para casi todo: teclas, colores, herramientas por defecto, aspecto de la selección, nombre y formato de los archivos e idioma de la interfaz (9 idiomas).
- Icono en la bandeja con inicio automático, un botón para asignar la tecla de acceso rápido y una comprobación de actualizaciones opcional (desactivada por defecto).

## Estado

Proyecto personal en desarrollo activo. **Todavía no hay ninguna versión publicada**, así que por ahora hay que compilarlo desde el código fuente (lleva unos minutos). El objetivo principal es GNOME en Wayland (Linux); las sesiones **X11** (GNOME sobre Xorg, KDE, XFCE y otras) también funcionan, pero ese soporte es nuevo y hasta ahora solo se ha probado en un servidor X virtual, así que se agradecen informes de máquinas reales. Existe una compilación para Windows, pero la captura de pantalla aún no está escrita; consulta [docs/WINDOWS.md](../WINDOWS.md).

## Requisitos

- GNOME en Wayland con PipeWire, `xdg-desktop-portal` (1.21 o superior, con soporte para registrar aplicaciones) y `xdg-desktop-portal-gnome`. Desarrollado y probado en Ubuntu con GNOME 50.
- `wl-clipboard` (el comando `wl-copy`) para el portapapeles y `libnotify` (el comando `notify-send`) para las notificaciones.
- En **X11** no hay portal: la pantalla se captura directamente y el portapapeles necesita `xclip` en lugar de `wl-clipboard`.

```bash
sudo apt install wl-clipboard libnotify-bin   # Wayland
sudo apt install xclip libnotify-bin          # X11
```

## Instalación

### 1. Obtener el programa

**Desde el repositorio apt** (Debian, Ubuntu; las actualizaciones llegan con `apt upgrade`):

```bash
curl -fsSL https://jexme.github.io/scrawl/scrawl-archive-keyring.gpg | sudo tee /usr/share/keyrings/scrawl-archive-keyring.gpg >/dev/null
echo "deb [signed-by=/usr/share/keyrings/scrawl-archive-keyring.gpg] https://jexme.github.io/scrawl stable main" | sudo tee /etc/apt/sources.list.d/scrawl.list
sudo apt update && sudo apt install scrawl
```

El repositorio está firmado con la clave `E62C DAF5 A1C9 0DB7 A899 D24D FA43 15C7 154A 9EF0`.

**Desde el código fuente (funciona hoy):**

```bash
sudo apt install gcc pkg-config patch libwayland-dev libx11-dev libx11-xcb-dev \
  libxkbcommon-x11-dev libgles2-mesa-dev libegl1-mesa-dev libffi-dev \
  libxcursor-dev libvulkan-dev libpipewire-0.3-dev
git clone https://github.com/jexme/scrawl.git && cd scrawl
make install            # compila e instala en ~/.local (PREFIX=/usr/local para todo el sistema)
```

Necesitas Go (la versión de `go.mod`). ¿Sin `sudo`? `scripts/fetch-pipewire-headers.sh` extrae las cabeceras de PipeWire en `.deps/` y el Makefile las usa.

**Desde un archivo de versión (cuando haya una publicada):** descarga `scrawl-VERSION-linux-amd64.tar.gz` y `SHA256SUMS` de la página de [Releases](https://github.com/jexme/scrawl/releases) y ejecuta:

```bash
sha256sum -c --ignore-missing SHA256SUMS
tar xzf scrawl-*-linux-amd64.tar.gz && cd scrawl-*-linux-amd64
./install.sh            # instala en ~/.local (PREFIX=/usr/local para todo el sistema)
sudo apt install libpipewire-0.3-0 libwayland-client0 libwayland-egl1 libwayland-cursor0 \
  libxkbcommon-x11-0 libegl1 libx11-6 libxcursor1
```

Comprueba que `~/.local/bin` está en tu `PATH`.

### 2. Primer uso: tres pasos, una sola vez

*En X11 no hacen falta los pasos 1 y 2 (no hay portal), solo la tecla de acceso rápido.*

Abre la ventana de ajustes (`scrawl -settings`, tab “System”) y usa estos botones, o ejecuta los comandos:

1. **Elegir los monitores:** botón «Pick monitors again» o `scrawl -setup`. GNOME muestra un diálogo **por monitor**. En cada uno elige un monitor *distinto*, deja marcada la casilla **«Recordar esta elección»** y acepta compartir. Después, no habrá más diálogos.
2. **Conceder el permiso de reserva:** botón «Grant permission» o `scrawl -grant-screenshot`. Permite a Scrawl usar el portal de capturas, más lento, en las raras ocasiones en que PipeWire no entrega imagen.
3. **Asignar una tecla de acceso rápido:** el campo «Launch hotkey» y el botón «Assign» (por defecto `Print`), o `scripts/gnome-hotkey.sh [TECLA]`. Crea un atajo personalizado de GNOME y nunca sobrescribe una combinación ocupada. Si `Print` sigue abriendo la herramienta de capturas de GNOME, desactívala en *Configuración → Teclado → Ver y personalizar los atajos → Capturas de pantalla*.

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
| `Esc` | cancelar |
| `R` | recuperar el área de la captura anterior |
| `A` | añadir la captura a la colección |
| `G` | unir la colección (y el área actual) en una sola imagen |
| `D` | guardar la colección como archivos separados y poner la lista de archivos en el portapapeles |
| rueda del ratón | grosor, tamaño del texto o intensidad del desenfoque de la herramienta actual |

Todas las teclas se pueden cambiar en la ventana de ajustes («Keys»).

### Barra de herramientas

De izquierda a derecha: seleccionar, lápiz, línea, flecha, recuadro, marcador, desenfoque, texto · ocho colores · `−` grosor `+` · ajustes (cierra la superposición y abre la ventana de ajustes) · añadir a la colección · deshacer · copiar · guardar · cerrar.

### Colecciones antes/después

Haz la primera captura y pulsa `A`. Haz la siguiente: el área anterior ya está seleccionada, así que fotografías el mismo sitio otra vez. Pulsa `G` para unir las capturas en una imagen (se guarda y se copia) o `D` para obtener archivos separados. El pequeño panel de la parte inferior de la pantalla muestra lo que hay en la colección y ofrece las mismas acciones como botones.

## Icono en la bandeja e inicio automático

`scrawl -tray` mantiene un icono en el área de notificaciones: el clic izquierdo hace una captura y el menú tiene «Screenshot», «Settings» y «Quit». Para iniciarlo al entrar en la sesión, activa *Settings → System → Autostart*. En GNOME el icono necesita la extensión AppIndicator (activada por defecto en Ubuntu).

## Ajustes

![La ventana de ajustes](../images/settings-en.png)

Ábrela desde la barra de herramientas (engranaje), con `scrawl -settings` o desde el menú de aplicaciones. Los cambios se guardan al instante en `~/.config/scrawl/config.json`, un archivo JSON normal que también puedes editar a mano. Los valores que faltan o no son válidos vuelven a los valores por defecto. Cada clave está descrita en la [referencia de ajustes](../SETTINGS.md) (en inglés o [ruso](../SETTINGS.ru.md)).

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

## Solución de problemas

- **No pasa nada al pulsar la tecla.** Puede que `xdg-desktop-portal` se haya colgado. Reinícialo: `systemctl --user restart xdg-desktop-portal xdg-desktop-portal-gnome`.
- **GNOME pregunta qué pantalla compartir en cada captura.** No se marcó «Recordar esta elección»; repite el paso 1.
- **Se añadió o cambió un monitor.** Elige los monitores otra vez (paso 1).
- **La superposición solo se abre en un monitor.** Igual: el paso 1 hay que hacerlo para cada monitor.
- **El mensaje dice que el portal se negó.** Concede el permiso (paso 2).
- **Al pegar en alguna aplicación no aparece nada.** Comprueba que `wl-copy` está instalado (paquete `wl-clipboard`).
- **No hay notificaciones.** Instala `libnotify-bin` (el comando `notify-send`) y comprueba que las notificaciones están activadas en los ajustes.
- **No hay icono en la bandeja.** Activa la extensión AppIndicator de GNOME.
- **`Print` abre la herramienta de capturas de GNOME.** Desactívala en los atajos de teclado (consulta el paso 3).

## Cómo funciona

Scrawl captura la pantalla mediante la interfaz ScreenCast de `xdg-desktop-portal` con su propio cliente PipeWire (`internal/pwcapture`). Un token de restauración guardado para cada monitor elimina el diálogo. El compositor (Mutter) entrega imágenes solo cuando el monitor se redibuja y, en nuestras pruebas, no entregó ninguna sobre una ventana a pantalla completa (salida directa); si no llega ninguna imagen en 250 ms, Scrawl recurre al portal de capturas, que es más lento (unos 0,5-1 s). Las superposiciones son ventanas de [Gio](https://gioui.org) a pantalla completa, una por monitor; Gio lleva un parche (`third_party/gio.patch`) para colocar una ventana en una salida concreta de Wayland. El resultado se dibuja con el mismo código que lo muestra en pantalla, así que la imagen guardada coincide con lo que viste.

## Compilar y contribuir

```bash
make build    # bin/scrawl; descarga y parchea Gio la primera vez
make test     # pruebas unitarias, incluido el icono de la bandeja en el D-Bus de sesión
make dist     # archivo de versión y suma de comprobación en dist/
make notices  # regenera THIRD_PARTY_NOTICES.md
```

Los textos de la interfaz están en `internal/i18n/` (un archivo por idioma; los textos originales del código están en ruso); `go test ./...` comprueba que cada texto esté traducido a cada idioma. Plan e ideas: [docs/PLAN.md](../PLAN.md) (en ruso).

## Licencia

MIT, consulta [LICENSE](../../LICENSE). Componentes de terceros: [THIRD_PARTY_NOTICES.md](../../THIRD_PARTY_NOTICES.md) (las formas de los iconos son de [Tabler Icons](https://tabler.io/icons), MIT).
