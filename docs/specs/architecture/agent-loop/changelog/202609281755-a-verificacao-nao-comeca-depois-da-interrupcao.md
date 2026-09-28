# A verificação de pronto não começa depois da interrupção

**Data:** 2026-09-28
**Specs afetadas:** `202608072335-agent-loop` (`.p` — seção 6 e invariantes; `.i` — Passo 7)
**Fonte:** o que o #395 deixou de fora de propósito — a interrupção que chega
numa resposta final, sem chamada

## O que mudou

Resposta sem chamada que termina junto com a interrupção **encerra o turno**. O
laço olha o contexto quando o stream termina, antes da verificação de pronto, e
se a pessoa já parou o turno a verificação não começa: nenhum critério roda, e
nada sobre uma verificação que não aconteceu chega ao modelo nem ao selo. O
turno termina em `StopInterrupted`, sem erro, com a resposta inteira no
histórico e a nota que todo turno interrompido deixa sobre os arquivos que a
sessão escreveu.

## O defeito

O stream pode terminar `done` no instante em que a pessoa pede para parar: o
pump ainda decodifica um frame terminal que tirou antes do cancelamento, janela
que o #392 estreita e não consegue fechar. Sem chamada na resposta, `Run` ia
direto para `checkDone` com o contexto cancelado.

Nesse contexto, o executor de critério real não consegue começar nada.
`sandbox.Runner.Run` chama `cmd.Start()`, e `exec.Cmd.Start` responde contexto
já cancelado com o erro do contexto antes de existir processo. Não é erro de
saída, e `Check` o classifica, com razão, como critério que não pôde rodar:
`unavailable`. Com arquivo escrito no turno, `VerificationOf` dava
`unavailable`, e `checkDone` devolvia como reentrada o lembrete de trabalho não
conferido. O lembrete entrava no histórico, uma rodada era contada, e só então
o topo do laço via o contexto e encerrava em `StopInterrupted`. O selo em
`turn.completed` saía com os critérios como `unavailable`.

O modelo ouvia *"You changed files and there is no check here that could
confirm them"*, quando a verdade era que a pessoa tinha parado o turno.

O teste que reproduz isso usa um executor espião que responde como o real
responde a contexto cancelado. Com o teste que segura a decisão da seção "Sem
definição de pronto", 1.200 de 1.200 execuções vermelhas antes da correção:
2 testes × 200, `-race`, com `-cpu=1`, 4 e 8.

## Onde a checagem fica, e por quê

No começo do caminho sem chamada, logo antes de `checkDone`.

- **Não no topo da rodada seguinte**, que é onde a interrupção era vista: ele
  vem depois da verificação e da rodada gasta com ela.
- **Não dentro de `checkDone`, antes de `Check`.** Ali só pegaria o caso em que
  haveria algo para verificar, e o nome do mesmo instante passaria a depender
  da configuração — ver "Sem definição de pronto".
- **Não antes do desvio `if len(calls) == 0`**, embora o instante seja o mesmo
  para os dois caminhos. Resposta com chamada não pode terminar só retornando:
  chamada sem resposta é conversa que o provider recusa no turno seguinte —
  Anthropic exige um `tool_result` para cada `tool_use`, OpenAI uma mensagem
  `tool` para cada `tool_call_id` —, e `Assemble` manda o histórico como está.
  Responder as chamadas como não executadas é o que o #395 faz.

Quando os dois estiverem na `main`, as duas checagens fazem a mesma pergunta no
mesmo instante, e a forma certa é uma só, antes do desvio: responde como não
executada cada chamada que houver — nenhuma, no caminho sem chamada — e
encerra. Duas checagens para um instante são dois lugares para manter de
acordo. Quem entrar por último junta as duas.

## Por que `StopInterrupted`, se a resposta está completa

A resposta **fica**, inteira: a pessoa a viu chegar, e a interrupção veio depois
dela. O que a interrupção cortou foi o fim do turno, e com definição de pronto o
fim do turno é a verificação. As alternativas dizem algo falso ou desobedecem:

- **`StopDone`** afirma uma verificação que não aconteceu — exatamente a
  afirmação que a RN-9 existe para impedir.
- **`StopUnverified`** afirma que nenhuma verificação podia rodar, e podia; quem
  a impediu foi a pessoa.
- **Rodar a verificação com outro contexto** ignora o pedido de parar, e uma
  suíte de testes leva minutos. A RN-5 diz que todo ponto de espera é
  interrompível.

## Sem definição de pronto

Sem definição de pronto não havia o que verificar, e o turno encerra em
`StopInterrupted` do mesmo jeito. Antes encerrava em `StopDone`: `checkDone`
devolvia `StopDone` sem rodar nada, e o laço não olhava o contexto nesse
instante.

A escolha é deliberada. O laço viu a interrupção antes de o turno acabar, e o
que ele faria em seguida é configuração, não o que aconteceu. O mesmo instante
tem o mesmo nome qualquer que seja a configuração — e, quando o #395 entrar,
também com ou sem chamada na resposta. A seção 6 diz que interrupção termina em
`StopInterrupted`; a exceção ficaria por conta de uma configuração que a pessoa
que apertou a tecla não está vendo.

O que isso muda, sem definição de pronto:

- o motivo em `turn.completed` — no log de eventos, no protocolo e na linha
  final de `dcode "…"` — passa de `done` para `interrupted`;
- se a sessão escreveu algum arquivo e os lembretes estão ligados, o histórico
  ganha a nota que todo turno interrompido deixa: que o turno foi interrompido
  depois de aqueles arquivos mudarem, e que o modelo não deve supor nada sobre
  eles.

Nada mais. Nenhuma verificação rodava antes e nenhuma roda agora, nenhuma rodada
é gasta, e o relatório de pronto não muda. A TUI não dá entrada própria nem a
`done` nem a `interrupted`. `TestAnInterruptAsTheAnswerEndsReadsInterruptedWithNothingToCheck`
segura a decisão: pôr a checagem dentro de `checkDone`, só onde haveria
verificação, o deixa vermelho.

## O que não mudou

- Critério que roda com o contexto vivo continua classificado como sempre.
  `Check` e `VerificationOf` não mudaram.
- O selo de um turno interrompido é o que a última verificação de fato
  encontrou, com a defasagem que `WriteSeq` aplica, como em toda interrupção.
  Sessão que nunca verificou nada não tem selo.
- O caminho com chamadas não mudou aqui.

## Fica de fora

- **Interrupção durante a verificação.** A pessoa aperta a tecla enquanto
  `make test` roda — janela de minutos, não de microssegundos, e por isso a mais
  provável das duas. O critério cortado, e cada critério seguinte, que já não
  consegue começar, conta como `unavailable`: sem outro critério reprovado, o
  modelo ouve que o trabalho não pôde ser conferido, e o selo diz o mesmo.
  Corrigir pede que `Check` distinga
  critério interrompido de critério indisponível, e uma decisão sobre o que um
  relatório pela metade vale para o selo. É outra branch.
