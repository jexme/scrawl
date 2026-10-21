# Scrawl

**Captures d'écran rapides avec annotations pour GNOME sous Wayland.** Appuyez sur une touche, sélectionnez une zone sur n'importe quel écran, dessinez des flèches, des cadres et du texte, floutez les données privées, et le résultat est déjà dans le presse-papiers et enregistré dans un fichier.

[English](../../README.md) · [Русский](README.ru.md) · [简体中文](README.zh-CN.md) · [Español](README.es.md) · [Português (Brasil)](README.pt-BR.md) · [日本語](README.ja.md) · [Deutsch](README.de.md) · **Français** · [한국어](README.ko.md)

![Surcouche Scrawl : une zone sélectionnée avec un cadre, une flèche, un texte, un trait de surligneur et une partie floutée, la barre d'outils en dessous](../images/overlay.png)

*L'image est un vrai rendu de la surcouche sur un bureau imaginaire.*

> L'interface est disponible en anglais, russe, chinois simplifié, espagnol, portugais, japonais, allemand, français et coréen ; les noms de boutons ci-dessous sont ceux de l'interface en anglais, avec leurs équivalents dans chaque langue.

## Pourquoi Scrawl

Sous GNOME avec Wayland, les applications ne peuvent pas capturer l'écran directement : elles le demandent au portail du bureau, et le portail **demande « Quel écran voulez-vous partager ? » à chaque capture**. Scrawl ne le demande qu'une fois par écran et retient la réponse ; ensuite, une capture se fait d'une seule pression : l'écran figé apparaît en environ 150 ms.

## Fonctionnalités

**Capture**
- Fonctionne sur tous les écrans à la fois : la surcouche s'ouvre sur chacun et vous sélectionnez sur n'importe lequel.
- Sélection avec un cadre en pointillés. Faites glisser les coins pour redimensionner ; avec l'outil *sélection*, faites glisser un bord pour redimensionner ou l'intérieur pour déplacer la zone.
- Retient la dernière zone et la ramène d'une touche (`R`).
- En option, le curseur de la souris apparaît sur la capture.

**Annotations**
- Crayon, ligne, flèche, cadre, surligneur, **flou** et texte ; huit couleurs.
- La molette de la souris (ou les boutons `−` / `+`) règle l'épaisseur, la taille du texte ou l'intensité du flou.
- Annulation avec `Ctrl+Z`.

**Enregistrer et partager**
- `Enter` copie l'image dans le presse-papiers et enregistre un PNG dans `~/Pictures/Screenshots`. `Ctrl+C` copie seulement, `Ctrl+S` enregistre seulement. Le JPEG est aussi possible.
- **Collections avant/après :** mettez des captures de côté avec `A`, puis assemblez-les en une seule image avec légendes (« Before », « After », ou « Step 1… » s'il y en a plus), côte à côte ou l'une sous l'autre. Ou envoyez la collection sous forme de fichiers séparés, qui se collent dans une discussion comme pièces jointes.

**À votre façon**
- Une fenêtre de réglages pour presque tout : touches, couleurs, outils par défaut, aspect de la sélection, nom et format des fichiers, et langue de l'interface (9 langues).
- Une icône dans la zone de notification avec démarrage automatique, un bouton pour définir le raccourci de lancement et une vérification des mises à jour facultative (désactivée par défaut).

## État

Projet personnel en développement actif. **Aucune version n'a encore été publiée** : pour l'instant il faut le compiler depuis les sources (quelques minutes). La cible principale est GNOME sous Wayland (Linux) ; les sessions **X11** (GNOME sur Xorg, KDE, XFCE et d'autres) fonctionnent aussi, mais ce support est récent et n'a été testé jusqu'ici que sur un serveur X virtuel ; les retours de vraies machines sont les bienvenus. Une version Windows compile, mais la capture d'écran n'est pas encore écrite ; voir [docs/WINDOWS.md](../WINDOWS.md).

## Prérequis

- GNOME sous Wayland avec PipeWire, `xdg-desktop-portal` (1.21 ou plus récent, avec l'enregistrement des applications) et `xdg-desktop-portal-gnome`. Développé et testé sous Ubuntu avec GNOME 50.
- `wl-clipboard` (la commande `wl-copy`) pour le presse-papiers et `libnotify` (la commande `notify-send`) pour les notifications .
- Sous **X11** il n'y a pas de portail : l'écran est capturé directement, et le presse-papiers demande `xclip` au lieu de `wl-clipboard`.

```bash
sudo apt install wl-clipboard libnotify-bin   # Wayland
sudo apt install xclip libnotify-bin          # X11
```

## Installation

### 1. Obtenir le programme

**Depuis le dépôt apt** (Debian, Ubuntu ; les mises à jour arrivent avec `apt upgrade`) :

```bash
curl -fsSL https://jexme.github.io/scrawl/scrawl-archive-keyring.gpg | sudo tee /usr/share/keyrings/scrawl-archive-keyring.gpg >/dev/null
echo "deb [signed-by=/usr/share/keyrings/scrawl-archive-keyring.gpg] https://jexme.github.io/scrawl stable main" | sudo tee /etc/apt/sources.list.d/scrawl.list
sudo apt update && sudo apt install scrawl
```

Le dépôt est signé avec la clé `E62C DAF5 A1C9 0DB7 A899 D24D FA43 15C7 154A 9EF0`.

**Depuis les sources (fonctionne dès aujourd'hui) :**

```bash
sudo apt install gcc pkg-config patch libwayland-dev libx11-dev libx11-xcb-dev \
  libxkbcommon-x11-dev libgles2-mesa-dev libegl1-mesa-dev libffi-dev \
  libxcursor-dev libvulkan-dev libpipewire-0.3-dev
git clone https://github.com/jexme/scrawl.git && cd scrawl
make install            # compile et installe dans ~/.local (PREFIX=/usr/local pour tout le système)
```

Il faut Go (la version de `go.mod`). Pas de `sudo` ? `scripts/fetch-pipewire-headers.sh` extrait les en-têtes de PipeWire dans `.deps/` et le Makefile les utilise.

**Depuis une archive de version (une fois qu'une version sera publiée) :** téléchargez `scrawl-VERSION-linux-amd64.tar.gz` et `SHA256SUMS` depuis la page [Releases](https://github.com/jexme/scrawl/releases), puis :

```bash
sha256sum -c --ignore-missing SHA256SUMS
tar xzf scrawl-*-linux-amd64.tar.gz && cd scrawl-*-linux-amd64
./install.sh            # installe dans ~/.local (PREFIX=/usr/local pour tout le système)
sudo apt install libpipewire-0.3-0 libwayland-client0 libwayland-egl1 libwayland-cursor0 \
  libxkbcommon-x11-0 libegl1 libx11-6 libxcursor1
```

Vérifiez que `~/.local/bin` figure dans votre `PATH`.

### 2. Premier lancement : trois étapes, une seule fois

*Sous X11, les étapes 1 et 2 ne sont pas nécessaires (pas de portail), seul le raccourci l'est.*

Ouvrez la fenêtre de réglages (`scrawl -settings`, tab “System”) et utilisez ces boutons, ou lancez les commandes :

1. **Choisir les écrans :** bouton « Pick monitors again » ou `scrawl -setup`. GNOME affiche une boîte de dialogue **par écran**. Dans chacune, choisissez un écran *différent*, laissez cochée la case **« Se souvenir de ce choix »** et acceptez le partage. Ensuite, plus de boîtes de dialogue.
2. **Accorder l'autorisation de secours :** bouton « Grant permission » ou `scrawl -grant-screenshot`. Elle permet à Scrawl d'utiliser le portail de capture, plus lent, dans les rares cas où PipeWire ne fournit aucune image.
3. **Définir un raccourci :** le champ « Launch hotkey » et le bouton « Assign » (par défaut `Print`), ou `scripts/gnome-hotkey.sh [TOUCHE]`. Il crée un raccourci personnalisé GNOME et n'écrase jamais une combinaison déjà prise. Si `Print` ouvre toujours l'outil de capture de GNOME, désactivez-le dans *Paramètres → Clavier → Afficher et personnaliser les raccourcis → Captures d'écran*.

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
| `Esc` | annuler |
| `R` | rétablir la zone de la capture précédente |
| `A` | ajouter la capture à la collection |
| `G` | assembler la collection (et la zone actuelle) en une seule image |
| `D` | enregistrer la collection en fichiers séparés et mettre la liste des fichiers dans le presse-papiers |
| molette de la souris | épaisseur, taille du texte ou intensité du flou de l'outil actuel |

Chaque touche peut être modifiée dans la fenêtre de réglages (« Keys »).

### Barre d'outils

De gauche à droite : sélection, crayon, ligne, flèche, cadre, surligneur, flou, texte · huit couleurs · `−` épaisseur `+` · réglages (ferme la surcouche et ouvre la fenêtre de réglages) · ajouter à la collection · annuler · copier · enregistrer · fermer.

### Collections avant/après

Faites la première capture et appuyez sur `A`. Faites la suivante : la zone précédente est déjà sélectionnée, vous photographiez donc de nouveau le même endroit. Appuyez sur `G` pour assembler les captures en une image (elle est enregistrée et copiée) ou sur `D` pour obtenir des fichiers séparés. Le petit panneau en bas de l'écran montre le contenu de la collection et propose les mêmes actions sous forme de boutons.

## Icône de la zone de notification et démarrage automatique

`scrawl -tray` garde une icône dans la zone de notification : le clic gauche fait une capture et le menu propose « Screenshot », « Settings » et « Quit ». Pour la lancer à l'ouverture de session, activez *Settings → System → Autostart*. Sous GNOME, l'icône nécessite l'extension AppIndicator (activée par défaut dans Ubuntu).

## Réglages

![La fenêtre de réglages](../images/settings-en.png)

Ouvrez-les depuis la barre d'outils (engrenage), avec `scrawl -settings` ou depuis le menu des applications. Les modifications sont enregistrées aussitôt dans `~/.config/scrawl/config.json`, un simple fichier JSON que vous pouvez aussi modifier à la main. Les valeurs absentes ou invalides reviennent aux valeurs par défaut. Chaque clé est décrite dans la [référence des réglages](../SETTINGS.md) (en anglais ou en [russe](../SETTINGS.md)).

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

## Dépannage

- **Rien ne se passe avec le raccourci.** `xdg-desktop-portal` est peut-être bloqué. Redémarrez-le : `systemctl --user restart xdg-desktop-portal xdg-desktop-portal-gnome`.
- **GNOME demande quel écran partager à chaque capture.** La case « Se souvenir de ce choix » n'était pas cochée ; refaites l'étape 1.
- **Un écran a été ajouté ou remplacé.** Choisissez de nouveau les écrans (étape 1).
- **La surcouche ne s'ouvre que sur un écran.** Même chose : l'étape 1 doit être faite pour chaque écran.
- **Le message indique que le portail a refusé.** Accordez l'autorisation (étape 2).
- **Le collage dans certaines applications ne donne rien.** Vérifiez que `wl-copy` est installé (paquet `wl-clipboard`).
- **Pas de notifications.** Installez `libnotify-bin` (la commande `notify-send`) et vérifiez que les notifications sont activées dans les réglages.
- **Pas d'icône dans la zone de notification.** Activez l'extension GNOME AppIndicator.
- **`Print` ouvre l'outil de capture de GNOME.** Désactivez-le dans les raccourcis clavier (voir l'étape 3).

## Fonctionnement

Scrawl capture l'écran via l'interface ScreenCast de `xdg-desktop-portal` avec son propre petit client PipeWire (`internal/pwcapture`). Un jeton de restauration enregistré pour chaque écran supprime la boîte de dialogue. Le compositeur (Mutter) ne livre des images que lorsque l'écran se redessine, et lors de nos tests il n'en a livré aucune au-dessus d'une fenêtre plein écran (affichage direct) ; si aucune image n'arrive en 250 ms, Scrawl se rabat sur le portail de capture, plus lent (environ 0,5-1 s). Les surcouches sont des fenêtres [Gio](https://gioui.org) plein écran, une par écran ; Gio est corrigé par un patch (`third_party/gio.patch`) pour placer une fenêtre sur une sortie Wayland donnée. Le résultat est dessiné par le même code que celui qui l'affiche à l'écran, si bien que l'image enregistrée correspond à ce que vous avez vu.

## Compiler et contribuer

```bash
make build    # bin/scrawl ; télécharge et patche Gio au premier lancement
make test     # tests unitaires, y compris l'icône de notification sur le D-Bus de session
make dist     # archive de version et somme de contrôle dans dist/
make notices  # régénère THIRD_PARTY_NOTICES.md
```

Les textes de l'interface sont dans `internal/i18n/` (un fichier par langue ; les textes d'origine dans le code sont en russe) ; `go test ./...` vérifie que chaque texte est traduit dans chaque langue. Plan et idées : [docs/PLAN.md](../PLAN.md) (en russe).

## Licence

MIT, voir [LICENSE](../../LICENSE). Composants tiers : [THIRD_PARTY_NOTICES.md](../../THIRD_PARTY_NOTICES.md) (les formes des icônes viennent de [Tabler Icons](https://tabler.io/icons), MIT).
