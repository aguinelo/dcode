# Decisões do desktop — D39 em diante

A continuação de [`DECISIONS-2.md`](DECISIONS-2.md), que vai de D21 a D40, e de
[`DECISIONS.md`](DECISIONS.md), que tem a introdução, D1 a D20, as decisões em
aberto, as lacunas e os pedidos ao núcleo. A numeração é uma só; D39 veio para cá
porque o DECISIONS-2 passaria de 500 linhas com ela.

**D39. A aprovação oferece as respostas que a TUI oferece** (emenda D22). A
janela oferecia sempre uma vez, nesta sessão e negar; o protocolo tem cinco
respostas, e a TUI escolhe quais pela travessia (`approvalKeys`,
`standingScope`). Agora a janela faz o mesmo:

- **Rede:** permitir uma vez, neste projeto, sempre, e negar — sem "nesta
  sessão", como na TUI. Só a rede tem resposta que sobrevive à sessão: uma
  concessão permanente de escrita fora do workspace seria o sistema de arquivos
  inteiro, dado uma vez por um motivo que ninguém registra.
- **Qualquer outra travessia:** permitir uma vez, nesta sessão, e negar.
- **Teclas:** de `1` a `n` na ordem mostrada; `↵` e `esc` negam, e negar continua
  a opção destacada (D22). As respostas que ficam gravadas — neste projeto,
  sempre — vêm no tom de aviso, como na TUI, com o título dizendo onde ficam.
- **A razão do daemon aparece no cartão**, abaixo do que se pede, e não só no
  título: é o que se lê antes de responder, e um título só aparece a quem passa o
  ponteiro.

A tela de referência 03 é uma travessia de rede, então ela muda com isto: quatro
opções no lugar de três, e a razão numa linha — 0,73% contra a captura anterior.
O dono viu a tela nova e autorizou trocar a 03 neste mesmo PR, só ela: foi
recapturada pela própria verificação, nas condições de sempre, e a 02 continua a
captura de 9656919, a 0,00%.

Descartados: manter as três para todas (a janela ofereceria "nesta sessão" para
a rede, que a TUI não oferece, e esconderia as duas respostas permanentes);
oferecer as cinco sempre (concessão permanente de escrita fora do projeto).

**D42. Uma conversa terminada abre no histórico dela** (emenda D21). Abrir uma
conversa que terminou mostrava a thread vazia e, no painel da direita do Crew,
"Uma conversa terminada não traz os eventos dela": a pessoa achou que as sessões
tinham se perdido. Elas estavam no disco; o protocolo só não tinha como lê-las
sem continuar. Com `GET /v1/conversations/{id}/events` (N7), a janela agora as lê:

- **Onde:** na janela da grade, na conversa maximizada e na thread do Crew — onde
  quer que uma conversa gravada, sem sessão viva, apareça. A lateral continua
  como D21: clicar numa gravada continua a conversa numa sessão nova.
- **Como:** o processo principal pede a rota página por página (um canal, com o
  id conferido no alfabeto do núcleo e a posição a partir de 1), e o renderer
  dobra os eventos pelo mesmo `applyEvent` de uma sessão viva — a conversa lida
  é a mesma que se via enquanto rodava, com o mesmo `FlowBlocks`. Só leitura: o
  cartão de aprovação não aparece (o núcleo não manda pergunta já decidida), e
  "Rodar no terminal" fica de fora; "Copiar" fica.
- **Abaixo:** a mesma linha de antes — "Escrever continua a conversa numa sessão
  nova, com o histórico" — e o campo que continua.
- **Mudanças:** o painel do Crew lê os arquivos do histórico, como lê os de uma
  sessão viva (`filesTouched`).
- **Enquanto lê:** "Lendo o histórico desta conversa…". **Se falha:** o motivo
  do daemon na thread, em vermelho, com "Tentar de novo", e também num aviso —
  nunca uma thread vazia como se nada tivesse acontecido. Daemon mais velho que a
  rota diz que não a expõe, com as palavras dele.
- **Longa demais:** até 20 páginas (100 mil eventos). Passando disso, mostra o
  que leu e diz quantos eventos ficaram de fora — o fim da conversa, já que a
  leitura vai do começo.

Cada conversa é lida uma vez por janela, quando aparece: uma gravada não muda.

Descartados: continuar a conversa só para mostrá-la (abriria uma sessão que
ninguém pediu); ler o arquivo do registro pelo processo principal (acoplaria o
desktop ao formato do disco, que é do núcleo); mostrar só as últimas páginas
(sem o começo, a pergunta que abriu a conversa some).
