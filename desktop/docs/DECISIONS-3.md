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

**D43. Renomear uma conversa pela janela.** O daemon nomeia conversas desde
`POST /v1/sessions/{id}/name` (`RenameSessionRequest`), vivas ou terminadas, e
a janela nunca pedia: o título era sempre a primeira pergunta.

- **Onde:** duplo clique no título — da janela da grade, da maximizada e da
  thread do Crew — vira um campo no lugar dele, com a mesma classe, ocupando o
  mesmo espaço. "Renomear" no menu ⋯ (D40) abre o mesmo campo, e fica
  oferecido no meio de um turno: o daemon aceita o nome a qualquer hora. A
  lateral não tem menu por conversa, então não ganha um. Duplo clique no resto
  do cabeçalho da janela da grade continua maximizando; no título, renomeia.
- **Teclas:** Enter salva, Esc desiste, sair do campo salva. Nada mudou fecha
  sem pedir nada. Nome vazio é um nome: devolve o título automático. Enquanto o
  campo está aberto, nenhuma tecla vai para a janela (o campo para a
  propagação de todas, como o campo da mensagem faz com o Esc).
- **Não otimista:** o título que se vê é o do daemon. Uma conversa viva anuncia
  o nome em `session.renamed`, e a lista o recebe pelo stream dela. Uma
  terminada é nomeada no registro, e nenhum stream anuncia isso: o processo
  principal relê `GET /v1/conversations` depois do sim e passa a linha dela
  adiante como a mudança que ela é. Se a releitura falha, a janela diz que o
  daemon guardou o nome e que o título aparece quando a lista mudar — nunca um
  sim calado com o título velho.
- **Recusa:** um aviso com o motivo do daemon, e o campo continua aberto com o
  texto, com o cursor nele. Sair do campo depois de uma recusa não reenvia o
  mesmo texto; Enter pede de novo, Esc desiste.
- **Na fronteira:** o canal confere o id no alfabeto do núcleo (como D42: ele
  vira caminho no daemon), apara o nome e recusa, sem cortar, o que passa dos
  120 caracteres do `session.NameLimit`.
- Nada em repouso muda: as referências 02 e 03 medem 0,00%.

Descartados: mudar o título ao digitar e desfazer na recusa (o título mostraria
por um instante um nome que o daemon não guardou); um diálogo para o nome (tira
o olho do título que se está mudando); um item na lateral (ela não tem menu por
conversa, e criar um é outra decisão). Pedido ao núcleo, sem pressa: o stream
da lista anunciar o nome de uma conversa terminada, o que tornaria a releitura
desnecessária.

**D44. O turno ao vivo: rodadas, chamadas em voo e o que o turno custou.** A
janela já dobrava os `progress` de `rounds` e `in_flight` e não os mostrava, e
do `usage` do `turn.completed` guardava só o contexto: saída, entrada e cache
eram descartados.

- **Rodadas e chamadas em voo, nos termos da TUI** (`ceilingSegment` em
  `internal/tui/statusbar.go`): "rodada N/M" aparece a partir da metade do teto,
  em tom de aviso, e em tom de erro a partir de três quartos; "N/M em voo"
  aparece só quando todas as vagas estão ocupadas. Longe de um teto o número é
  um que ninguém usa, e uma fileira de números que ninguém usa é uma que se
  deixa de ler. Onde: na barra do composer, ao lado do medidor — a mesma barra
  é o rodapé da janela da grade (compacto) e o da conversa maximizada.
- **A contagem inteira, sempre, onde não pesa:** passar o mouse no botão de
  parar (e no indicador, quando aparece) diz "rodada 2 de 100 · 2 de 4 chamadas
  em voo", ou que a primeira rodada ainda não foi contada. No Crew, a aba Sessão
  ganha a linha "turno" enquanto um turno corre, com as cores dos tetos.
- **Tokens:** o `usage` é guardado inteiro, do turno que terminou por último, e
  zerado no `turn.started` — deixado de pé, seria lido como o do turno novo. A
  aba Sessão mostra saída, entrada (somada pelas rodadas: é o que o turno
  custou, não o quanto o contexto está cheio) e cache, quando houver; o medidor
  de contexto do composer diz o mesmo ao passar o mouse. Enquanto um turno
  corre, a aba diz que os tokens vêm ao fim dele; um turno que terminou sem
  `usage` diz que o provedor não informou, em tom de aviso — nunca o número do
  turno anterior.
- **Tokens durante o turno continuam fora (L5, D25):** o `usage` só viaja no
  `turn.completed`. Estimar a partir do texto que chega seria um número que o
  daemon nunca mandou.
- Nada em repouso muda: o indicador só existe com um turno perto de um teto, e
  na gravação o turno de 02 está em 2/100 e 2/4. As referências 02 e 03 medem
  0,00%.

Descartados: mostrar sempre "rodada N/M" (um número parado na tela, que muda a
referência 02 sem dizer nada que se use); tokens do turno anterior no composer
(o número ficaria ao lado de um turno que não é o dele); uma estimativa de
tokens ao vivo (L5).
