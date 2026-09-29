# DCode Desktop — brief para o Claude Design

> Para quem vai desenhar o app desktop do dcode. Escrito a partir do código em
> 28/09/2026, na `main` em `f08d742` (release 0.21.1). Onde este brief e as specs
> em `docs/specs/architecture/` divergirem, **a spec vence**.
>
> Anexe junto: a pasta `docs/brand/`, o `refs/design/HANDOFF.md` (TUI v5), o
> `refs/design/DCode TUI v5.dc.html` e prints do Claude Code desktop e do MiniMax
> Code desktop — as duas referências de interface que motivaram este app.

## 1. O que é o dcode

Um harness de codificação agêntica em Go: um daemon, um cliente e o laço do
agente entre os dois, num binário estático. É **agnóstico de LLM** — MiniMax-M3,
Claude, Gemini e qualquer endpoint compatível com OpenAI (inclusive modelos
locais), configurados como perfis nomeados.

O que o diferencia: o comportamento do agente é **medido**. São contratos com
limiar, rodados contra um modelo de verdade, e o placar é público — 60 declarados,
21 medidos.

Hoje o único cliente é a TUI, no terminal. **O desktop é um segundo cliente do
mesmo daemon**, como a TUI: o núcleo não muda, e a tela só mostra e pede o que o
daemon já oferece (seção 6).

Para quem: primeiro, o próprio autor, no dia a dia. Aberto a quem quiser usar.

## 2. Decisões já tomadas

- **Electron.** Chromium embutido, para o visual ser **idêntico em macOS, Linux e
  Windows**. Mac primeiro, Linux depois. Windows espera o núcleo, que ainda não
  tem sandbox lá.
- **Janela:** barra de título própria, mantendo os botões nativos (os semáforos
  no Mac; o overlay de controles no Windows).
- **Fontes embutidas no app**, nunca a do sistema — a escolha é parte deste design.
  Uma para interface e uma monoespaçada para código e saída de comando.
- **Várias sessões abertas ao mesmo tempo**, numa sidebar. Cada sessão tem o seu
  projeto, o seu modelo e o seu modo.
- **Trocar de modelo é continuar a sessão com outro modelo.** O daemon abre uma
  sessão nova carregando o histórico da anterior — não troca o modelo no lugar.
  A interface deve deixar essa continuidade visível (de onde veio, com qual
  modelo seguiu).
- **A nota do modelo nos contratos** aparece onde se escolhe o modelo. É a
  informação que só o dcode tem: o quanto aquele modelo cumpre o comportamento
  prometido.

**Propostas a validar no design:**
- Seguir o claro/escuro do sistema, com o âmbar só onde carrega significado.
- Primeira rodada só com o núcleo (seção 4); o resto numa segunda.

## 3. O que herdar

**Marca — imutável (`docs/brand/`).**

| Token | Hex | Papel |
|---|---|---|
| highlight | `#EFC066` | face de cima |
| body | `#E0A030` | face frontal — **a cor primária** |
| shadow | `#B87D1E` | base e aresta |
| eye | `#A8452A` | o olho do mascote, e nada mais |

O `⏺` é o olho do mascote **e** o marcador de toda linha de ferramenta
(`⏺ read`, `⏺ edit`, `⏺ bash`): cada execução na tela repete a marca. Logomark
(`logomark.svg`) para identificar; mascote (`mascot.svg`) para personalidade.

**Da TUI — o vocabulário (`refs/design/HANDOFF.md`, v5).** A tarefa é um card;
delegação é um card com os filhos dentro, e o filho que não respondeu aparece ali
com o motivo; a barra de atividade diz o que está acontecendo com um verbo; o selo
de conclusão fecha o turno; cada pergunta abre com uma régua; o pedido para cruzar
uma fronteira aparece **no fluxo**, na raia dele, e fica no lugar com a resposta.

**Princípios.**
- **Cor só onde carrega significado.** A TUI tem um tema só: texto por peso
  (normal, negrito, esmaecido, itálico) e estado por cor. Papéis: accent (âmbar),
  ok/adicionado (verde), erro/removido (vermelho), aviso (amarelo), perigo — **só**
  para acesso total —, esmaecido. Medidor de contexto: esmaecido até 74%, aviso de
  75% a 89%, erro a partir de 90%.
- **A conversa fica com o espaço.** Painéis laterais cedem antes dela.
- **Toda falha é explícita e visível.** Timeout, erro do modelo, daemon que caiu:
  a tela diz o que aconteceu e por quê. Nada que falhou pode parecer sucesso ou
  "só lento".

## 4. Telas da primeira rodada

1. **Janela principal:** sidebar de sessões, conversa, campo de entrada e barra de
   status (modelo, modo, branch, contexto).
2. **Nova sessão:** pasta do projeto, perfil de modelo, modo (plan, assist, auto).
3. **A conversa em andamento:** texto chegando, raciocínio do modelo (recolhível —
   não é resposta), ferramentas como cards com o resultado (linhas, arquivos,
   `+/−`, código de saída, duração) e o diff inline, plano, progresso.
4. **Aprovação:** o pedido para cruzar uma fronteira, com as decisões possíveis.
5. **Troca de modelo:** o seletor no cabeçalho da sessão, com a nota de cada
   modelo, e o marcador de continuidade depois da troca.

