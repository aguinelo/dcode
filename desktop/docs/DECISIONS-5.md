# Decisões do desktop — D62 em diante

A continuação de [`DECISIONS-4.md`](DECISIONS-4.md), que vai de D49 a D61, e
dos anteriores, a partir de [`DECISIONS.md`](DECISIONS.md), que tem a
introdução, D1 a D20, as decisões em aberto, as lacunas e os pedidos ao núcleo.
A numeração é uma só; este arquivo começa porque o DECISIONS-4 passaria de 500
linhas com D62.

**D62. A revisão mostra as propostas anteriores do turno, recolhidas embaixo da
escrita.** Decisão do dono, de 2026-10-16: construir o histórico que a
`done-qualifier` promete (§4) como defesa contra uma reproposta que troca um
critério por um trivial. O daemon guarda toda proposta do turno qualificador,
mede e escreve a última, e o `done.proposed` passa a levar `sequence`,
`proposed_at` e `earlier` (registro `202610161200`).

- **O card continua sendo o da proposta escrita** (D56). Quando o turno
  repropôs, o título ganha "· 3ª proposta do turno", e embaixo, antes dos
  caminhos protegidos da escrita, um `<details>` fechado: "2 propostas
  anteriores · o turno repropôs; só a última foi medida e escrita".
- **Aberto, cada anterior em ordem, da mais antiga:** "#1 · 3 critérios · 4m00s
  antes da última", e cada critério com o nome, o comando, o que o proponente
  disse dele ("disse que falharia" / "disse que passaria") e o porquê; os
  protegidos dela, se havia. Sem classe, sem exit, sem ✓/✗: as anteriores
  nunca rodaram, e desenhar uma medição que não aconteceu seria inventá-la.
- **O tempo é contado da última, não do relógio.** "4m00s antes da última" não
  depende do fuso da máquina e diz o que importa — quanto o turno pensou entre
  uma e outra.
- **Uma proposta só não muda nada:** sem sufixo no título e sem o `<details>`.
  O caso comum fica com a cara de antes.

As réguas: o card só aparece com um loop esperando revisão, que as telas 02 e 03
não têm; as duas ficam como estavam.

Descartados: as anteriores lado a lado com a escrita, aberto por padrão (o card
já é o mais alto da régua, e o olho deve cair primeiro no que vai ser usado);
um diff entre propostas (custa desenho próprio, e a lista em ordem já mostra a
troca); medir as anteriores para dar classe a elas (rodar critérios que ninguém
vai usar, só para enfeitar a lista).

**D63. A Crew é a tela principal e leva todo elemento novo; a grade fica como
alternativa** (emenda D32 e D33). Decisão do dono, de 2026-10-17, depois de ver
as capturas da régua estendida: as telas seguem o visual da Crew — a fonte, as
cores, a paleta e o arranjo de três painéis (Sessões com busca e grupos por
projeto à esquerda, a conversa no meio com o título, a linha de onde trabalha e
o chip do modelo, as abas Mudanças, Pronto e Sessão à direita) —, e cada
elemento feito desde D41 aparece nela.

- **O padrão é a Crew.** Sem escolha guardada, a janela abre na Crew; quem
  escolheu a grade continua nela, e o seletor da barra do topo alterna as duas
  como antes. A ordem do seletor fica "Grade · Crew": trocá-la mudaria a barra
  das telas 02 e 03, que a régua mede.
- **Onde cada elemento mora na Crew:**
  - a régua do loop, com as pastas e os botões de parar (D55), e o card da
    revisão com as propostas anteriores (D56, D62): no alto da conversa, abaixo
    do título, como na maximizada. A aba Pronto mantém a régua com as pastas
    abertas, sem um segundo card: diz que a revisão está no alto da conversa;
  - os critérios em "ciclo N · parado s de L" (D54): na mesma faixa, abaixo da
    régua, e de novo na aba Pronto; o "⟲ Ciclo N voltou" fica no fluxo;
  - o plano fixo como checklist (D57), os chips de atividade com "+N" (D58), os
    anexos no campo e na mensagem (D46, D59) e o card de aprovação com as quatro
    respostas (D39): já estavam no corpo da conversa que a Crew e a maximizada
    compartilham, e seguem com os tokens da Crew;
  - "↻ loop" no menu "Nova": escolher o projeto abre a folha "Começar loop",
    como na janela de Nova sessão da grade; o `/loop` no campo continua;
  - o ⌘K (D27) abre sobre a Crew, como sobre a grade;
  - o PR e o CI (D60, D61) continuam no alto da aba Mudanças.
- **A thread segue o loop.** A Crew tem uma conversa no meio, e um loop troca
  de sessão a cada pasta: a thread vai da sessão que o loop abriu para a que ele
  ocupa agora, e um loop começado nesta janela toma o meio quando a primeira
  sessão dele abre. Abrir a conversa de uma pasta pela régua a põe no meio, em
  vez de maximizá-la numa grade que não está desenhada.
- **O que a régua do daemon lê** fica igual nas duas telas: a linha da conversa
  com `data-session-id` e `data-state`, a janela da conversa com
  `data-pane-id`, o palco como `main`, e "Nova sessão" como nome do botão
  "Nova".
- **Só na grade:** várias conversas lado a lado e a maximizada com os pares no
  topo. A lateral da grade continua a continuar uma gravada ao clique (D21,
  D42); na Crew, clicar numa gravada a mostra, e escrever a continua (D33).

As réguas: as telas 02 e 03 são da grade maximizada e continuam abrindo nela
no modo fixture, a 0,00%. O `check:daemon` passa 12 de 13: "continues-recorded"
clica numa conversa gravada da lista e espera a continuação, o comportamento da
lateral da grade, e a Crew mostra o histórico; o cenário é caminho protegido e
fica como está até o dono decidir.

Descartados: a Crew continuar uma gravada ao clique só para o cenário passar
(trocaria a leitura do histórico por uma sessão que ninguém pediu, o que D42
recusou); trocar a ordem do seletor (moveria a régua da 02 e da 03); um segundo
card de revisão na aba Pronto (dois "Começar o trabalho" para a mesma
proposta); refazer a grade com os tokens da Crew (o pedido é a Crew como tela
principal, não a grade com outra cara).
