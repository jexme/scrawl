# Scrawl

**Capturas de tela rápidas com anotações para o GNOME no Wayland.** Aperte uma tecla, selecione uma área em qualquer monitor, desenhe setas, caixas e texto, desfoque dados privados, e o resultado já está na área de transferência e salvo em arquivo.

[English](../../README.md) · [Русский](../../README.ru.md) · [简体中文](README.zh-CN.md) · [Español](README.es.md) · **Português (Brasil)** · [日本語](README.ja.md) · [Deutsch](README.de.md) · [Français](README.fr.md) · [한국어](README.ko.md)

![Sobreposição do Scrawl: uma área selecionada com uma caixa, uma seta, um texto, um traço de marca-texto e um trecho desfocado, com a barra de ferramentas abaixo](../images/overlay.png)

*A imagem é uma renderização real da sobreposição sobre uma área de trabalho inventada.*

> A interface está disponível em inglês, russo, chinês simplificado, espanhol, português, japonês, alemão, francês e coreano; os nomes dos botões abaixo são os da interface em inglês, com seus equivalentes em cada idioma.

## Por que o Scrawl

No GNOME com Wayland, os aplicativos não conseguem capturar a tela diretamente: eles pedem ao portal da área de trabalho, e o portal **pergunta “qual tela você quer compartilhar?” a cada captura**. O Scrawl pergunta uma vez por monitor e lembra da resposta; depois disso, uma captura é um único toque de tecla: a tela congelada aparece em cerca de 150 ms.

## Recursos

**Captura**
- Funciona em todos os monitores ao mesmo tempo: a sobreposição abre em cada um e você pode selecionar em qualquer um.
- Seleção com moldura tracejada. Arraste os cantos para redimensionar; com a ferramenta *selecionar*, arraste uma borda para redimensionar ou o interior para mover a área.
- Lembra da última área e a traz de volta com uma tecla (`R`).
- Opcionalmente, inclui o cursor do mouse na captura.

**Anotações**
- Caneta, linha, seta, caixa, marca-texto, **desfoque** e texto; oito cores.
- A roda do mouse (ou os botões `−` / `+`) muda a espessura, o tamanho do texto ou a intensidade do desfoque.
- Desfazer com `Ctrl+Z`.

**Salvar e compartilhar**
- `Enter` copia a imagem para a área de transferência e salva um PNG em `~/Pictures/Screenshots`. `Ctrl+C` só copia, `Ctrl+S` só salva. JPEG também está disponível.
- **Coleções antes/depois:** separe capturas com `A` e una-as em uma única imagem com legendas (“Before”, “After” ou “Step 1…” para mais), lado a lado ou uma sobre a outra. Ou envie a coleção como arquivos separados, que são colados em um chat como anexos.

**Do seu jeito**
- Uma janela de configurações para quase tudo: teclas, cores, ferramentas padrão, aparência da seleção, nome e formato dos arquivos e idioma da interface (9 idiomas).
- Ícone na bandeja com início automático, um botão para definir a tecla de atalho e uma verificação de atualizações opcional (desativada por padrão).

## Status

Projeto pessoal em desenvolvimento ativo. **Ainda não há nenhuma versão publicada**, então por enquanto é preciso compilar a partir do código-fonte (leva alguns minutos). O alvo principal é o GNOME no Wayland (Linux); as sessões **X11** (GNOME no Xorg, KDE, XFCE e outras) também funcionam, mas esse suporte é novo e até agora só foi testado em um servidor X virtual; relatos de máquinas reais são bem-vindos. Existe uma compilação para Windows, mas a captura de tela ainda não foi escrita; veja [docs/WINDOWS.md](../WINDOWS.md).

## Requisitos

- GNOME no Wayland com PipeWire, `xdg-desktop-portal` (1.21 ou mais recente, com suporte ao registro de aplicativos) e `xdg-desktop-portal-gnome`. Desenvolvido e testado no Ubuntu com GNOME 50.
- `wl-clipboard` (o comando `wl-copy`) para a área de transferência e `libnotify` (o comando `notify-send`) para as notificações.
- No **X11** não há portal: a tela é capturada diretamente e a área de transferência precisa do `xclip` em vez do `wl-clipboard`.

