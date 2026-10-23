# Scrawl

**Capturas de tela rápidas com anotações para o GNOME no Wayland.** Aperte uma tecla, selecione uma área em qualquer monitor, desenhe setas, caixas e texto, desfoque dados privados, e o resultado já está na área de transferência e salvo em arquivo.

[English](../../README.md) · [Русский](README.ru.md) · [简体中文](README.zh-CN.md) · [Español](README.es.md) · **Português (Brasil)** · [日本語](README.ja.md) · [Deutsch](README.de.md) · [Français](README.fr.md) · [한국어](README.ko.md)

![Sobreposição do Scrawl: uma área selecionada com uma caixa, uma seta, um texto, um traço de marca-texto e um trecho desfocado, com a barra de ferramentas abaixo](../images/overlay.png)

*A imagem é uma renderização real da sobreposição sobre uma área de trabalho inventada.*

> A interface está disponível em inglês, russo, chinês simplificado, espanhol, português, japonês, alemão, francês e coreano; os nomes dos botões abaixo são os da interface em português.

## Por que o Scrawl

No GNOME com Wayland, os aplicativos não conseguem capturar a tela diretamente: eles pedem ao portal da área de trabalho, e o portal **pergunta “qual tela você quer compartilhar?” a cada captura**. O Scrawl pergunta uma vez por monitor e lembra da resposta; depois disso, uma captura é um único toque de tecla: a tela congelada aparece em cerca de 150 ms.

## Início rápido

No Ubuntu ou no Debian (Wayland ou X11) são três passos.

**1. Instale** a partir do repositório apt assinado (as atualizações chegam depois com `apt upgrade`):

```bash
sudo curl -fsSL https://jexme.github.io/scrawl/scrawl.sources -o /etc/apt/sources.list.d/scrawl.sources
sudo apt update && sudo apt install scrawl
```

