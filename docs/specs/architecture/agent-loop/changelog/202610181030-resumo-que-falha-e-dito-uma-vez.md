# Resumo que falha é dito uma vez

**Data:** 2026-10-18
**Specs afetadas:** `202608072335-agent-loop` (`.p`)

> **Regra:** no gatilho automático, um resumo que falhou não é pedido de novo
> no mesmo turno. A falha é dita uma vez, e o aviso diz que o gatilho não tenta
> mais naquele turno.

## O que acontecia

Desde `202610161200-resumo-que-falha-nao-compacta`, resumo que falha não aplica
o plano e emite `session.error` com `compaction_failed`. Mas o gatilho
automático é conferido a cada iteração, e o histórico, que ficou inteiro,
continua acima do limiar: a iteração seguinte monta o mesmo plano, pede o mesmo
resumo e falha pela mesma causa. Uma causa que não passa — credencial revogada,
cota esgotada — custava uma chamada a mais ao provider e um `compaction_failed`
a mais por rodada, dizendo a mesma coisa.

## O que mudou

- O turno trava o gatilho automático na primeira falha de resumo e não o
  confere mais até o fim do turno. O turno seguinte volta a tentar: a causa
  pode ter passado.
- O aviso dessa primeira falha diz, além da causa, que a compactação
  automática não tenta de novo naquele turno. O silêncio das iterações
  seguintes é o que o aviso anunciou, não um silêncio.
- A compactação forçada pela recusa `context_size` não lê a trava: o espaço
  acabou de fato, e uma causa passageira pode ter passado. `/compact` também
  não, por ser a pessoa pedindo.

## Por que uma vez por turno e não um recuo por crescimento do contexto

Foram consideradas duas formas: tentar uma vez por turno, ou esperar o
contexto crescer uma margem antes de tentar de novo. A margem pede um número
novo, sem medida que o justifique, e uma causa permanente falharia de novo a
cada margem. A trava por turno é a mais simples que cumpre as duas metades:
nunca calado, porque a primeira falha é dita e diz o que vem depois; e sem
repetir, porque o que vem depois é não tentar. Quando o espaço acabar de fato,
a recusa do provider força a tentativa, e esse caminho já existia.

## O que não mudou

Nenhum limiar, nenhuma chave de configuração, nenhum evento novo: o código
continua `compaction_failed`, e a mensagem ganhou uma frase.
