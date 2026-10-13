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

**D45. Uma chamada de ferramenta do modelo abre no que ela fez.** O fluxo
mostrava de cada chamada só a linha de resumo; só a saída de um
comando digitado com `!` aparecia aberta (D40). O `tool.completed` sempre
trouxe a saída inteira e, de quem muda arquivo, o `diff` unificado da chamada
(`internal/tools/diff.go`, cortado em 400 linhas com uma marca dizendo isso),
e a janela não mostrava nenhum dos dois. A TUI mostra, com Tab.

- **Como abre:** a linha inteira é o botão — clique, ou foco e ↵/espaço. Os
  botões Copiar e Rodar no terminal que já estão na linha continuam sendo só
  deles. Fechada por padrão, e fechada ela desenha exatamente o que desenhava:
  só o ponteiro, o hover e o anel de foco dizem que abre. As referências 02 e
  03 medem 0,00%.
- **O que mostra:** de uma mudança, o diff, com + e − nas cores do painel do
  git (o mesmo desenho, `DiffPre`), e não a saída, que numa edição é o eco do
  diff para o modelo — a escolha da TUI (`toolBody`). De qualquer outra, a
  saída em monoespaçado, sem os códigos ANSI e com só o que um `\r` deixou na
  linha. Uma falha mostra o erro inteiro, ao lado de um diff se houver.
- **Cortes ditos:** saída de mais de 40 linhas abre nas 40 primeiras, com
  "mostrar tudo · N linhas"; erro nunca é cortado. Quando a ferramenta cortou
  o que devolveu (`truncated`), a janela diz. A marca de corte do daemon no
  diff vira aviso, não uma linha do arquivo.
- **Onde:** no fluxo vivo e no histórico de uma conversa terminada (D42), que
  dobram os mesmos eventos. Chamada rodando, sem nada a mostrar ou digitada
  pela pessoa não abre.

Descartados: abrir o diff sem pedir, como a prévia da TUI (mudaria o fluxo em
repouso, que as referências medem); ler o diff do git (é o da pasta desde o
último commit, não o da chamada); uma seta ao lado da linha (mudaria a linha
fechada).

**D46. Imagens vão com o turno** (emenda D23 e A5). O daemon aceita imagens num
turno desde `SubmitTurnRequest.images` — base64 com o tipo, por valor e não por
caminho, porque o daemon pode estar em outra máquina — e a TUI as manda
(`/image`, colar); a janela mandava só o texto, e o "+" do campo avisava que
ainda não existia. Agora:

- **Três jeitos de anexar:** o "+" abre o seletor do sistema (só imagens, várias
  de uma vez), colar com uma imagem na área de transferência anexa em vez de
  digitar, e soltar arquivos sobre o campo anexa — o campo ganha a borda
  tracejada do acento enquanto algo é arrastado sobre ele. Na janela da grade o
  campo não tem o "+" (não tinha antes), mas colar e soltar valem.
- **O processo principal lê e confere, o renderer nunca vê um caminho.** Um
  arquivo escolhido é lido lá, com o tamanho visto antes da leitura; uma imagem
  colada ou solta chega lá como bytes e passa pela mesma conferência. O tipo vem
  dos bytes, nunca do nome — como `session.ReadImage` decide no núcleo: PNG,
  JPEG, GIF e WebP, e nada mais. Até 10 MiB por imagem, o `MaxImageBytes` da
  TUI. Até 20 por turno: o daemon não limita a contagem, e o limite é do app,
  para que um turno grande demais para o provedor seja recusado enquanto ainda
  dá para tirar uma. No envio, o canal confere tudo de novo — contagem, tipo
  declarado contra os bytes, base64, tamanho — porque o que vem do renderer não
  é aceito pela palavra dele.
- **Miniaturas acima do campo**, 56 px, cada uma com seu ×, com o nome e o
  tamanho no título. São desenhadas de URLs `data:`, que o CSP estrito do
  renderer já permite (`img-src 'self' data:`); nada no CSP mudou. As imagens
  ficam por conversa, como o rascunho: trocar de conversa não perde nada, e
  continuar uma conversa terminada leva as imagens junto com o texto.