```bash
sudo apt install wl-clipboard libnotify-bin   # Wayland
sudo apt install xclip libnotify-bin          # X11
```

## Instalação

### 1. Obtenha o programa

**A partir do código-fonte (funciona hoje):**

```bash
sudo apt install gcc pkg-config patch libwayland-dev libx11-dev libx11-xcb-dev \
  libxkbcommon-x11-dev libgles2-mesa-dev libegl1-mesa-dev libffi-dev \
  libxcursor-dev libvulkan-dev libpipewire-0.3-dev
git clone https://github.com/jexme/scrawl.git && cd scrawl
make install            # compila e instala em ~/.local (PREFIX=/usr/local para todo o sistema)
```

Você precisa do Go (a versão de `go.mod`). Sem `sudo`? `scripts/fetch-pipewire-headers.sh` extrai os cabeçalhos do PipeWire em `.deps/` e o Makefile os usa.

**A partir de um arquivo de versão (quando houver uma publicada):** baixe `scrawl-VERSION-linux-amd64.tar.gz` e `SHA256SUMS` na página de [Releases](https://github.com/jexme/scrawl/releases) e execute:

```bash
sha256sum -c --ignore-missing SHA256SUMS
tar xzf scrawl-*-linux-amd64.tar.gz && cd scrawl-*-linux-amd64
./install.sh            # instala em ~/.local (PREFIX=/usr/local para todo o sistema)
sudo apt install libpipewire-0.3-0 libwayland-client0 libwayland-egl1 libwayland-cursor0 \
  libxkbcommon-x11-0 libegl1 libx11-6 libxcursor1
```

Confira se `~/.local/bin` está no seu `PATH`.

### 2. Primeira execução: três passos, uma única vez

*No X11 os passos 1 e 2 não são necessários (não há portal), apenas a tecla de atalho.*

Abra a janela de configurações (`scrawl -settings`, tab “System”) e use estes botões, ou execute os comandos:

1. **Escolher os monitores:** botão “Pick monitors again” ou `scrawl -setup`. O GNOME mostra uma caixa de diálogo **por monitor**. Em cada uma, escolha um monitor *diferente*, mantenha marcada a opção **“Lembrar esta escolha”** e confirme o compartilhamento. Depois disso, nenhuma caixa de diálogo.
2. **Conceder a permissão de reserva:** botão “Grant permission” ou `scrawl -grant-screenshot`. Isso permite que o Scrawl use o portal de capturas, mais lento, nas raras vezes em que o PipeWire não entrega imagem.
3. **Definir uma tecla de atalho:** o campo “Launch hotkey” e o botão “Assign” (padrão `Print`), ou `scripts/gnome-hotkey.sh [TECLA]`. Ele cria um atalho personalizado do GNOME e nunca sobrescreve uma combinação já ocupada. Se `Print` ainda abrir a ferramenta de captura do GNOME, desative-a em *Configurações → Teclado → Ver e personalizar atalhos → Capturas de tela*.

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
| `Esc` | cancelar |
| `R` | restaurar a área da captura anterior |
| `A` | adicionar a captura à coleção |
| `G` | unir a coleção (e a área atual) em uma única imagem |
| `D` | salvar a coleção como arquivos separados e colocar a lista de arquivos na área de transferência |
| roda do mouse | espessura, tamanho do texto ou intensidade do desfoque da ferramenta atual |

Todas as teclas podem ser alteradas na janela de configurações (“Keys”).

### Barra de ferramentas

Da esquerda para a direita: selecionar, caneta, linha, seta, caixa, marca-texto, desfoque, texto · oito cores · `−` espessura `+` · configurações (fecha a sobreposição e abre a janela de configurações) · adicionar à coleção · desfazer · copiar · salvar · fechar.

### Coleções antes/depois

Faça a primeira captura e pressione `A`. Faça a seguinte: a área anterior já está selecionada, então você fotografa o mesmo lugar de novo. Pressione `G` para unir as capturas em uma imagem (ela é salva e copiada) ou `D` para obter arquivos separados. O pequeno painel na parte de baixo da tela mostra o que está na coleção e tem as mesmas ações como botões.

## Ícone na bandeja e início automático

`scrawl -tray` mantém um ícone na área de notificação: o clique esquerdo tira uma captura e o menu tem “Screenshot”, “Settings” e “Quit”. Para iniciar no login, ative *Settings → System → Autostart*. No GNOME, o ícone precisa da extensão AppIndicator (ativada por padrão no Ubuntu).

## Configurações

![A janela de configurações](../images/settings-en.png)

Abra pela barra de ferramentas (engrenagem), com `scrawl -settings` ou pelo menu de aplicativos. As alterações são salvas na hora em `~/.config/scrawl/config.json`, um arquivo JSON simples que você também pode editar à mão. Valores ausentes ou inválidos voltam ao padrão. Cada chave está descrita na [referência de configurações](../SETTINGS.md) (em inglês ou [russo](../SETTINGS.ru.md)).

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

## Solução de problemas

- **Nada acontece ao apertar a tecla de atalho.** O `xdg-desktop-portal` pode ter travado. Reinicie-o: `systemctl --user restart xdg-desktop-portal xdg-desktop-portal-gnome`.
- **O GNOME pergunta qual tela compartilhar a cada captura.** A opção “Lembrar esta escolha” não foi marcada; repita o passo 1.
- **Um monitor foi adicionado ou trocado.** Escolha os monitores de novo (passo 1).
- **A sobreposição abre em apenas um monitor.** O mesmo: o passo 1 precisa ser feito para cada monitor.
- **A mensagem diz que o portal recusou.** Conceda a permissão (passo 2).
- **Colar em algum aplicativo não traz nada.** Verifique se o `wl-copy` está instalado (pacote `wl-clipboard`).
- **Sem notificações.** Instale o `libnotify-bin` (o comando `notify-send`) e verifique se as notificações estão ativadas nas configurações.
- **Sem ícone na bandeja.** Ative a extensão AppIndicator do GNOME.
- **`Print` abre a ferramenta de captura do GNOME.** Desative-a nos atalhos de teclado (veja o passo 3).

## Como funciona

O Scrawl captura a tela pela interface ScreenCast do `xdg-desktop-portal`, com seu próprio cliente PipeWire (`internal/pwcapture`). Um token de restauração salvo para cada monitor elimina a caixa de diálogo. O compositor (Mutter) só entrega quadros quando o monitor é redesenhado e, nos nossos testes, não entregou nenhum sobre uma janela em tela cheia (saída direta); se nenhum quadro chegar em 250 ms, o Scrawl recorre ao portal de capturas, que é mais lento (cerca de 0,5-1 s). As sobreposições são janelas de [Gio](https://gioui.org) em tela cheia, uma por monitor; o Gio recebe um patch (`third_party/gio.patch`) para colocar uma janela em uma saída específica do Wayland. O resultado é desenhado pelo mesmo código que o exibe na tela, então a imagem salva é igual ao que você viu.

## Compilação e contribuição

```bash
make build    # bin/scrawl; baixa e aplica o patch no Gio na primeira execução
make test     # testes unitários, incluindo o ícone da bandeja no D-Bus da sessão
make dist     # arquivo de versão e soma de verificação em dist/
make notices  # regenera o THIRD_PARTY_NOTICES.md
```

Os textos da interface ficam em `internal/i18n/` (um arquivo por idioma; os textos originais no código estão em russo); `go test ./...` verifica se cada texto está traduzido para cada idioma. Plano e ideias: [docs/PLAN.md](../PLAN.md) (em russo).

## Licença

MIT, veja [LICENSE](../../LICENSE). Componentes de terceiros: [THIRD_PARTY_NOTICES.md](../../THIRD_PARTY_NOTICES.md) (as formas dos ícones são de [Tabler Icons](https://tabler.io/icons), MIT).
