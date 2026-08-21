# `progress`: um evento para "quão longe já foi"

**Data:** 2026-08-21
**Specs afetadas:** `202608072240-client-server-protocol` (`.p`), `202608081250-client-tui` (`.p`)
**Fonte:** `refs/design/HANDOFF.md` (v5, §3 e §5)

## O que mudou

Evento novo no protocolo:

```json
{"turn_id": "t1", "tool_call_id": "c1", "kind": "rounds", "done": 2, "total": 100}
```

O laço emite `rounds` onde o contador anda, e `in_flight` na fronteira de cada
grupo de chamadas. O cliente mostra os dois na seção TURNO do painel.

## Um evento, não dois

Ferramenta contando arquivos e turno contando rodadas são **a mesma pergunta
feita a sujeitos diferentes**. Acrescentar superfície versionada duas vezes para
um tipo de pergunta é como ela sai torta: a segunda sempre responde um pouco
diferente da primeira, e aí existem duas formas de dizer "quão longe".

`tool_call_id` vazio significa que é do turno. Um campo, não um segundo evento.

## `kind` é conjunto fechado, não palavra para imprimir

O idioma do daemon não é o de quem lê, e cliente que imprime o texto do payload
mostra o idioma errado para metade dos usuários. `kind` é chave; o cliente diz na
língua da interface.

E só o que **alguém de fato emite** está declarado. `kind` que nenhum código
escreve é promessa em superfície versionada que ninguém cumpre — o defeito que
este repositório encontra com mais frequência que qualquer outro.

Por isso `files`, `lines` e `tests` **não** estão aqui: eles vêm quando as
ferramentas emitirem. E `tests` provavelmente nunca vem: contar teste que passou
exige parsear a saída do `bash`, que é exatamente o que o comentário do
`ToolCompleted` proíbe com todas as letras.

## Entra na sequência, e essa foi a decisão difícil

A alternativa era deixá-lo **fora** do `Seq`, com o argumento de que ninguém
reproduz uma contagem.

Isso abriria buraco na única propriedade sobre a qual o registro é construído —
*"por sessão, monotônico a partir de 1, nunca reusado e nunca com lacuna"* — e
registro com buraco é registro cuja reprodução não é confiável sobre mais nada
também.

O `message.delta` já é conversador e já está lá, gravado com `Seq` e apenas sem
forçar flush. Progresso segue o mesmo caminho, em vez de inventar exceção para um
segundo tipo.

## Rodada é ciclo que continuou

`Iterations` conta as vezes em que o laço deu mais uma volta, então **turno que
respondeu numa passada não reporta rodada nenhuma**. Não há teto se aproximando,
e `0/100` na tela é número que significa que nada está acontecendo.

O teto viaja junto da contagem. Contagem sem o limite responde "quantas" quando a
pergunta é "quão perto", e cliente carregando o limite à parte carregaria cópia de
uma configuração que ele não vê mudar.

## Por que o painel abre com isso

Antes ele só abria com plano — e a maioria dos turnos não tem plano, então o teto
de rodadas ficava escondido justamente no painel que só abria quando outra coisa
já estava lá. É o item 1 do `docs/ROADMAP.md`, o único com evidência medida de
dano.

Os números sobrevivem ao turno que os produziu, então o painel abre no primeiro
turno e fica. Painel que aparece a cada turno e vai embora com ele seria
movimento onde a tela deveria estar parada.
