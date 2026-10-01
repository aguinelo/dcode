# Desktop usável, ligado ao daemon

Ao fim deste loop, o desktop serve para trabalhar no dia a dia. Ele é cliente de
um `dcode serve` de verdade, como a TUI: anexa ao daemon do socket ou sobe o seu,
mostra na lateral todas as conversas, as vivas e as que terminaram, abre uma nova,
conversa, redireciona o turno, interrompe, responde aprovações — com `↵` negando
—, continua uma conversa antiga e acha qualquer uma pelo ⌘K. O porquê está em
`desktop/docs/DECISIONS.md` (D19–D23); a régua é `npm run check:daemon`, em
`desktop/tests/daemon/`, e a janela continua igual ao design (`check:visual`).

O ciclo anterior, a janela sobre eventos gravados, está pronto: os critérios dele
continuam aqui como regressão.

## Antes de rodar

- **O núcleo já tem o que este loop usa:** `dcode socket` (N1), a configuração
  de cada projeto (N2) e a lista de conversas com o fluxo dela (N3).
- **Acesso total.** Os critérios rodam dentro do sandbox da sessão do loop. O
  `check:daemon` sobe um `dcode serve`, que abre o próprio sandbox, e o macOS não
  aninha sandbox; e sobe um Electron, que precisa da sessão gráfica. Em
  `workspace-write`, o critério reprova por isso, não pelo código.
- **Na máquina:** Go, git e o Chromium do Playwright
  (`npx playwright install chromium`).

## O que construir, nesta ordem

1. **O cliente do daemon, no processo principal** (`src/main/`). HTTP sobre o
   socket Unix, o SSE de cada sessão aberta desde o seq 1, retomando do último seq
   quando o fluxo cai, e o fluxo único da lista (`GET /v1/conversations/events`),
   que abre com o retrato e manda só o que muda. O renderer continua sem Node e
   sem socket: recebe por IPC e pede cada ação por uma função com nome no
   `DcodeApi` — nenhum canal genérico.
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
4. **A lateral com todas as conversas** (N3): as vivas, com o estado ao vivo, e as
   gravadas, por projeto, pela atividade mais recente, com o título de cada uma.
   Vem da lista e do fluxo dela, nunca de um SSE por sessão nem do disco.
5. **Abrir uma conversa.** Uma viva abre o fluxo dela. Uma gravada é continuada
   numa sessão nova (`CreateSession{Resume}`), que a janela abre — a conversa
   antiga aparece carregada, e a nova segue dali.
6. **Nova sessão**, pelo botão da lateral e pelo ⌘N. Pede a pasta por
   `dialog.showOpenDialog`, cria a sessão nela com os padrões do daemon e a abre.
7. **O campo de mensagem.**
   - Com a sessão ociosa, envia um turno.
   - Com o turno rodando, redireciona (`POST …/steer`, D23), e o texto do campo diz
     "Escreva para redirecionar este turno".
   - O campo esvazia quando o daemon aceitou, e guarda o texto quando recusou,
     dizendo por quê.
   - Parar é `POST …/interrupt`.
8. **A aprovação** (D22), pela rota de aprovações.
   - `1` permite uma vez e `2` nesta sessão; `3`, `esc` e `↵` negam.
   - A opção destacada é negar.
   - Enquanto a aprovação espera, o campo fica desabilitado, com o texto
     "Responda à aprovação acima".
9. **O ⌘K** procura entre todas as conversas pelo título e pelo projeto; `↵` abre
   a primeira, viva ou gravada, pelas mesmas regras do item 5.
10. **Toda falha dita.**
    - Um pedido recusado mostra a mensagem do daemon — inclusive a de um projeto
      cuja configuração não se lê, que recusa a sessão (N2).
    - Um fluxo que cai tenta retomar.
    - Com o daemon morto, a barra diz que caiu e o envio para.
11. **Os documentos, no mesmo ciclo.** Em `DECISIONS.md`, D2, D5, A5 e L13 passam a
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
| `data-session-id` e `data-state` | Na linha de cada conversa da lateral, com o título dela no texto. O estado é o do fio: `idle`, `running`, `blocked`, ou `recorded` para uma que terminou. |
| `aria-current="true"` | Na linha da conversa aberta na janela. |
| `data-approval-id` e `data-decision` | No card de aprovação. A decisão é a do fio, depois de respondida. |
| `main textarea` | O campo de mensagem da sessão aberta, `disabled` enquanto uma aprovação espera. |
| O botão "Nova sessão" | Na lateral. A pasta vem de `dialog.showOpenDialog`, chamado pelo objeto `dialog`, que a régua substitui. |
| ⌘K | Abre a busca com o foco no campo dela; o que se digita filtra; `↵` abre o primeiro resultado. |
| `data-ready` | No `<html>`, como hoje: `true` quando a janela está pronta. |
| `DCODE_SOCKET`, `DCODE_BIN` e `DCODE_DESKTOP_USER_DATA` | No ambiente do app. O último é a pasta de dados do Electron (`app.setPath('userData')`, antes do `ready`). |

Os cenários, o modelo roteirizado e o daemon da régua estão em
`desktop/tests/daemon/`. `npm run check:daemon -- <nome>` roda só um cenário. Cada
falha deixa uma captura da janela e os logs em `desktop/test-results/daemon/`.

## Fora deste loop

- A troca de modelo (continuar com outro modelo), e a tela do loop, que espera o
  N4.
- Imagens e comandos no campo de mensagem.
- Empacotar, assinar, e o ambiente do shell de login para o app aberto pelo Dock.