O arquivo `.sources` leva a chave de assinatura dentro dele, então não há um chaveiro separado para gerenciar. Os pacotes são
compilados apenas para **amd64**; no arm64, [compile a partir do código-fonte](#instalação).

**2. Configure uma única vez:** inicie o Scrawl pelo menu de aplicativos (ou execute `scrawl`): a primeira execução abre um guia curto
na aba “Sistema” (você pode abri-lo quando quiser com `scrawl -settings`). No Wayland, pressione “Configurar a captura”
(o GNOME pergunta uma vez por monitor: escolha um monitor diferente em cada caixa de diálogo e marque “Lembrar esta escolha”). Depois digite uma tecla de atalho
(`Print` por padrão) e pressione “Atribuir”. A ferramenta de captura do próprio GNOME também usa `Print`: pressione “Liberar a tecla” para
desativar esse atalho embutido (só essa tecla, e você pode desfazer). No X11 só a tecla de atalho é necessária.

**3. Tire uma captura:** aperte a tecla de atalho, arraste uma área e pressione `Enter`. A imagem está na área de transferência e em
`~/Pictures/Screenshots`.

Outros sistemas: [Windows](#windows-10-e-11), outras distribuições Linux (um arquivo de versão ou uma compilação a partir do código-fonte), veja
[Instalação](#instalação) abaixo.

## Recursos

**Captura**
- Funciona em todos os monitores ao mesmo tempo: a sobreposição abre em cada um e você pode selecionar em qualquer um.
- Seleção com moldura tracejada. Arraste os cantos para redimensionar; com a ferramenta *selecionar*, arraste uma borda para redimensionar ou o interior para mover a área.
- Enquanto você arrasta uma moldura, uma lupa ao lado do cursor mostra os pixels 6 vezes maiores, com a borda da moldura
  sobre a grade de pixels, para que você possa alinhá-la com exatidão (pode ser desativada nas configurações).
- Lembra da última área e a traz de volta com uma tecla (`R`).
- Opcionalmente, inclui o cursor do mouse na captura (por enquanto só no Wayland).

**Anotações**
- Caneta, linha, seta, caixa, marca-texto, **desfoque**, texto e **carimbos numerados** (um clique coloca um círculo com 1, 2, 3 e assim por diante, como no flameshot); oito cores.
- A roda do mouse (ou os botões `−` / `+`) muda a espessura, o tamanho do texto ou a intensidade do desfoque.
- O **desfoque** tem seis níveis (roda ou `−` / `+`): 1-3 desfoque gaussiano suave, 4-6 mosaico com quadrados cada vez maiores. As células do mosaico começam no canto da área, e a nota que aparece ao mudar de nível indica o modo.
- Segure **Shift** enquanto desenha: a caneta e o marca-texto fazem uma linha reta, uma linha ou uma seta se ajusta a 45°, uma caixa ou um desfoque vira um quadrado.
- Desfazer com `Ctrl+Z`.

**Salvar e compartilhar**
- `Enter` copia a imagem para a área de transferência e salva um PNG em `~/Pictures/Screenshots`. `Ctrl+C` só copia, `Ctrl+S` só salva. JPEG também está disponível.
- Um **clique duplo** dentro da área selecionada a copia na hora (nas configurações, aba “Geral”, seção “Ações”, ele também pode copiar e salvar, ou não fazer nada).
- **Coleções antes/depois:** separe capturas com `A` e una-as em uma única imagem com legendas (“Before”, “After” ou “Step 1…” para mais), lado a lado ou uma sobre a outra. Ou envie a coleção como arquivos separados, que são colados em um chat como anexos.

**Do seu jeito**
- Uma janela de configurações para quase tudo: teclas, cores, ferramentas padrão, aparência da seleção, nome e formato dos arquivos e idioma da interface (9 idiomas: inglês, russo, chinês, espanhol, português, japonês, alemão, francês, coreano).
- Ícone na bandeja com início automático, um botão para definir a tecla de atalho e uma verificação de atualizações opcional (desativada por padrão).

## Status

Projeto pessoal em desenvolvimento ativo; a versão atual é uma pré-versão (`v0.1.0-rc3`). Os pacotes prontos estão na página de
[Releases](https://github.com/jexme/scrawl/releases); você também pode compilar a partir do código-fonte (leva alguns minutos).
Linux: o alvo principal é o GNOME no Wayland. As sessões **X11** (GNOME no Xorg, KDE,
XFCE e outras) também funcionam, mas esse suporte é novo e até agora só foi testado em um
servidor X virtual; relatos de máquinas reais são bem-vindos. **Windows 10 e 11** também funcionam (captura, anotações,
área de transferência, ícone na bandeja e uma tecla de atalho global); este port é novo e só foi testado com escala de 100%, veja
[docs/WINDOWS.md](../WINDOWS.md) para compilar e saber o que falta.

## Requisitos

A instalação com apt ou com o `.deb` traz tudo isso para você; a lista vale para o arquivo de versão e para a compilação a partir do código-fonte.

- GNOME no Wayland com PipeWire, `xdg-desktop-portal` (1.21 ou mais recente, com suporte ao registro de aplicativos) e `xdg-desktop-portal-gnome`. Desenvolvido e testado no Ubuntu com GNOME 50.
- `wl-clipboard` (o comando `wl-copy`) para a área de transferência e `libnotify` (o comando `notify-send`) para as notificações. Sem o `wl-copy`, o Scrawl recorre ao
  `xclip` por meio do XWayland, se ambos estiverem presentes; sem nenhum dos dois, você recebe uma dica do que instalar.
- No **X11** não há portal: a tela é capturada diretamente e a área de transferência precisa do `xclip` em vez do `wl-clipboard`.

```bash
sudo apt install wl-clipboard libnotify-bin   # Wayland
sudo apt install xclip libnotify-bin          # X11
```

## Instalação

### 1. Obtenha o programa

**Do repositório apt** (Debian, Ubuntu; as atualizações chegam com `apt upgrade`):

```bash
sudo curl -fsSL https://jexme.github.io/scrawl/scrawl.sources -o /etc/apt/sources.list.d/scrawl.sources
sudo apt update && sudo apt install scrawl
```

Os pacotes são compilados apenas para amd64 (no arm64: compile a partir do código-fonte, veja abaixo). O repositório é assinado com a
chave `E62C DAF5 A1C9 0DB7 A899 D24D FA43 15C7 154A 9EF0`, que vai embutida no `scrawl.sources`.

Em sistemas mais antigos (Ubuntu 20.04, Debian 11: um apt anterior à 2.4 não consegue ler uma chave dentro de um arquivo `.sources`) use no lugar um
chaveiro separado e uma fonte de uma linha:

```bash
curl -fsSL https://jexme.github.io/scrawl/scrawl-archive-keyring.gpg | sudo tee /usr/share/keyrings/scrawl-archive-keyring.gpg >/dev/null
echo "deb [signed-by=/usr/share/keyrings/scrawl-archive-keyring.gpg] https://jexme.github.io/scrawl stable main" | sudo tee /etc/apt/sources.list.d/scrawl.list
sudo apt update && sudo apt install scrawl
```

A chave também é publicada em formato ASCII-armored como `scrawl-archive-keyring.asc`.

**De um pacote .deb** (Debian, Ubuntu): baixe `scrawl_VERSION_amd64.deb`
na página de [Releases](https://github.com/jexme/scrawl/releases) e depois
`sudo apt install ./scrawl_*.deb`. Ele instala as bibliotecas necessárias e recomenda `wl-clipboard`, `xclip`
e `libnotify-bin`. O pacote não atribui a tecla de atalho (ela é por usuário): faça isso em `scrawl -settings`.
Para compilá-lo você mesmo: `make deb`.

**De um arquivo de versão**: baixe
`scrawl-VERSION-linux-amd64.tar.gz` e `SHA256SUMS` na página de
[Releases](https://github.com/jexme/scrawl/releases) e execute:

```bash
sha256sum -c --ignore-missing SHA256SUMS   # verifica cada arquivo baixado listado em SHA256SUMS
tar xzf scrawl-*-linux-amd64.tar.gz && cd scrawl-*-linux-amd64
./install.sh            # instala em ~/.local (PREFIX=/usr/local para todo o sistema)
sudo apt install libpipewire-0.3-0 libwayland-client0 libwayland-egl1 libwayland-cursor0 \
  libxkbcommon-x11-0 libegl1 libx11-6 libxcursor1 wl-clipboard libnotify-bin
```

`wl-clipboard` (a área de transferência) e `libnotify-bin` (as notificações) também são necessários em tempo de execução: o
instalador os verifica e mostra o que está faltando. No X11 use `xclip` em vez de `wl-clipboard`.

**A partir do código-fonte**:

```bash
sudo apt install gcc pkg-config patch libwayland-dev libx11-dev libx11-xcb-dev \
  libxkbcommon-x11-dev libgles2-mesa-dev libegl1-mesa-dev libffi-dev \
  libxcursor-dev libvulkan-dev libpipewire-0.3-dev
git clone https://github.com/jexme/scrawl.git && cd scrawl
make install            # compila e instala em ~/.local (PREFIX=/usr/local para todo o sistema)
```

Você precisa do Go (a versão de `go.mod`). Sem `sudo`? `scripts/fetch-pipewire-headers.sh` extrai os cabeçalhos do PipeWire em `.deps/` e o Makefile os usa.

Confira se `~/.local/bin` está no seu `PATH`.

### Windows 10 e 11

**Recomendado: o instalador.** Baixe `scrawl-VERSION-windows-amd64-setup.exe` na página de
[Releases](https://github.com/jexme/scrawl/releases) e execute-o. Ele instala apenas para a sua conta (sem direitos de administrador),
adiciona uma entrada no menu Iniciar, pode iniciar o Scrawl quando você entra na sessão e o inicia ao terminar. Para atualizar, execute o
instalador mais novo: ele fecha primeiro o Scrawl antigo. Se uma versão não tiver o arquivo de instalação, use uma das opções abaixo.

Alternativas:

- **Zip portátil.** Baixe `scrawl-VERSION-windows-amd64.zip` e o respectivo `.sha256`, descompacte em uma pasta permanente (não
  direto do visualizador de arquivos compactados) e execute `scrawl.exe`. Dar um clique duplo em `scrawl.exe` inicia o ícone da área de notificação.
  Um único `scrawl-VERSION-windows-amd64.exe` também é publicado. Para atualizar, feche o Scrawl pelo menu do ícone e substitua o arquivo.
- **[Scoop](https://scoop.sh):** `scoop install https://github.com/jexme/scrawl/releases/download/v0.1.0-rc3/scrawl.json`.
  Esse endereço fixa uma pré-versão específica, então entre uma versão candidata e outra você atualiza com `scoop uninstall scrawl` e `scoop install`
  com a URL da nova versão (até haver uma versão estável ou um bucket). Quando o manifesto for servido a partir de
  `https://github.com/jexme/scrawl/releases/latest/download/scrawl.json` (existir uma versão estável), `scoop update scrawl` funcionará.

- **O ícone** fica na área de notificação; no Windows 11 ele pode ficar escondido atrás da seta `^` ao lado do relógio: arraste-o para fora para mantê-lo visível.
- **A tecla Print** é a tecla de atalho padrão. Muitas vezes outro programa a mantém ocupada (Greenshot, ShareX, Lightshot). O próprio Windows também pode:
  Configurações, Acessibilidade, Teclado, *Usar o botão Print Screen para abrir a captura de tela*: desative. Ou escolha outra tecla em
  `scrawl.exe -settings`, por exemplo `<Control><Alt>a`. O ícone avisa quando a tecla está ocupada e continua tentando.
- **“O Windows protegeu seu computador”** (SmartScreen) aparece porque o programa ainda não é assinado: clique em *Mais informações* e depois em
  *Executar assim mesmo*. Para verificar o arquivo antes, compare `Get-FileHash` com o arquivo `.sha256`, ou execute
  `gh attestation verify scrawl-*-setup.exe --repo jexme/scrawl` (a compilação vem de uma CI pública). Antivírus podem sinalizar
  qualquer programa novo em Go sem assinatura; relate esse caso como uma issue.
- **O início automático** é a caixa de seleção em `scrawl.exe -settings`, aba *Sistema* (um valor em `HKCU\...\Run`), ou uma caixa no instalador.
  Se você mover ou atualizar o programa, a próxima inicialização do ícone reescreve a entrada com o novo local (um início automático desativado continua desativado).
- **Iniciar uma versão mais nova enquanto uma mais antiga está em execução:** o Scrawl informa qual versão está rodando; feche-a primeiro pelo
  menu do ícone e depois inicie a nova.

Compilação a partir do código-fonte e o que falta: [docs/WINDOWS.md](../WINDOWS.md).

### 2. Primeira execução: três passos, uma única vez

*No X11 os passos 1 e 2 não são necessários (não há portal), apenas a tecla de atalho.*

Abra a janela de configurações (`scrawl -settings`, aba “Sistema”). O botão “Configurar a captura” faz os passos 1 e 2 de uma vez (na primeira execução a janela abre sozinha). Ou use os botões separados ou execute os comandos:

1. **Escolher os monitores:** botão “Escolher monitores de novo” ou `scrawl -setup`. O GNOME mostra uma caixa de diálogo **por monitor**. Em cada uma, escolha um monitor *diferente*, mantenha marcada a opção **“Lembrar esta escolha”** e confirme o compartilhamento. Depois disso, nenhuma caixa de diálogo.
2. **Conceder a permissão de reserva:** botão “Conceder permissão” ou `scrawl -grant-screenshot`. Isso permite que o Scrawl use o portal de capturas, mais lento, nas raras vezes em que o PipeWire não entrega imagem.
3. **Definir uma tecla de atalho:** no GNOME, o instalador do arquivo de versão (`install.sh`) e o `make install` atribuem `Print` por conta própria, a menos que outro atalho personalizado já a use. Depois de instalar com apt ou com o `.deb`, ou para mudar a tecla: o campo “Tecla de atalho de abertura” e o botão “Atribuir” (padrão `Print`), ou `scripts/gnome-hotkey.sh [TECLA]`. Ele cria um atalho personalizado do GNOME chamado “scrawl” (ou o atualiza se já existir, então executá-lo de novo nunca cria um segundo atalho) e nunca sobrescreve o atalho de outra pessoa se a combinação estiver ocupada. Se `Print` ainda abrir a ferramenta de captura do GNOME, desative-a em *Configurações → Teclado → Ver e personalizar atalhos → Capturas de tela*.

### Atualizando a partir do screen-editor

Scrawl é o novo nome do screen-editor. Suas configurações são lidas de `~/.config/screen-editor/config.json` até que `~/.config/scrawl/config.json` exista (o arquivo antigo não é alterado). O portal associa a escolha de monitores e as permissões ao identificador do aplicativo, que mudou; por isso repita os passos 1 a 3 uma vez e aponte seu atalho para `scrawl`. A última área e a coleção começam do zero.

## Como usar

1. Aperte sua tecla de atalho. A tela congela e escurece.
2. Arraste para selecionar uma área (em qualquer monitor). Uma barra de ferramentas aparece embaixo dela.
3. Se quiser, anote: escolha uma ferramenta e uma cor e desenhe sobre a área.
4. Pressione `Enter`. A imagem está na área de transferência e em `~/Pictures/Screenshots`; cole onde quiser.

`Esc` cancela a qualquer momento.

### Teclas

| Tecla | Ação |
| --- | --- |
| `Enter` | copiar para a área de transferência **e** salvar em arquivo (o que ela faz é configurável) |
| `Ctrl+C` / `Ctrl+S` | só copiar / só salvar |
| `Ctrl+Z` | desfazer a última anotação |
| `Esc` | cancelar (primeiro fecha um menu aberto; enquanto você digita um texto, descarta apenas o texto) |
| `Ctrl+A` | selecionar a tela inteira do monitor (não pode ser remapeada) |
| `R` | restaurar a área da captura anterior |
| `V` `P` `L` `W` `B` `M` `U` `T` `N` | ferramentas: selecionar, caneta, linha, seta, caixa, marca-texto, desfoque, texto, carimbo numerado (quando uma área está selecionada) |
| `+` / `-` | espessura, tamanho do texto ou intensidade do desfoque da ferramenta atual (como a roda do mouse) |
| `A` | adicionar a captura à coleção |
| `G` | unir a coleção (e a área atual) em uma única imagem |
| `D` | salvar a coleção como arquivos separados e colocar a lista de arquivos na área de transferência |
| roda do mouse | espessura, tamanho do texto ou intensidade do desfoque da ferramenta atual |
| `Shift` ao desenhar | caneta e marca-texto: uma linha reta; linha e seta: se ajustam a 45°; caixa e desfoque: um quadrado. `Shift` ao selecionar uma área: um quadrado |

Todas as teclas podem ser alteradas na janela de configurações (aba “Teclas”).

### Barra de ferramentas

A linha principal, da esquerda para a direita: selecionar, caneta, linha, seta, caixa, marca-texto, desfoque, texto, carimbo
numerado · desfazer · copiar · salvar · o menu `⋯` (configurações, que fecha a sobreposição e abre a janela
de configurações, e adicionar à coleção) · fechar.

Enquanto uma ferramenta de desenho está ativa, aparece uma segunda linha embaixo: oito cores, um botão arco-íris para
a sua própria cor (abre uma paleta; a cor é lembrada) e `−` espessura `+`. A ferramenta “selecionar”
não tem nenhum dos dois, então a linha fica oculta.

Os botões que têm uma tecla a mostram embaixo do ícone (as teclas vêm das configurações). Deixe o cursor sobre um botão por um instante para ver seu nome e sua tecla. Os atalhos funcionam em qualquer layout de teclado (um layout russo, por exemplo), pela posição da tecla.

### Coleções antes/depois

Faça a primeira captura e pressione `A`. Faça a seguinte: a área anterior já está selecionada, então você fotografa o mesmo lugar de novo. Pressione `G` para unir as capturas em uma imagem (ela é salva e copiada) ou `D` para obter arquivos separados. O pequeno painel na parte de baixo da tela mostra o que está na coleção e tem as mesmas ações como botões.

## Ícone na bandeja e início automático

`scrawl -tray` mantém um ícone na área de notificação: o clique esquerdo tira uma captura e o menu tem “Fazer uma captura”, “Configurações” e “Sair”. Para iniciar no login, ative *Configurações → Sistema → Inicialização automática*. No GNOME, o ícone precisa da extensão AppIndicator (ativada por padrão no Ubuntu).

## Configurações

![A janela de configurações](../images/settings-en.png)

Abra pela barra de ferramentas (engrenagem), com `scrawl -settings` ou pelo menu de aplicativos. As alterações são salvas na hora em `~/.config/scrawl/config.json`, um arquivo JSON simples que você também pode editar à mão. Valores ausentes ou inválidos voltam ao padrão. Cada chave está descrita na [referência de configurações](../SETTINGS.md) (em inglês ou [russo](SETTINGS.ru.md)).

## Arquivos e privacidade

- As imagens vão para `~/Pictures/Screenshots` (mude nas configurações), com nomes como `scrawl-20260131-142501.png`.
- Configurações: `~/.config/scrawl/`. Tokens dos monitores, última área e coleção: `~/.cache/scrawl/`.
- O Scrawl nunca acessa a internet por conta própria. A única requisição de rede do programa é a verificação de atualizações, opcional e desativada por padrão: ela pergunta ao GitHub qual é a última versão e, além dos cabeçalhos HTTP habituais, envia apenas a versão do programa.
- As capturas não saem do seu computador a menos que você as cole em outro lugar.

## Linha de comando

| Opção | O que faz |
| --- | --- |
| `-settings` | abre a janela de configurações |
| `-setup` | escolhe os monitores de novo (diálogos do GNOME) |
| `-grant-screenshot` | concede a permissão de captura de reserva e sai |
| `-tray` | mantém o ícone na área de notificação |
| `-fresh` | não oferece a área da captura anterior |
| `-v` | mostra os tempos |
| `-version` | mostra a versão e sai |
| `-exit-after DURATION` | sai normalmente após o tempo indicado, para testes (por exemplo `-exit-after 3s`) |

## Solução de problemas

- **Nada acontece ao apertar a tecla de atalho.** O `xdg-desktop-portal` pode ter travado. Reinicie-o: `systemctl --user restart xdg-desktop-portal xdg-desktop-portal-gnome`.
- **O GNOME pergunta qual tela compartilhar a cada captura.** A opção “Lembrar esta escolha” não foi marcada; repita o passo 1.
- **Um monitor foi adicionado ou trocado.** Escolha os monitores de novo (passo 1).
- **A sobreposição abre em apenas um monitor.** O mesmo: o passo 1 precisa ser feito para cada monitor.
- **A mensagem diz que o portal recusou.** Conceda a permissão (passo 2).
- **“Clipboard error” ou colar não traz nada.** Verifique se o `wl-copy` está instalado (pacote `wl-clipboard`).
- **Sem notificações.** Instale o `libnotify-bin` (o comando `notify-send`) e verifique se as notificações estão ativadas nas configurações.
- **Sem ícone na bandeja.** Ative a extensão AppIndicator do GNOME.
- **`Print` abre a ferramenta de captura do GNOME.** Desative-a nos atalhos de teclado (veja o passo 3).

## Desinstalar

**apt ou `.deb`:**

```bash
sudo apt remove scrawl     # ou: sudo apt purge scrawl (o pacote não deixa arquivos fora de /usr, então dá no mesmo)
sudo rm /etc/apt/sources.list.d/scrawl.sources   # opcional: esquecer o repositório (scrawl.list na forma antiga)
```

O apt não pode mexer na sua pasta pessoal, então sobram alguns resíduos por usuário: o atalho do GNOME, a entrada de início automático, se
você a ativou, e as suas configurações. Remova o atalho com o `gnome-hotkey.sh` do Scrawl (ele está em `scripts/` do
repositório e no arquivo de versão) ou à mão em *Configurações → Teclado → Ver e personalizar atalhos → Atalhos
personalizados*; o resto são arquivos comuns:

```bash
sh gnome-hotkey.sh --remove
rm -f ~/.config/autostart/io.github.jexme.Scrawl.desktop
rm -rf ~/.config/scrawl ~/.cache/scrawl     # opcional: configurações e cache
```

**Arquivo de versão ou `make install`:** execute `./uninstall.sh` a partir do arquivo descompactado (`make uninstall` em uma cópia do
código-fonte). Ele remove o binário, o arquivo `.desktop`, o atalho do GNOME e a entrada de início automático. Acrescente `--purge`
(`make uninstall PURGE=1`) para apagar também as configurações (`~/.config/scrawl`) e o cache (`~/.cache/scrawl`). Passe
o mesmo `PREFIX` da instalação, se você o mudou. Suas capturas em `~/Pictures/Screenshots` nunca são tocadas.

## Como funciona

O Scrawl captura a tela pela interface ScreenCast do `xdg-desktop-portal`, com seu próprio cliente PipeWire (`internal/pwcapture`). Um token de restauração salvo para cada monitor elimina a caixa de diálogo. O compositor (Mutter) só entrega quadros quando o monitor é redesenhado e, nos nossos testes, não entregou nenhum sobre uma janela em tela cheia (saída direta); se nenhum quadro chegar em 250 ms, o Scrawl recorre ao portal de capturas, que é mais lento (cerca de 0,5-1 s). As sobreposições são janelas de [Gio](https://gioui.org) em tela cheia, uma por monitor; o Gio recebe um patch (`third_party/gio.patch`) para colocar uma janela em uma saída específica do Wayland. O resultado é desenhado pelo mesmo código que o exibe na tela, então a imagem salva é igual ao que você viu.

## Compilação e contribuição

```bash
make build    # bin/scrawl; baixa e aplica o patch no Gio na primeira execução
make test     # testes unitários, incluindo o ícone da bandeja no D-Bus da sessão
make dist     # arquivo de versão e soma de verificação em dist/
make deb      # pacote .deb para Debian e Ubuntu em dist/
make notices  # regenera o THIRD_PARTY_NOTICES.md
```

Os textos da interface ficam em `internal/i18n/` (um arquivo por idioma, `lang_ru.go`, `lang_de.go` e assim por diante; os textos originais no código estão em inglês);
`go test ./...` verifica se cada texto está traduzido para cada idioma. Como relatar um bug e enviar uma alteração: [CONTRIBUTING.md](../../CONTRIBUTING.md). Plano e ideias: [docs/PLAN.md](../PLAN.md) (em russo).

## Licença

MIT, veja [LICENSE](../../LICENSE). Componentes de terceiros: [THIRD_PARTY_NOTICES.md](../../THIRD_PARTY_NOTICES.md) (as formas dos ícones são de [Tabler Icons](https://tabler.io/icons), MIT).
