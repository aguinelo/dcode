# O cartão de perguntas na TUI

**Data:** 2026-10-18
**Specs afetadas:** `202608081250-client-tui` (`.p`, §2.2, §7 e §10),
`202610132000-orchestration` (`.p`, §5.5, "o que o cliente desenha")

O núcleo passou a juntar num cartão as perguntas do mesmo tipo que esperam juntas (`question.card`,
`POST /questions/answers`), e a `.p` da `orchestration` diz o que o cliente desenha: o cartão de
pergunta do Claude Code. Esta é a metade da TUI; a do desktop é a D67 dele.

## O que entrou

- **O cartão é uma entrada do fluxo** (`KindCard`), posta onde o `question.card` chegou pela
  primeira vez e trocada no lugar a cada versão nova. Uma versão mais velha que a guardada não
  volta. Inline, como a aprovação: o fluxo guarda o que foi perguntado e o que foi respondido, e
  um cartão fechado vira a lista das respostas.
- **Em texto**, um item por vez: o tipo e o `header` com "1 de N" à direita; o texto da pergunta;
  o `summary` e, num merge, a `evidence` com o commit curto; as opções numeradas, a selecionada
  marcada por caractere, a recomendada com "(Recomendado)" e a `description` embaixo; "Outro"
  quando `allow_other`; e a linha "Precisa da sua contribuição · N pendentes". O proposto abre com
  "Você pediu: …" e o porquê da orquestradora, e a nota proposta aparece embaixo das opções.
- **`^Q` põe o teclado no cartão** mais recente que espera, e rola até ele. Sem cartão, a
  sobreposição de antes. No cartão: `↑` `↓` movem a escolha, `1`–`9` marcam, `←` `→` passam os
  itens, `s` pula, Enter vai ao próximo e no último confirma; `a` aplica a recomendada a todos;
  com "Outro" marcado, as letras são as palavras. `esc` solta o teclado; o cartão fica.
- **A confirmação é um `POST` só**, com cada item visto e não pulado e a escolha dele; os pulados
  e os que a pessoa não chegou a ver vão em `skipped` e continuam esperando — nada é respondido sem
  ser visto. "Outro" sem palavras não sai, e isso é dito.
- **O que voltou é dito.** Os cartões da resposta entram no fluxo, as perguntas respondidas saem
  da lista, e a resposta recusada é dita com o código e a mensagem do daemon, embaixo do item e
  numa nota. Um lote recusado inteiro é dito, e o cartão continua com o teclado.

## Por que assim

- **Inline e não sobreposição.** A sobreposição de `^Q` some quando a pergunta é respondida, e o
  dono quer decidir na conversa, como no Claude Code. O fluxo já tinha o precedente da aprovação.
- **As palavras do cartão numa tabela própria** (`card_view.go`), e não em `lang.go`: o cartão se
  desenha de um arquivo só, e `lang.go` já passa muito de 500 linhas.
- **`/questions` não traz os cartões para a TUI.** O cliente lê o log da orquestradora inteiro ao
  abrir, e o `question.card` está nele; ler os cartões também da rota mudaria a interface
  `Orchestration` sem nada novo para mostrar.
