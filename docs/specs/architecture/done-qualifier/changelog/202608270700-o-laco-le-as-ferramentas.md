# O laço lê as ferramentas, não o contrário

**Data:** 2026-08-27
**Specs afetadas:** `202608261730-done-qualifier` — onze invariantes migram de
previstas para verificáveis. Implementa a etapa 3 por um desenho diferente do
que o `.p` previa, e este arquivo é o porquê.

## A ordem, e de quem ela é

> "o loop starta lendo, projeta, qualifica, entende se tem tudo e aí executa,
> o loop lê tools e não ao contrário"

Isso corrige o desenho anterior. O `.p` punha `done_propose` como ferramenta
que **o modelo decide** chamar — e uma ferramenta que decide quando qualificar é
o modelo escolhendo quando ser medido.

Agora quem decide é o laço. A ferramenta é o canal, e a sequência é dele.

## Modo planejamento, e o conflito que ele expôs

O turno que descobre o pronto roda em **plan**: `read-only` e sem aprovação.
Descobrir por que régua você vai ser medido é **ler**, e um agente que pudesse
escrever enquanto decide pode mover a coisa que está prestes a medi-lo.

A primeira versão da ferramenta declarava que escrevia o `done.toml`. Honesto
quanto à consequência, errado quanto ao ator — e `read-only` nega toda escrita,
sem exceção:

```go
if mode == ModeReadOnly {
    return Verdict{Decision: DecisionDeny, ...}
}
```

Rodando contra o modelo real, ele reportou exatamente isso:

> *"The plan-mode read-only restriction blocks `done_propose` here as well — it
> counts as a write. Stopping now as instructed."*

Duas saídas. Abrir uma exceção no `read-only`, ou tirar a escrita do turno. A
exceção foi recusada: uma garantia com uma exceção é a garantia que a próxima
pessoa amplia.

## O desenho que ficou

| quem | o quê | onde |
|---|---|---|
| modelo | lê e propõe, chamando a ferramenta | dentro do turno, `read-only` |
| ferramenta | valida e **grava em memória** | toca nada, `Declare` sem caminho |
| laço | pede o commit quando o turno termina | fora do turno |
| daemon | mede e escreve o arquivo | sob a fronteira do **trabalho** |

E medir fora do turno conserta um segundo defeito que a exceção teria deixado
passar: medir em `read-only` chamaria de **quebrado** um critério que o sandbox
recusou por falta de diretório de cache. Uma proposta que nasce com medição
falsa é pior que nenhuma.

## O que o modelo real produziu

Spec só com prosa — *"`reverse.sh abc` imprime `cba`"* — e o `done.toml` que
saiu:

```toml
# Spec criterion 1: `reverse.sh abc` deve imprimir `cba`.
# now: acceptance (exit 1)
[reverse abc]
command = "test \"$(bash reverse.sh abc)\" = cba"

# Spec criterion 3: sem argumento, sai com código diferente de zero.
# now: regression (exit 0) — proposed as "fail" and it did the opposite
[reverse no arg exits nonzero]
command = "bash reverse.sh; test $? -ne 0"
```

**O terceiro é a demonstração inteira.** O modelo disse que falharia e ele
passou — porque `bash reverse.sh` sem o script sai 127, e `test 127 -ne 0` é
verdadeiro. Está verde **pelo motivo errado**: não valida argumento nenhum,
apenas registra que o arquivo não existe.

Nenhuma revisão humana de uma lista de comandos pegaria isso de relance. A
discordância entre o que o proponente declarou e o que a medição achou pegou —
e está escrita no arquivo, na linha acima do critério, onde o revisor olha.

Era exatamente para isso que o `Expects` existia, e é a primeira vez que ele
pega alguma coisa.

## Por que o arquivo, e não uma tela

O `done.toml` é a superfície de revisão: diffável, sobrevive à sessão que o
produziu, e é o que o próximo `/loop` lê. Um número que só apareceu numa tela é
um número ao qual ninguém volta.

E é uma revisão **por spec, uma vez na vida**. Depois dela o arquivo existe e o
laço roda sozinho — que é a independência pedida, sem trocar a assinatura por
confiança.

## O que ainda não existe

O `/loop` ainda não decide sozinho qualificar quando a pasta não declara nada.
A fase existe, o transporte existe, e ligar as duas é o passo seguinte.