- **Recusa visível duas vezes:** um aviso, e uma linha no tom de erro abaixo das
  miniaturas, que fica até o próximo anexo ou envio. Um arquivo recusado não
  impede os outros do mesmo lote; o que passa de 20 é recusado pelo nome, nunca
  cortado calado. Uma recusa do daemon ao turno vem com o motivo dele, e as
  imagens continuam anexadas.
- **Correção leva só texto** (emenda D23). `SteerRequest` tem só `text`, então
  uma mensagem com imagens durante um turno não vira correção: a janela diz por
  quê, guarda texto e imagens, e a pessoa manda quando o turno terminar. O mesmo
  vale quando a janela achava a sessão parada e o daemon responde que um turno
  roda: com imagens, nada é redirecionado. D23 dizia que imagem entraria numa
  fila; descartado pelo mesmo motivo que a fila de texto foi: mandaria sozinha,
  depois, uma mensagem que a pessoa pode já não querer. A fila de D47 guarda
  só texto, então ⌥↵ com imagens anexadas é recusado do mesmo jeito, com o
  porquê, e nada sai do campo; uma mensagem da fila nunca leva as imagens que
  estão no campo.
- **No fluxo, a mensagem mostra as imagens que foram com ela**, acima do texto,
  até 160 × 120. Os eventos não carregam imagens — `turn.started` traz só o
  texto —, então o que se vê é o que esta janela mandou: guardado contra o seq
  em que a sessão estava no envio, e mostrado na primeira mensagem (não
  correção) depois dele. Uma conversa relida do registro (D42), ou aberta em
  outra janela, mostra só o texto.
- **Modelo que não lê imagens:** o daemon não recusa no envio — a TUI decide
  isso do lado dela (`acceptsImages`), e a lista de modelos não diz quais leem.
  O erro do provedor chega como o erro do turno, com o motivo, no fluxo.
- Nada em repouso muda: as referências 02 e 03 medem 0,00%.

Descartados: o processo principal guardar as imagens e o renderer só um
identificador (estado no processo principal que a janela teria de limpar, e um
identificador velho depois de um reload); `blob:` para as miniaturas (afrouxaria
o CSP); ler o caminho de um arquivo solto pelo preload (o renderer teria
caminhos). Pedidos ao núcleo, sem pressa: `turn.started` levar as imagens (ou um
resumo delas) para que o histórico as mostre; o daemon recusar no envio, com
motivo, imagens para um modelo que não as lê; e `GET /v1/models` dizer quais
modelos leem imagens.

**D47. O campo lembra o que se pediu e guarda o que vem depois, como na TUI.**
Na TUI, ↑/↓ trazem de volta o que foi enviado e uma mensagem pode esperar o
turno acabar; no desktop, ↑ só andava entre janelas e toda mensagem durante o
turno redirecionava (D23).

- **↑/↓ no campo** percorrem o que a conversa já pediu, da mais nova para a
  mais velha, e passar da mais nova devolve o rascunho que estava no campo —
  como `HistoryPrev`/`HistoryNext` da TUI. ↑ só lembra na primeira linha, sem
  seleção, e com o campo vazio, o cursor no começo ou o passeio já começado:
  num rascunho, o primeiro ↑ vai ao começo, como num campo qualquer. ↓ só
  avança passeando, e na última linha. Assim uma mensagem de várias linhas
  continua editada como texto. As setas da grade não mudam: a janela já não
  lê setas digitadas num campo (D31). Com o menu "/" aberto (D48), as setas
  são dele: ele trata a tecla primeiro e o histórico ignora a que já foi
  tratada.
- **De onde vem:** dos eventos da própria conversa, `turn.started` e
  `turn.steered`, que o daemon devolve ao seguir uma sessão — sobrevive a
  reiniciar o app sem um segundo armazenamento. Uma conversa continuada
  (`session.resumed`) lê, na primeira vez que ↑ passa da mais velha, o
  registro da conversa de origem (D42), até oito continuações para trás; uma
  falha vira aviso com o motivo, e o próximo ↑ tenta de novo. Repetição
  seguida conta uma vez. Comandos (`!`, `/`) não entram: não são eventos de
  conversa, e lembrá-los pediria um armazenamento à parte.
