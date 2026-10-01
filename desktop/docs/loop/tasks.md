# Desktop ligado ao daemon

A janela deixa de ser uma gravação e passa a ser cliente de um `dcode serve` de
verdade, como a TUI: anexa ao daemon do socket ou sobe o seu, lista as sessões
vivas, abre uma nova, conversa, redireciona o turno, interrompe e responde
aprovações — com `↵` negando. O porquê está em `desktop/docs/DECISIONS.md`
(D19–D23); a régua é `npm run check:daemon`, em `desktop/tests/daemon/`, e a
janela continua igual ao design (`check:visual`).

O ciclo anterior, a janela sobre eventos gravados, está pronto: os critérios dele
continuam aqui como regressão.

## Antes de rodar

- **O N1 no `main`** (`DECISIONS.md`, "Pedidos ao núcleo"): é o `dcode socket` que
  diz ao app o caminho do socket. A régua não depende dele, porque passa
  `DCODE_SOCKET`, mas o app de verdade depende.
- **Acesso total.** Os critérios rodam dentro do sandbox da sessão do loop. O
  `check:daemon` sobe um `dcode serve`, que abre o próprio sandbox, e o macOS não
  aninha sandbox; e sobe um Electron, que precisa da sessão gráfica. Em
  `workspace-write`, o critério reprova por isso, não pelo código.
- **Na máquina:** Go, git e o Chromium do Playwright
  (`npx playwright install chromium`).

## O que construir, nesta ordem

1. **O cliente do daemon, no processo principal** (`src/main/`). HTTP sobre o
   socket Unix, e o SSE de cada sessão viva desde o seq 1, retomando do último seq
   quando o fluxo cai. O renderer continua sem Node e sem socket: recebe os
   eventos por IPC e pede cada ação por uma função com nome no `DcodeApi` —
   nenhum canal genérico.
2. **Achar ou subir o daemon** (D19).
   - O socket é o de `DCODE_SOCKET`, senão o que `dcode socket` imprimir (N1). Se
     ele falhar, o motivo que ele deu é o que a barra diz.
   - O binário é o de `DCODE_BIN`, senão `~/.local/bin/dcode`, senão o do `PATH`.
   - Sobe `dcode serve --socket <caminho>` como filho e espera o `/health` com
     prazo.
   - Ao sair, encerra o filho, e nunca um daemon que não subiu. Fechar com sessão
     rodando ou esperando aprovação pergunta antes.
3. **O estado do daemon na barra inferior**, no lugar de `gravação`: conectando;
   conectado, com a versão do `GET /version`; caiu; não subiu. Os dois últimos
   dizem o porquê. O modo fixture (navegador, `?fixture=`) continua mostrando a
   gravação e dizendo `gravação`.
4. **A lateral com as sessões vivas do daemon**, inclusive as abertas por outro
   cliente depois da janela, cada uma com o estado ao vivo. Até o N3 trazer o
   fluxo da lista, a fonte é o `GET /v1/sessions`.
5. **Nova sessão**, pelo botão da lateral e pelo ⌘N. Pede a pasta por
   `dialog.showOpenDialog`, cria a sessão nela com os padrões do daemon e a abre.
6. **O campo de mensagem.**
   - Com a sessão ociosa, envia um turno.
   - Com o turno rodando, redireciona (`POST …/steer`, D23), e o texto do campo diz
     "Escreva para redirecionar este turno".
   - O campo esvazia quando o daemon aceitou, e guarda o texto quando recusou,
     dizendo por quê.
   - Parar é `POST …/interrupt`.
7. **A aprovação** (D22), pela rota de aprovações.
   - `1` permite uma vez e `2` nesta sessão; `3`, `esc` e `↵` negam.
   - A opção destacada é negar.
   - Enquanto a aprovação espera, o campo fica desabilitado, com o texto
     "Responda à aprovação acima".
8. **Toda falha dita.**
   - Um pedido recusado mostra a mensagem do daemon.
   - Um fluxo que cai tenta retomar.
   - Com o daemon morto, a barra diz que caiu e o envio para.
9. **Os documentos, no mesmo ciclo.** Em `DECISIONS.md`, D2, D5, A5 e L13 passam a
   dizer o que vale. O changelog do desktop, nas duas línguas.

As medidas do `check:visual` com os textos de D22 e D23 já foram tiradas: 1,20% na
tela 02 e 1,21% na 03, dentro do limite de 1,25%. Se passar disso, o que mudou foi
outra coisa.

## O que a régua lê

O `check:daemon` dirige a janela por cliques e teclas e lê só o que está abaixo.
O resto é livre.

| O quê | Onde |
|---|---|
| `data-daemon` | Na barra inferior: `connecting`, `connected`, `lost` ou `failed`. O texto do elemento diz a versão, quando conectado, ou o porquê. |
| `data-session-id` e `data-state` | Na linha de cada sessão da lateral. O estado é o do fio: `idle`, `running`, `blocked` ou `closed`. |
| `data-approval-id` e `data-decision` | No card de aprovação. A decisão é a do fio, depois de respondida. |
| `main textarea` | O campo de mensagem da sessão aberta, `disabled` enquanto uma aprovação espera. |
| O botão "Nova sessão" | Na lateral. A pasta vem de `dialog.showOpenDialog`, chamado pelo objeto `dialog`, que a régua substitui. |
| `data-ready` | No `<html>`, como hoje: `true` quando a janela está pronta. |
| `DCODE_SOCKET`, `DCODE_BIN` e `DCODE_DESKTOP_USER_DATA` | No ambiente do app. O último é a pasta de dados do Electron (`app.setPath('userData')`, antes do `ready`). |

Os cenários, o modelo roteirizado e o daemon da régua estão em
`desktop/tests/daemon/`. `npm run check:daemon -- <nome>` roda só um cenário. Cada
falha deixa uma captura da janela e os logs em `desktop/test-results/daemon/`.

## Fora deste loop

- As sessões gravadas, o ⌘K e a troca de modelo, que esperam o N3.
- A tela do loop, que espera o N4.
- Imagens e comandos no campo de mensagem.
- Empacotar, assinar, e o ambiente do shell de login para o app aberto pelo Dock.
