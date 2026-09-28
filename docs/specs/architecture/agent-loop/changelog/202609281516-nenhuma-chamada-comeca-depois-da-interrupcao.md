# Nenhuma chamada começa depois da interrupção

**Data:** 2026-09-28
**Specs afetadas:** `202608072335-agent-loop` (`.p` — seção 6 e invariantes; `.i` — Passo 7)
**Fonte:** o que o #392 deixou de fora de propósito — o `done` que o pump
estreita e não consegue impedir, e o laço que o tomava ao pé da letra

## O que mudou

Chamada que ainda não começou quando a interrupção fica visível **não
começa**: não vai ao usuário e não executa. O laço olha o contexto em três
instantes:

1. quando o stream termina, antes de o lote entrar em execução;
2. no início de cada chamada, o que cobre os grupos seguintes de um lote;
3. logo antes de `Execute`, depois da aprovação.

Toda chamada que não começou é respondida no histórico com resultado de erro —
`not run: the turn was interrupted before this call started` — e o turno
termina em `StopInterrupted`, sem erro.

## O defeito

Toda cancelação tem um último instante em que pode ser vista. Para o pump do
provider é a leitura do transporte: um frame tirado antes do cancelamento ainda
é decodificado, e se era o terminal o stream termina `done`. O #392 reduziu essa
janela ao menor tamanho que o pump consegue; fechá-la não está ao alcance dele.

O laço tomava o `done` ao pé da letra. Anexava a resposta e ia direto para
`execute`, e o contexto só era olhado de novo no topo da iteração seguinte,
depois de as chamadas terem rodado. `runOne` também não olhava, e `Write` e
`Edit` ignoram o contexto que recebem — de propósito, porque edição pela metade
é pior que edição lenta. Uma escrita aprovada automaticamente caía no disco
depois de o turno ter acabado.

Os testes que reproduziram isso acharam mais dois caminhos com a mesma forma:

- **Interrupção durante um grupo.** O que estava rodando termina, e o resultado
  fica (RN-5), mas os grupos seguintes rodavam. A chamada entre eles que
  precisava de aprovação era posta ao usuário; o aprovador da sessão responde
  contexto cancelado com negação, e negação chega ao modelo como recusa da
  pessoa — *"The user just refused this attempt … Do not retry it"*. A pessoa
  não recusou nada; ela parou o turno.
- **Aprovação concedida junto com a interrupção.** Concessão permanente e
  "permitir nesta sessão" respondem antes de o aprovador olhar o contexto, e o
  "permitir" de uma pessoa pode correr contra o próprio pedido de parada. A
  ferramenta executava.

Antes da correção, 600 de 600 execuções vermelhas: 3 testes × 200, `-race`, com
`-cpu=1`, 4 e 8.

## O que fica no histórico, e por quê

A resposta do modelo **fica**, e cada chamada dela é **respondida** como não
executada. Três alternativas foram descartadas:

- **Anexar a resposta com as chamadas sem resposta.** É conversa que o provider
  recusa no turno seguinte: Anthropic exige um `tool_result` para cada
  `tool_use`, OpenAI uma mensagem `tool` para cada `tool_call_id`. O histórico
  só cresce, então a chamada órfã ficaria nele, recusada de novo a cada turno.
  `Rebuild` já descarta chamada sem resposta pelo mesmo motivo.
- **Descartar a resposta inteira.** A pessoa viu o texto chegar, e viu cada
  chamada pedida (`tool.requested`). O modelo, não: um "não, faça de outro
  jeito" no turno seguinte falaria de algo que ele nunca disse.
- **Anexar só o texto.** O texto do modelo — *"Escrevendo os dois arquivos."* —
  fica prometendo o que não aconteceu, sem nada que diga o contrário. É
  histórico que mente sobre o disco, e a RN-5 põe isso abaixo de histórico
  incompleto.

Respondida é a única forma em que o histórico está bem formado e diz exatamente
o que aconteceu: o modelo pediu, e nada rodou.

## Por que três instantes, e não um

Um só — logo antes de `Execute` — já impede qualquer ferramenta de rodar. Os
outros dois existem pelo que ficaria errado sem eles, e cada instante tem o
teste que fica vermelho quando só ele é retirado:

- **Sem o do fim do stream**, o lote que ninguém executou passa pela máquina de
  lote: conta rodada, passa pelo detector de repetição e ganha o lembrete
  *"Those tools ran at the same time"* sobre ferramentas que não rodaram.
- **Sem o do início da chamada**, a chamada de grupo seguinte que precisa de
  aprovação é posta ao usuário e vira a recusa que ninguém fez.
- **Sem o de antes de `Execute`**, a concessão que chega junto com a interrupção
  executa a ferramenta.

## O que não mudou

- Ferramenta **já iniciada** termina, e o resultado é anexado se produziu efeito
  (RN-5). `write` e `edit` continuam ignorando o contexto.
- Interrupção durante uma aprovação que a pessoa está de fato respondendo
  continua na linha `blocked` da seção 6: o aprovador nega, e o resultado é
  anexado como negado.
- Nenhum `tool.completed` para a chamada que não rodou — como para toda resposta
  que não é da própria ferramenta: negada, não permitida, ferramenta
  desconhecida. A consequência também é a delas: a linha do cliente fica
  aberta, e `Rebuild` descarta a chamada numa sessão retomada.
- Nada aqui depende do #392. A regra fecha a janela pelo lado do laço, seja qual
  for o jeito como o stream terminou.

## Fica de fora

- **Interrupção que chega numa resposta final, sem chamada.** A checagem nova
  fica no caminho que tem chamadas. Sem chamada, o laço ainda segue para a
  definição de pronto com o contexto cancelado; critério que não consegue nem
  começar conta como `unavailable`, e o modelo ouve que o trabalho não pôde ser
  conferido. É outro defeito, para outra branch.
- **Negação causada pela interrupção lida como recusa da pessoa**, na linha
  `blocked`. Mudar isso é mudar a regra daquela linha, e é outra decisão.