- **A fila:** durante o turno, ↵ continua redirecionando (D23); **⌥↵** põe a
  mensagem na fila. Escolhido ⌥ porque ⇧↵ quebra linha e ⌘↵ maximiza (D31).
  A fila aparece em fichas acima do campo, na ordem, cada uma com ×; **⌃X**
  (sem seleção no campo) tira a última posta — a que a pessoa acabou de
  decidir, ao contrário da TUI, que tira a mais velha. Quando a conversa volta
  a ficar parada, a mais velha é enviada como mensagem nova; a seguinte espera
  esse turno começar e acabar, para nunca cruzar num redirecionamento. Só
  mensagens entram: um comando vale quando é dado, e esperá-lo o rodaria
  noutro estado da sessão; a recusa é dita e o texto fica no campo.
- **Falha:** um envio recusado para a fila, que diz "A fila parou: …" com o
  motivo, mantém a mensagem e espera "Retomar". Nunca um envio calado.
- A fila vive enquanto a janela vive: fechar o app a perde, e isso está dito
  aqui. Nada em repouso muda: as referências 02 e 03 medem 0,00%.

Descartados: ↑ lembrar com qualquer texto no campo (tiraria a edição de várias
linhas); guardar o histórico no `localStorage` (uma segunda cópia do que o
registro já tem, que diverge); ⌥↵ redirecionar e ↵ enfileirar (inverteria D23).

**D48. Os comandos do campo num menu, e uma ajuda com eles e com as teclas.**
O campo já atendia `/undo`, `/compact`, `/mode` e `!` (D40), e nada dizia que
existiam: quem não lia o changelog só descobria errando um.

- **O menu:** `/` no começo do campo abre a lista dos comandos, filtrada pelo
  prefixo enquanto se digita, como o `Complete` da TUI
  (`internal/tui/commands.go`): só enquanto o campo é uma barra e um nome; com o
  primeiro espaço a escolha está feita e o menu sai do caminho. ↑ ↓ escolhem, ⇥
  completa, ↵ completa e envia o que não pede mais nada (`/undo`, `/plan`), e
  só completa o que pede argumento (`/mode `, `!`). Esc fecha o menu para o que
  está escrito; um segundo Esc sai do campo, como antes. Clicar num item é o ↵
  dele. Enquanto o menu está aberto as setas são dele: ele previne o padrão da
  tecla, e o histórico do campo (D47) deixa em paz uma tecla prevenida.
- **A ajuda:** `/help`, ou `?` no campo vazio, abre por cima da janela os
  comandos e as teclas — as da janela (as de `useWindowKeys`: ⌘K, ⌘N, ⌘0, ⌘1–9,
  ⌘↵, ⇧⇥, as setas, esc), as do campo e as da aprovação. As teclas ficam numa
  lista só (`src/state/help.ts`), com teste dizendo quais têm de estar nela.
  Esc, um clique fora ou "Fechar" fecham, e o foco volta para onde estava.
- **`/init`** manda o turno que a TUI manda, palavra por palavra: o teste lê o
  `InitPrompt` de `internal/tui/commands.go` e compara, então os dois não se
  separam calados. **`/plan`** sozinho mostra o plano atual — o último
  `plan.updated` da sessão — com quanto já foi feito; `/plan <texto>` manda o
  `ReplanPrompt` da TUI com o texto. Os dois começam um turno, e no meio de um a
  janela recusa dizendo por quê, em vez de redirecionar o turno com outro
  pedido (a TUI enfileira; o desktop ainda não tem fila).
- No campo de uma conversa terminada não há menu (comandos valem numa viva),
  mas `/help` digitado ali abre a ajuda. Nada em repouso muda: as referências 02
  e 03 medem 0,00%.

Descartados: abrir o menu com `/` em qualquer ponto do texto (um caminho como
`a/b` abriria o menu no meio de uma frase); `?` com texto no campo (é pontuação);
a lista de teclas tirada do código de `useWindowKeys` (as teclas são tratadas
num `switch`, não numa tabela, e reescrevê-lo como tabela para a ajuda era
mudança maior que a ajuda).
