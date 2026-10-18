# A orquestradora sabe o próprio papel

**Data:** 2026-10-18
**Specs afetadas:** `202608080016-behavior-definition` (`.p`, seções 2, 2.1, 3,
7 e 8); `202610132000-orchestration` (`.p`, seção 10, ponteiro)
**Fonte:** o primeiro teste ao vivo da orquestradora, num modelo de verdade.

> **Regra:** sessão de projeto orquestrado é de um papel, e o prompt diz qual.
> A orquestradora nunca escreve e nunca pede troca de modo: divide o trabalho
> por tema e chama `delegate`.

## O relato

MiniMax-M3, repositório de brinquedo, tarefa "três pacotes independentes,
divididos entre agentes". A orquestradora leu o repositório, disse que o
terminal era somente leitura e que portanto nem ela nem os filhos podiam
escrever, e pediu à pessoa que trocasse o modo do sandbox. O daemon recusou,
corretamente. `delegate` nunca foi chamado.

## A causa

Nada no prompt dizia o que é uma orquestradora. Ela recebia o prompt de uma
sessão comum em `read-only`, cujo bloco de fronteira termina mandando dizer o
que falta e "deixar a pessoa decidir se troca o modo". Era o único texto sobre a
situação dela, e ela o seguiu ao pé da letra. A descrição de `delegate` dizia
como despachar, mas não que despachar é **como** se escreve quando se é a
orquestradora.

Os cenários roteirizados não pegavam, e não tinham como: o modelo roteirizado já
sabe chamar `delegate`. O defeito mora exatamente na metade que só um modelo de
verdade responde.

## O que mudou

- `Prompt.Role`, fixo na criação, e uma seção `Your role` logo depois da
  fronteira, com um texto constante para orquestradora, filho e revisor. Sessão
  sem papel não ganha seção, e o prefixo dela é byte a byte o de antes — as três
  goldens existentes não mudaram; uma quarta registra o da orquestradora.
- Para orquestradora e revisor, o bloco de `read-only` troca a última frase:
  o somente leitura deles é desenho, não modo a trocar. O preâmbulo que torna o
  bloco autoridade fica, e a frase sobre ninguém ser perguntado também.
- O papel sai das opções que os modos do daemon montam (`roleOf`): revisor,
  quem conduz, filho, nesta ordem. Nenhum modo dá as ferramentas de um papel sem
  que o prompt diga qual é.
- Papel desconhecido faz `Build` falhar, como doutrina sem identidade.
- A descrição de `delegate` passa a se sustentar sozinha: que quem escreve são
  os filhos, que `owns` são caminhos da raiz que nenhum outro filho possui, um
  item por tema, e que o harness commita, revisa, mede o gate e pergunta à
  pessoa sobre o merge.

## Por que no prefixo, e não num lembrete

O papel não muda dentro de uma sessão, e é a regra contra a qual a fronteira é
lida. Lembrete chega uma vez no histórico e decai — a mesma razão pela qual a
fronteira foi para o prefixo (`202609031200`). O custo é zero para toda sessão
que não é orquestrada.

## O contrato

`orchestrator-splits-by-theme`, o primeiro dos sete que o `.r` da orquestração
propõe, entra agora com fixture, juiz e o limiar proposto lá, **≥ 85%**: um
`delegate` de pelo menos três itens com `owns` disjuntos pela leitura do produto
(`orchestrate.OwnsOverlap`), e nenhuma frase pedindo troca de modo. O mundo é o
da orquestradora — `read-only`, papel `orchestrator`, `write` e `edit` negados
pelo modo como no produto — e o projeto por trás de `delegate` é um substituto
que aceita todo lote e não roda filho: o que se julga é o despacho. Mora nesta
família porque o que ele mede é o prompt. Compila em todo `make check` e não foi
medido.

## O que fica de fora

Os outros seis contratos da orquestração, que continuam na segunda fatia. E o
texto do relato que inicia o turno da orquestradora, que já diz que é o harness.
