# Scrawl

**Captures d'écran rapides avec annotations pour GNOME sous Wayland.** Appuyez sur une touche, sélectionnez une zone sur n'importe quel écran, dessinez des flèches, des cadres et du texte, floutez les données privées, et le résultat est déjà dans le presse-papiers et enregistré dans un fichier.

[English](../../README.md) · [Русский](README.ru.md) · [简体中文](README.zh-CN.md) · [Español](README.es.md) · [Português (Brasil)](README.pt-BR.md) · [日本語](README.ja.md) · [Deutsch](README.de.md) · **Français** · [한국어](README.ko.md)

![Surcouche Scrawl : une zone sélectionnée avec un cadre, une flèche, un texte, un trait de surligneur et une partie floutée, la barre d'outils en dessous](../images/overlay.png)

*L'image est un vrai rendu de la surcouche sur un bureau imaginaire.*

> L'interface est disponible en anglais, russe, chinois simplifié, espagnol, portugais, japonais, allemand, français et coréen ; les noms de boutons ci-dessous sont ceux de l'interface en français.

## Pourquoi Scrawl

Sous GNOME avec Wayland, les applications ne peuvent pas capturer l'écran directement : elles le demandent au portail du bureau, et le portail **demande « Quel écran voulez-vous partager ? » à chaque capture**. Scrawl ne le demande qu'une fois par écran et retient la réponse ; ensuite, une capture se fait d'une seule pression : l'écran figé apparaît en environ 150 ms.

## Démarrage rapide

Sous Ubuntu ou Debian (Wayland ou X11), il suffit de trois étapes.

**1. Installer** depuis le dépôt apt signé (les mises à jour arrivent ensuite avec `apt upgrade`) :

```bash
sudo curl -fsSL https://jexme.github.io/scrawl/scrawl.sources -o /etc/apt/sources.list.d/scrawl.sources
sudo apt update && sudo apt install scrawl
```

Le fichier `.sources` contient lui-même la clé de signature, il n'y a donc pas de trousseau séparé à gérer. Les paquets sont
construits pour **amd64** et **arm64** (la version arm64 est faite au mieux) ; sur les autres architectures, [compilez depuis les sources](#installation).

**2. Régler une fois :** lancez Scrawl depuis le menu des applications (ou exécutez `scrawl`) : le premier lancement ouvre un court guide
dans l'onglet « Système » (vous pouvez l'ouvrir à tout moment avec `scrawl -settings`). Sous Wayland, appuyez sur « Régler la capture »
(GNOME demande une fois par écran : choisissez un écran différent dans chaque boîte de dialogue et cochez « Se souvenir de ce choix »). Saisissez ensuite un raccourci
(`Print` par défaut) et appuyez sur « Attribuer ». L'outil de capture de GNOME utilise aussi `Print` : appuyez sur « Libérer la touche » pour
désactiver ce raccourci intégré (uniquement cette touche, et vous pouvez l'annuler). Sous X11, seul le raccourci est nécessaire.

**3. Faire une capture :** appuyez sur le raccourci, tracez une zone, appuyez sur `Enter`. L'image est dans le presse-papiers et dans
`~/Pictures/Screenshots`.

Autres systèmes : [Windows](#windows-10-et-11), autres distributions Linux (une archive de version ou une compilation depuis les sources), voir
[Installation](#installation) plus bas.

## Fonctionnalités

**Capture**
- Fonctionne sur tous les écrans à la fois : la surcouche s'ouvre sur chacun et vous sélectionnez sur n'importe lequel.
- Sélection avec un cadre en pointillés. Faites glisser les coins pour redimensionner ; avec l'outil *sélection*, faites glisser un bord pour redimensionner ou l'intérieur pour déplacer la zone.
- Pendant que vous tirez un cadre, une loupe à côté du curseur montre les pixels agrandis 6 fois, avec le bord du cadre sur la grille de pixels, pour que vous puissiez l'aligner exactement (elle peut être désactivée dans les réglages).
- Retient la dernière zone et la ramène d'une touche (`R`).
- En option, le curseur de la souris apparaît sur la capture (pour l'instant sous Wayland uniquement).

**Annotations**
- Crayon, ligne, flèche, cadre, surligneur, **flou**, texte et **tampons numérotés** (un clic place un cercle avec 1, 2, 3 et ainsi de suite, comme dans flameshot) ; huit couleurs.
- La molette de la souris (ou les boutons `−` / `+`) règle l'épaisseur, la taille du texte ou l'intensité du flou.
- Le **flou** a six niveaux (molette ou `−` / `+`) : 1 à 3 flou gaussien doux, 4 à 6 mosaïque avec des carrés de plus en plus grands. Les cellules de la mosaïque partent du coin de la zone, et la note qui s'affiche quand vous changez de niveau indique le mode.
- Maintenez **Shift** pendant le dessin : le crayon et le surligneur tracent une ligne droite, une ligne ou une flèche s'aligne sur 45°, un cadre ou un flou devient un carré.
- Annulation avec `Ctrl+Z`.

**Enregistrer et partager**
- `Enter` copie l'image dans le presse-papiers et enregistre un PNG dans `~/Pictures/Screenshots`. `Ctrl+C` copie seulement, `Ctrl+S` enregistre seulement. Le JPEG est aussi possible.
- Un **double-clic** dans la zone sélectionnée la copie aussitôt (dans les réglages, onglet « Général », section « Actions », il peut aussi copier et enregistrer, ou ne rien faire).
- **Collections avant/après :** mettez des captures de côté avec `A`, puis assemblez-les en une seule image avec légendes (« Before », « After », ou « Step 1… » s'il y en a plus), côte à côte ou l'une sous l'autre. Ou envoyez la collection sous forme de fichiers séparés, qui se collent dans une discussion comme pièces jointes.

**À votre façon**
- Une fenêtre de réglages pour presque tout : touches, couleurs, outils par défaut, aspect de la sélection, nom et format des fichiers, et langue de l'interface (9 langues : anglais, russe, chinois, espagnol, portugais, japonais, allemand, français, coréen).
- Une icône dans la zone de notification avec démarrage automatique, un bouton pour définir le raccourci de lancement et une vérification des mises à jour facultative (désactivée par défaut).

## État

Projet personnel en développement actif ; la version actuelle est une préversion (`v0.1.0-rc4`). Des paquets prêts à l'emploi sont sur la page
[Releases](https://github.com/jexme/scrawl/releases) ; vous pouvez aussi le compiler depuis les sources (quelques minutes).
Linux : la cible principale est GNOME sous Wayland. Les sessions **X11** (GNOME sur Xorg, KDE,
XFCE et d'autres) fonctionnent aussi, mais ce support est récent et n'a été testé jusqu'ici que sur un
serveur X virtuel ; les retours de vraies machines sont les bienvenus. **Windows 10 et 11** fonctionnent aussi (capture, annotations,
presse-papiers, icône de la zone de notification et raccourci global) ; ce portage est récent et n'a été essayé qu'avec une échelle de 100 %, voir
[docs/WINDOWS.md](../WINDOWS.md) pour la compilation et ce qui manque.

## Prérequis

L'installation avec apt ou avec le `.deb` les installe pour vous ; la liste vaut pour l'archive de version et la compilation depuis les sources.

- GNOME sous Wayland avec PipeWire, `xdg-desktop-portal` (1.21 ou plus récent, avec l'enregistrement des applications) et `xdg-desktop-portal-gnome`. Développé et testé sous Ubuntu avec GNOME 50.
- `wl-clipboard` (la commande `wl-copy`) pour le presse-papiers et `libnotify` (la commande `notify-send`) pour les notifications. Sans `wl-copy`, Scrawl se rabat sur `xclip` via XWayland si les deux sont présents ; sans aucun des deux, un message indique ce qu'il faut installer.
- Sous **X11** il n'y a pas de portail : l'écran est capturé directement, et le presse-papiers demande `xclip` au lieu de `wl-clipboard`.

```bash
sudo apt install wl-clipboard libnotify-bin   # Wayland
sudo apt install xclip libnotify-bin          # X11
```

## Installation

### 1. Obtenir le programme

**Depuis le dépôt apt** (Debian, Ubuntu ; les mises à jour arrivent avec `apt upgrade`) :

```bash
sudo curl -fsSL https://jexme.github.io/scrawl/scrawl.sources -o /etc/apt/sources.list.d/scrawl.sources
sudo apt update && sudo apt install scrawl
```

Les paquets sont construits pour amd64 et arm64 (la version arm64 est faite au mieux ; sur les autres architectures, compilez depuis les sources, voir plus bas). Le dépôt est signé avec la
clé `E62C DAF5 A1C9 0DB7 A899 D24D FA43 15C7 154A 9EF0`, qui est incluse dans `scrawl.sources`.

Sur les systèmes plus anciens (Ubuntu 20.04, Debian 11 : un apt antérieur à 2.4 ne sait pas lire une clé dans un fichier `.sources`), utilisez plutôt un
trousseau séparé et une source sur une ligne :

```bash
curl -fsSL https://jexme.github.io/scrawl/scrawl-archive-keyring.gpg | sudo tee /usr/share/keyrings/scrawl-archive-keyring.gpg >/dev/null
echo "deb [signed-by=/usr/share/keyrings/scrawl-archive-keyring.gpg] https://jexme.github.io/scrawl stable main" | sudo tee /etc/apt/sources.list.d/scrawl.list
sudo apt update && sudo apt install scrawl
```

La clé est aussi publiée au format ASCII armor sous le nom `scrawl-archive-keyring.asc`.

**Depuis un paquet .deb** (Debian, Ubuntu) : téléchargez `scrawl_VERSION_amd64.deb`
depuis la page [Releases](https://github.com/jexme/scrawl/releases), puis
`sudo apt install ./scrawl_*.deb`. Il installe les bibliothèques et recommande `wl-clipboard`, `xclip`
et `libnotify-bin`. Le raccourci n'est pas attribué par le paquet (il est propre à chaque utilisateur) : faites-le dans `scrawl -settings`.
Pour le construire vous-même : `make deb`.

**Depuis une archive de version** : téléchargez
`scrawl-VERSION-linux-amd64.tar.gz` et `SHA256SUMS` depuis la page
[Releases](https://github.com/jexme/scrawl/releases), puis :

```bash
sha256sum -c --ignore-missing SHA256SUMS   # vérifie chaque fichier téléchargé qui figure dans SHA256SUMS
tar xzf scrawl-*-linux-amd64.tar.gz && cd scrawl-*-linux-amd64
./install.sh            # installe dans ~/.local (PREFIX=/usr/local pour tout le système)
sudo apt install libpipewire-0.3-0 libwayland-client0 libwayland-egl1 libwayland-cursor0 \
  libxkbcommon-x11-0 libegl1 libx11-6 libxcursor1 wl-clipboard libnotify-bin
```

`wl-clipboard` (le presse-papiers) et `libnotify-bin` (les notifications) sont aussi nécessaires à l'exécution : le
script d'installation les vérifie et indique ce qui manque. Sous X11, utilisez `xclip` à la place de `wl-clipboard`.

**Depuis les sources** :

```bash
sudo apt install gcc pkg-config patch libwayland-dev libx11-dev libx11-xcb-dev \
  libxkbcommon-x11-dev libgles2-mesa-dev libegl1-mesa-dev libffi-dev \
  libxcursor-dev libvulkan-dev libpipewire-0.3-dev
git clone https://github.com/jexme/scrawl.git && cd scrawl
make install            # compile et installe dans ~/.local (PREFIX=/usr/local pour tout le système)
```

Il faut Go (la version de `go.mod`). Pas de `sudo` ? `scripts/fetch-pipewire-headers.sh` extrait les en-têtes de PipeWire dans `.deps/` et le Makefile les utilise.

Vérifiez que `~/.local/bin` figure dans votre `PATH`.

### Windows 10 et 11

**Recommandé : l'installateur.** Téléchargez `scrawl-VERSION-windows-amd64-setup.exe` depuis la page
[Releases](https://github.com/jexme/scrawl/releases) et lancez-le. Il s'installe pour votre compte uniquement (sans droits d'administrateur),
ajoute une entrée au menu Démarrer, peut lancer Scrawl à l'ouverture de session et le lance une fois terminé. Pour mettre à jour, lancez le
nouvel installateur : il ferme d'abord l'ancien Scrawl. Si une version n'a pas de fichier d'installation, utilisez l'une des options ci-dessous.

Alternatives :

- **Zip portable.** Téléchargez `scrawl-VERSION-windows-amd64.zip` et son `.sha256`, décompressez-le dans un dossier permanent (et non
  directement depuis le visionneur d'archives) et lancez `scrawl.exe`. Un double-clic sur `scrawl.exe` démarre l'icône de la zone de notification.
  Un simple `scrawl-VERSION-windows-amd64.exe` est également publié. Pour mettre à jour, quittez Scrawl depuis le menu de son icône et remplacez le fichier.
- **[Scoop](https://scoop.sh) :** `scoop install https://github.com/jexme/scrawl/releases/download/v0.1.0-rc4/scrawl.json`.
  Cette adresse est liée à une préversion précise ; d'une préversion à l'autre, mettez donc à jour avec `scoop uninstall scrawl` puis `scoop install`
  avec l'URL de la nouvelle version (en attendant une version stable ou un bucket). Une fois le manifeste servi depuis
  `https://github.com/jexme/scrawl/releases/latest/download/scrawl.json` (quand une version stable existe), `scoop update scrawl` fonctionne.

- **L'icône** est dans la zone de notification ; sous Windows 11, elle peut se cacher derrière la flèche `^` près de l'horloge, faites-la glisser dehors pour la garder visible.
- **La touche Print** est le raccourci par défaut. Un autre programme l'occupe souvent (Greenshot, ShareX, Lightshot). Windows lui-même peut aussi l'occuper :
  Paramètres, Accessibilité, Clavier, option *Use the Print screen button to open screen capture* (nommée en conséquence dans l'interface française) : désactivez-la. Ou choisissez une autre touche dans
  `scrawl.exe -settings`, par exemple `<Control><Alt>a`. L'icône vous prévient quand la touche est prise et continue d'essayer.
- **« Windows a protégé votre ordinateur »** (« Windows protected your PC », SmartScreen) s'affiche parce que le programme n'est pas encore signé : cliquez sur *Informations complémentaires*, puis
  sur *Exécuter quand même*. Pour vérifier d'abord le fichier, comparez `Get-FileHash` avec le fichier `.sha256`, ou lancez
  `gh attestation verify scrawl-*-setup.exe --repo jexme/scrawl` (le build provient d'une CI publique). Les antivirus peuvent signaler
  tout nouveau programme Go non signé ; signalez un tel cas dans une issue.
- **Le démarrage automatique** est la case à cocher dans `scrawl.exe -settings`, onglet *Système* (une valeur dans `HKCU\...\Run`), ou une case de l'installateur.
  Si vous déplacez ou mettez à jour le programme, le prochain lancement de l'icône réécrit l'entrée vers le nouvel emplacement (un démarrage automatique désactivé reste désactivé).
- **Lancer une version plus récente alors qu'une ancienne tourne :** Scrawl vous indique quelle version est en cours d'exécution ; quittez-la d'abord depuis le
  menu de l'icône, puis lancez la nouvelle.

Compilation depuis les sources et ce qui manque : [docs/WINDOWS.md](../WINDOWS.md).

### 2. Premier lancement : trois étapes, une seule fois

*Sous X11, les étapes 1 et 2 ne sont pas nécessaires (pas de portail), seul le raccourci l'est.*

Ouvrez la fenêtre de réglages (`scrawl -settings`, onglet « Système »). Le bouton « Régler la capture » fait les étapes 1 et 2 d'un seul coup (au tout premier lancement, la fenêtre s'ouvre toute seule). Ou utilisez les boutons séparés, ou lancez les commandes :

1. **Choisir les écrans :** bouton « Choisir de nouveau les écrans » ou `scrawl -setup`. GNOME affiche une boîte de dialogue **par écran**. Dans chacune, choisissez un écran *différent*, laissez cochée la case **« Se souvenir de ce choix »** et acceptez le partage. Ensuite, plus de boîtes de dialogue.
2. **Accorder l'autorisation de secours :** bouton « Accorder l'autorisation » ou `scrawl -grant-screenshot`. Elle permet à Scrawl d'utiliser le portail de capture, plus lent, dans les rares cas où PipeWire ne fournit aucune image.
3. **Définir un raccourci :** sous GNOME, l'installateur de l'archive (`install.sh`) et `make install` attribuent `Print` eux-mêmes, sauf si un autre raccourci personnalisé l'utilise déjà. Après une installation avec apt ou le `.deb`, ou pour changer la touche : le champ « Raccourci de lancement » et le bouton « Attribuer » (par défaut `Print`), ou `scripts/gnome-hotkey.sh [TOUCHE]`. Il crée un raccourci personnalisé GNOME nommé « scrawl » (ou le met à jour s'il existe, si bien que le relancer n'en crée jamais un second) et n'écrase jamais le raccourci de quelqu'un d'autre qui est déjà pris. Si `Print` ouvre toujours l'outil de capture de GNOME, désactivez-le dans *Paramètres → Clavier → Afficher et personnaliser les raccourcis → Captures d'écran*.

### Passer depuis screen-editor

Scrawl est le nouveau nom de screen-editor. Vos réglages sont lus depuis `~/.config/screen-editor/config.json` tant que `~/.config/scrawl/config.json` n'existe pas (l'ancien fichier reste intact). Le portail lie le choix des écrans et les autorisations à l'identifiant de l'application, qui a changé : refaites donc une fois les étapes 1 à 3 et faites pointer votre raccourci vers `scrawl`. La dernière zone et la collection repartent de zéro.

## Utilisation

1. Appuyez sur votre raccourci. L'écran se fige et s'assombrit.
2. Faites glisser pour sélectionner une zone (sur n'importe quel écran). Une barre d'outils apparaît dessous.
3. Annotez si vous le souhaitez : choisissez un outil et une couleur, puis dessinez sur la zone.
4. Appuyez sur `Enter`. L'image est dans le presse-papiers et dans `~/Pictures/Screenshots` ; collez-la où vous voulez.

`Esc` annule à tout moment.

### Touches

| Touche | Action |
| --- | --- |
| `Enter` | copier dans le presse-papiers **et** enregistrer dans un fichier (ce qu'elle fait est configurable) |
| `Ctrl+C` / `Ctrl+S` | copier seulement / enregistrer seulement |
| `Ctrl+Z` | annuler la dernière annotation |
| `Esc` | annuler (ferme d'abord un menu ouvert ; pendant la saisie d'un texte, il abandonne seulement le texte) |
| `Ctrl+A` | sélectionner tout l'écran du moniteur (non reconfigurable) |
| `R` | rétablir la zone de la capture précédente |
| `V` `P` `L` `W` `B` `M` `U` `T` `N` | outils : sélection, crayon, ligne, flèche, cadre, surligneur, flou, texte, numéro (quand une zone est sélectionnée) |
| `+` / `-` | épaisseur, taille du texte ou intensité du flou de l'outil actuel (comme la molette de la souris) |
| `A` | ajouter la capture à la collection |
| `G` | assembler la collection (et la zone actuelle) en une seule image |
| `D` | enregistrer la collection en fichiers séparés et mettre la liste des fichiers dans le presse-papiers |
| molette de la souris | épaisseur, taille du texte ou intensité du flou de l'outil actuel |
| `Shift` pendant le dessin | crayon et surligneur : une ligne droite ; ligne et flèche : alignement sur 45° ; cadre et flou : un carré. `Shift` pendant la sélection d'une zone : un carré |

Chaque touche peut être modifiée dans la fenêtre de réglages (onglet « Touches »).

### Barre d'outils

La rangée principale, de gauche à droite : sélection, crayon, ligne, flèche, cadre, surligneur, flou, texte, numéro ·
annuler · copier · enregistrer · le menu `⋯` (réglages, qui ferme la surcouche et ouvre la fenêtre
de réglages, et à la collection) · fermer.

Quand un outil de dessin est actif, une seconde rangée apparaît dessous : huit couleurs, un bouton arc-en-ciel pour
votre propre couleur (ouvre une palette ; la couleur est mémorisée) et `−` épaisseur `+`. L'outil « sélection »
n'a ni l'un ni l'autre, la rangée est donc masquée.

Les boutons qui ont une touche l'affichent sous l'icône (les touches viennent des réglages). Laissez le curseur un instant sur un bouton pour voir son nom et sa touche. Les raccourcis fonctionnent avec n'importe quelle disposition de clavier (une disposition russe, par exemple), d'après la position de la touche.

### Collections avant/après

Faites la première capture et appuyez sur `A`. Faites la suivante : la zone précédente est déjà sélectionnée, vous photographiez donc de nouveau le même endroit. Appuyez sur `G` pour assembler les captures en une image (elle est enregistrée et copiée) ou sur `D` pour obtenir des fichiers séparés. Le petit panneau en bas de l'écran montre le contenu de la collection et propose les mêmes actions sous forme de boutons.

## Icône de la zone de notification et démarrage automatique

`scrawl -tray` garde une icône dans la zone de notification : le clic gauche fait une capture et le menu propose « Faire une capture », « Réglages » et « Quitter ». Pour la lancer à l'ouverture de session, activez *Réglages → Système → Démarrage automatique*. Sous GNOME, l'icône nécessite l'extension AppIndicator (activée par défaut dans Ubuntu).

## Réglages

![La fenêtre de réglages](../images/settings-en.png)

Ouvrez-les depuis la barre d'outils (engrenage), avec `scrawl -settings` ou depuis le menu des applications. Les modifications sont enregistrées aussitôt dans `~/.config/scrawl/config.json`, un simple fichier JSON que vous pouvez aussi modifier à la main. Les valeurs absentes ou invalides reviennent aux valeurs par défaut. Chaque clé est décrite dans la [référence des réglages](../SETTINGS.md) (en anglais ou en [russe](SETTINGS.ru.md)).

## Fichiers et vie privée

- Les images vont dans `~/Pictures/Screenshots` (modifiable dans les réglages), nommées par exemple `scrawl-20260131-142501.png`.
- Réglages : `~/.config/scrawl/`. Jetons des écrans, dernière zone et collection : `~/.cache/scrawl/`.
- Scrawl ne se connecte jamais à Internet de lui-même. La seule requête réseau du programme est la vérification des mises à jour, facultative et désactivée par défaut : elle demande à GitHub la dernière version et n'envoie, en dehors des en-têtes HTTP habituels, que la version du programme.
- Les captures ne quittent pas votre ordinateur, sauf si vous les collez quelque part.

## Ligne de commande

| Option | Effet |
| --- | --- |
| `-settings` | ouvre la fenêtre de réglages |
| `-setup` | rechoisit les écrans (boîtes de dialogue GNOME) |
| `-grant-screenshot` | accorde l'autorisation de capture de secours et quitte |
| `-tray` | garde l'icône dans la zone de notification |
| `-fresh` | ne propose pas la zone de la capture précédente |
| `-v` | affiche les temps mesurés |
| `-version` | affiche la version et quitte |
| `-exit-after DURATION` | quitte normalement après la durée donnée, pour les tests (par exemple `-exit-after 3s`) |

## Dépannage

- **Rien ne se passe avec le raccourci.** `xdg-desktop-portal` est peut-être bloqué. Redémarrez-le : `systemctl --user restart xdg-desktop-portal xdg-desktop-portal-gnome`.
- **GNOME demande quel écran partager à chaque capture.** La case « Se souvenir de ce choix » n'était pas cochée ; refaites l'étape 1.
- **Un écran a été ajouté ou remplacé.** Choisissez de nouveau les écrans (étape 1).
- **La surcouche ne s'ouvre que sur un écran.** Même chose : l'étape 1 doit être faite pour chaque écran.
- **Le message indique que le portail a refusé.** Accordez l'autorisation (étape 2).
- **« Clipboard error » ou le collage ne donne rien.** Vérifiez que `wl-copy` est installé (`wl-clipboard`).
- **Pas de notifications.** Installez `libnotify-bin` (la commande `notify-send`) et vérifiez que les notifications sont activées dans les réglages.
- **Pas d'icône dans la zone de notification.** Activez l'extension GNOME AppIndicator.
- **`Print` ouvre l'outil de capture de GNOME.** Désactivez-le dans les raccourcis clavier (voir l'étape 3).

## Désinstallation

**apt ou `.deb` :**

```bash
sudo apt remove scrawl     # ou : sudo apt purge scrawl (le paquet ne laisse aucun fichier hors de /usr, c'est donc identique)
sudo rm /etc/apt/sources.list.d/scrawl.sources   # facultatif : oublier le dépôt (scrawl.list pour l'ancienne forme)
```

apt ne peut pas toucher à votre dossier personnel, donc quelques restes propres à l'utilisateur subsistent : le raccourci GNOME, l'entrée de démarrage automatique si vous
l'aviez activée, et vos réglages. Supprimez le raccourci avec `gnome-hotkey.sh` de Scrawl (il se trouve dans `scripts/` du
dépôt et dans l'archive de version) ou à la main dans *Paramètres → Clavier → Afficher et personnaliser les raccourcis → Raccourcis
personnalisés* ; le reste, ce sont de simples fichiers :

```bash
sh gnome-hotkey.sh --remove
rm -f ~/.config/autostart/io.github.jexme.Scrawl.desktop
rm -rf ~/.config/scrawl ~/.cache/scrawl     # facultatif : réglages et cache
```

**Archive de version ou `make install` :** lancez `./uninstall.sh` depuis l'archive décompressée (`make uninstall` dans une copie des
sources). Il supprime le binaire, le fichier `.desktop`, le raccourci GNOME et l'entrée de démarrage automatique. Ajoutez `--purge`
(`make uninstall PURGE=1`) pour supprimer aussi les réglages (`~/.config/scrawl`) et le cache (`~/.cache/scrawl`). Indiquez
le même `PREFIX` que pour l'installation si vous l'aviez changé. Vos captures dans `~/Pictures/Screenshots` ne sont jamais touchées.

## Fonctionnement

Scrawl capture l'écran via l'interface ScreenCast de `xdg-desktop-portal` avec son propre petit client PipeWire (`internal/pwcapture`). Un jeton de restauration enregistré pour chaque écran supprime la boîte de dialogue. Le compositeur (Mutter) ne livre des images que lorsque l'écran se redessine, et lors de nos tests il n'en a livré aucune au-dessus d'une fenêtre plein écran (affichage direct) ; si aucune image n'arrive en 250 ms, Scrawl se rabat sur le portail de capture, plus lent (environ 0,5-1 s). Les surcouches sont des fenêtres [Gio](https://gioui.org) plein écran, une par écran ; Gio est corrigé par un patch (`third_party/gio.patch`) pour placer une fenêtre sur une sortie Wayland donnée. Le résultat est dessiné par le même code que celui qui l'affiche à l'écran, si bien que l'image enregistrée correspond à ce que vous avez vu.

## Compiler et contribuer

```bash
make build    # bin/scrawl ; télécharge et patche Gio au premier lancement
make test     # tests unitaires, y compris l'icône de notification sur le D-Bus de session
make dist     # archive de version et somme de contrôle dans dist/
make deb      # paquet .deb pour Debian et Ubuntu dans dist/
make notices  # régénère THIRD_PARTY_NOTICES.md
```

Les textes de l'interface sont dans `internal/i18n/` (un fichier par langue, `lang_ru.go`, `lang_de.go` et ainsi de suite ; les textes d'origine dans le code sont en anglais) ;
`go test ./...` vérifie que chaque texte est traduit dans chaque langue. Comment signaler un bogue et envoyer une modification : [CONTRIBUTING.md](../../CONTRIBUTING.md). Plan et idées : [docs/PLAN.md](../PLAN.md) (en russe).

## Licence

MIT, voir [LICENSE](../../LICENSE). Composants tiers : [THIRD_PARTY_NOTICES.md](../../THIRD_PARTY_NOTICES.md) (les formes des icônes viennent de [Tabler Icons](https://tabler.io/icons), MIT).
