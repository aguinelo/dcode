# O teclado do v5, decidido por convenção

**Data:** 2026-08-20
**Specs afetadas:** `202608081250-client-tui` (`.p`, seção 7 — **ainda não**)
**Fonte:** `refs/design/HANDOFF.md` (v5)

## O que mudou

Nada no código, e **nada na tabela da seção 7**. Isto fixa quais teclas o design
v5 vai usar, e por quê, antes de qualquer uma existir.

A tabela do `.p` só ganha uma linha quando a fase que a implementa entrar. Tecla
declarada em spec que nenhum código executa é **algo declarado que um lado lê e
nenhum lado escreve** — a forma que este repositório encontra sem parar, e que o
changelog da cópia dona do teclado documenta na própria seção "por que passou".

| Tecla | Ação | Fase |
|---|---|---|
| `^B` | dobra/expande a coluna lateral | 3 |
| `^R` | foca a trilha de sessões | 4 |
| `r` | renomeia a sessão sob o cursor, com a trilha dona do teclado | 4 |
| `F2` | idem, apelido | 4 |

## Por que estas

O handoff propõe `^E`, `^R`, `^N`, `^B` e `^Z`. Três colidem. A decisão foi
tomada **por convenção com outros programas**, não copiada do handoff.

**`^B` — a coluna lateral.** É o que "sidebar" significa hoje: VS Code alterna a
barra lateral com `⌘B`/`^B`, e a geração que aprendeu editor nele leva a tecla
junto. Está livre no dcode. O handoff já queria `^B` para "dobrar a coluna
inteira", que é a mesma coisa dita de outro jeito.

Um risco a nomear: `^B` é o prefixo default do tmux, que o intercepta antes de
chegar aqui. Quem usa tmux já convive com isso em todo programa que usa `^B`, e
já remapeou ou não. Não é motivo para escolher pior.

**`^R` — a trilha de sessões.** No readline, no bash e no zsh, `^R` é
*reverse-i-search*: procurar no histórico. Sessão **é** histórico. Este é o caso
raro em que tomar o acorde emprestado **reforça** o sentido em vez de brigar com
ele — a pessoa que aperta `^R` por reflexo quer exatamente o que a coluna faz.

**`r` e `F2` — renomear.** Duas teclas para um ato, pelo mesmo motivo que a
seção 7 já aceita três para quebrar linha: mental models diferentes, e uma que
pode não chegar.

`F2` é a ligação de "renomear" mais consistente entre programas — Explorer,
VS Code, IntelliJ, gerenciadores de arquivo. Mas F-key atravessa terminal, tmux e
ssh com menos garantia que letra, e a spec já escolheu tolerância a isso antes.

`r` é a convenção dos gerenciadores de arquivo **de terminal** — ranger, lf, nnn,
vifm —, que é o que a trilha é. E é legal: ver a seção seguinte.

## A RN-16 é mais estreita do que a frase curta sugere

O handoff resume como *"letra nunca é atalho"*. A regra escrita é **"numa linha
em que se digita, atalho é tecla de controle ou pontuação"**, e o código é ainda
mais preciso que a spec: o gate de `v` (modo de cópia) não é linha vazia, é
**foco no fluxo** — `p.model.Cursor >= 0`.

O comentário em `program.go:521` diz por que a distinção importa: *"linha de
entrada vazia é onde toda mensagem começa, então ligar um modo a uma letra ali
comia a primeira letra de qualquer coisa começando com v — 'voce' chegava como
'oce'"*.

Então letra é atalho legítimo onde não se digita: no modo de cópia (`k`, `j`,
`y`, `q`), no modal de aprovação (`[d] [a] [P] [G]`), no picker (`j`). A trilha
em modo navegando é a mesma situação — ela é **dona do teclado**, como a cópia —,
e `r` ali não tira letra de ninguém.

Nada na spec precisou de correção: `202608081250-client-tui.p.spec.md:300` e o
changelog `202608082330` já carregam a ressalva inteira. Quem circula com a frase
curta é o handoff, e a divergência está anotada em `refs/design/README.md` em vez
de editada dentro dele.

## As rejeitadas, com o motivo

**`^E` para ARQUIVOS** — é fim de linha (`program.go:494`), e é readline. O
handoff já tinha visto e sugerido trocar.

**`^F` para ARQUIVOS**, a sugestão do handoff — readline usa `^F` para avançar um
caractere, e em quase todo programa gráfico `^F` é *find*. Batizar uma árvore com
a tecla de busca ensina a coisa errada na primeira tentativa.

**Tecla nenhuma para a seção ARQUIVOS.** Editor nenhum dá acorde global a cada
seção da barra lateral: alterna a barra, e navega dentro dela. Com `^B` abrindo a
coluna, as duas seções aparecem juntas como no mock. **Isto apaga uma tecla em vez
de acrescentar uma**, e uma superfície `stable` que cresce menos é uma superfície
que envelhece melhor.

**`^N` para nomear** — readline usa como "próximo do histórico", e o
`picker.go:173` já usa como "descer", ao lado de `j` e `↓`. Seria a mesma tecla
com dois sentidos dentro do mesmo produto, e o sentido errado é o que o usuário
já treinou.

**`^Z` para desfazer o turno** — duas razões, e a segunda basta sozinha.

`^Z` é SIGTSTP em todo terminal que existe; o `⌘Z` gráfico não se transfere para
cá. Reatribuir a tecla de suspender é hostil com quem espera suspender.

E é desnecessário: **`/undo` já existe**, deliberadamente. `program.go:695`:
*"não é ferramenta. O modelo não desfaz o próprio trabalho, porque o julgamento
para o qual o undo existe é da pessoa."* Acorde novo para um ato raro e
consequente que já tem comando é superfície a mais sem ganho nenhum.

O handoff também anota que `^Z` "alcança a delegação inteira". Isso **já é
verdade** pelo `/undo`: `delegate.go:171` adota o estado do filho no pai
exatamente para que o undo do pai não pare na borda do que ele delegou.

## O que isto não decide

A tabela da seção 7 continua como está. Cada tecla entra nela no PR que a
implementa, com o teste que a cobra — que é também quando ela vira contrato de
uma superfície `stable`.