**Fora desta rodada:** diff em tela cheia, configuração de perfis de modelo,
`/loop` com definição de pronto, busca, atualização do app.

## 5. Catálogo de estados

- **Primeira execução:** nenhuma sessão; o daemon subindo.
- **Sessão:** ociosa, rodando, bloqueada (esperando aprovação), fechada.
- **Fim de turno**, pelo motivo: concluído, interrompido, teto de rodadas, chamada
  repetida em loop, teto de tokens, erro, não verificado, incompleto.
- **Contexto** subindo pelas faixas; conversa compactada (quantas mensagens
  viraram resumo, quantas ficaram).
- **Erro de sessão;** **daemon caiu** — a conexão some, e a tela diz por quê.
- **Sessão continuada:** o marcador "continua a sessão X, N turnos", inclusive
  quando foi para outro modelo.
- **Skill carregada** (uma orientação entrou no turno — sempre anunciada).
- **Modo trocado** (plan, assist, auto) e o sandbox que resulta dele; **acesso
  total é o único estado de perigo**.
- **Janela estreita:** a sidebar recolhe antes da conversa encolher.

## 6. O que o daemon entrega

Use só isto. Se o design precisar de algo fora desta lista, **marque como pedido
de protocolo** — foi o que faltou na TUI v5, cujo design pediu dados que ainda
não existiam.

### Sessão

Id; estado (`idle`, `running`, `blocked`, `closed`); pasta do projeto; modelo, com
família, transporte e endpoint; branch git; modo (`plan`, `assist`, `auto`) e o
sandbox que ele implica (`read-only`, `workspace-write` ou acesso total); tamanho
da janela de contexto, em tokens; quantos critérios de pronto a medem; quando foi
criada; o nome que a pessoa deu, se deu.

### Eventos, na ordem em que acontecem

| Evento | O que traz |
|---|---|
| `session.created` / `session.resumed` | a sessão; ao continuar, de qual sessão veio e quantos turnos |
| `turn.started` | o pedido da pessoa |
| `message.delta` | o texto da resposta chegando |
| `message.reasoning` | o raciocínio do modelo — mostrável, nunca é resposta |
| `tool.requested` | ferramenta e entrada, antes de rodar |
| `progress` | `rounds`: o turno contra o teto de rodadas · `in_flight`: chamadas simultâneas contra o teto · `arguments`: uma chamada ainda chegando, em bytes, **sem total** · `files`: uma varredura de arquivos, com total quando conhecido |
| `tool.approval_required` | ferramenta, comando, qual fronteira cruza, motivo e regra, quando expira |
| `tool.approval_resolved` | permitir uma vez · nesta sessão · neste projeto · sempre · negar |
| `tool.completed` | ok ou não, saída (e se foi truncada), linhas, arquivos, `+/−`, código de saída, duração, início e fim, diff |
| `plan.updated` | itens do plano: texto, status, o que bloqueia |
| `done.proposed` / `done.signed` | a definição de pronto: cada critério já rodado (nome, comando, saída esperada e obtida, resultado), arquivos protegidos |
| `turn.steered` | a pessoa corrigindo o turno em andamento |
| `turn.completed` | o motivo do fim; tokens (entrada, saída, cache lido e escrito, contexto); a verificação: critérios atendidos, não atendidos, indisponíveis, protegidos tocados |
| `context.band` | a faixa da janela e a fração usada |
| `session.compacted` | quantas mensagens foram resumidas e quantas mantidas |
| `skill.loaded` | qual skill entrou no turno e quando ela se aplica |
| `session.mode_changed` | modo anterior, modo novo, sandbox resultante |
| `session.renamed` · `session.error` | o nome novo · o erro, dito |

### Ações

Criar sessão (pasta, perfil de modelo, modo, continuar de outra sessão, spec de
pronto); listar, abrir, apagar e renomear sessões; enviar turno, com texto e
imagens; rodar um `!comando` fora do turno (pela mesma fronteira, entra no
histórico como algo que a pessoa fez); interromper; redirecionar o turno em
andamento; desfazer o último turno; responder uma aprovação; trocar o modo;
compactar; listar specs; assinar a definição de pronto.

### O que não existe

- **Sessões gravadas pelo protocolo.** A listagem traz as sessões vivas do daemon;
  as gravadas, o cliente lê do disco.
- **Trocar o modelo de uma sessão viva sem abrir outra.**
- **Sessão que sobrevive ao fechar o app.** É decisão do projeto: nada sobrevive a
  quem o criou. Reabrir é continuar a partir do registro, numa sessão nova.
- **Progresso de ferramenta que não é varredura de arquivos** — testes rodando,
  por exemplo, não têm contagem.
- **Custo em dinheiro por turno.** Há tokens, não preço.

## 7. O que entregar

- As telas da seção 4 e o catálogo da seção 5, em claro e escuro.
- **Tokens como variáveis CSS:** cor (os papéis da seção 3, nos dois temas),
  tipografia, espaçamento, raio, sombra. Desta vez o mock fica perto do código: o
  app é Electron, web por dentro.
- **Um `HANDOFF.md` no formato do `refs/design/HANDOFF.md` da TUI:** visão geral,
  layout, telas, interações e teclado, animações, estado, tokens, assets — e, no
  fim, **"o que precisa de spec antes de codar"**, listando cada pedido de
  protocolo.
- Nada de lógica. O comportamento mora nas specs do protocolo e do laço.
