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

**D41. Onde a conversa trabalha: sessão, pasta, branch de agora, worktree e o
diff da pasta** (emenda D8, D33). Relato de uso: "não fica claro em que sessão
estou, pasta de projeto, branch, worktree e diff, tá bem cru". A conversa em
foco mostrava o título e o nome curto da pasta; a branch era a `Session.branch`,
lida uma vez no início da sessão; e o único diff era o que as ferramentas
disseram ter escrito (D8).

- **O git vem do daemon** (`GET /v1/git` e `GET /v1/git/diff`, núcleo,
  `202610131900`), e não do processo principal rodando `git` com `execFile`. O
  daemon já lê git com o ambiente fixado e testado contra repositório de
  verdade, pode estar noutra máquina que não a da janela, e uma segunda leitura
  aqui seria a segunda cópia das regras de branch destacada, repositório sem
  commit e worktree. O renderer continua sem tocar em sistema de arquivos.
- **A janela nomeia a conversa, nunca a pasta.** Como o terminal (D37), o
  processo principal acha a pasta na lista do daemon pelo id da conversa — viva
  ou terminada — e só então pergunta. O arquivo de um diff é o que a lista do
  git deu, e o daemon recusa qualquer outro.
- **Crew:** debaixo do título, uma linha — `#id · pasta · branch de agora ·
  worktree de <checkout> · +a −r` —, cada parte inteira ao passar o ponteiro. Na
  direita, **Mudanças** lista o que a pasta tem desde o último commit, arquivo a
  arquivo com as linhas; abrir um mostra o diff unificado, cortado onde o daemon
  cortou e dizendo isso. Os arquivos que esta conversa escreveu levam uma marca,
  e embaixo continua a lista do que as ferramentas disseram (D8), com o nome que
  tem: o que *esta sessão* escreveu, não a pasta. **Sessão** abre com o id (com
  copiar), viva ou terminada, a pasta (com copiar), a branch de agora — e, se a
  pasta mudou de branch depois, em qual a sessão começou —, worktree sim/não e
  de qual checkout, o topo do repositório quando não é a pasta, e o commit.
- **Grade:** o chip da branch da janela maximizada e o rodapé de cada janela da
  grade abrem um cartão com os mesmos fatos e a mesma lista. Nada muda no que a
  grade desenha em repouso: as telas de referência medem 0,00% (02 e 03).
- **Quando relê:** ao mostrar, quando uma chamada de ferramenta termina ou a
  conversa muda de estado (não a cada fragmento de texto), quando a janela volta
  para a frente, e no botão de reler. Enquanto relê, a última resposta fica — uma
  lista que pisca vazia leria "nada mudou".
- **Toda falha é dita onde se olha.** Pasta fora de repositório diz "sem git" e
  por que isso importa; git ausente ou falhando mostra o motivo do daemon; um
  daemon sem a rota diz que é mais antigo que ela (como as skills e a memória);
  um diff que não veio diz por quê. Nenhum desses é uma lista vazia.
- **A gravação** (modo fixture) responde o git de cada pasta gravada: dcode numa
  worktree com trabalho, bizpack limpo, eva fora de repositório.

Descartados: rodar `git` no processo principal (acima); ler por sessão, em
`/sessions/{id}/git` (a conversa terminada, que é quando se quer ver o que
ficou na pasta, não tem sessão viva); pôr a linha de contexto na janela
maximizada da grade (muda a tela de referência; o chip já existe e abre o
cartão); diff contra a branch principal, como um PR (precisa decidir a base, e
fica para quando alguém pedir).

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
