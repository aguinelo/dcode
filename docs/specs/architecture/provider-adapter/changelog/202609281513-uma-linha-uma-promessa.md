# Uma linha, uma promessa

**Data:** 2026-09-28
**Specs afetadas:** `202608072334-provider-adapter` (`.p`, invariantes)

## O que mudou

A invariante do frame que traz uso e conteúdo juntos carregava duas promessas,
cada uma com o seu teste, e virou duas linhas:

| Linha | Teste |
|---|---|
| Frame que traz **uso e conteúdo juntos** entrega o conteúdo: o uso é lido depois das escolhas, nunca antes. | `TestAFrameCarryingUsageStillYieldsItsToolCall` |
| Uso em frame separado continua terminando sem reemitir a chamada. | `TestUsageOnItsOwnFrameTerminatesWithoutRepeatingTheCall` |

O texto é o mesmo, dividido: nenhuma promessa entra e nenhuma sai.

## Por quê

Foi nesta linha que o defeito do `specguard` apareceu. A guarda reivindicava
cada linha de invariante pelo primeiro fragmento do mapeamento que ela
contivesse, percorrendo um `map` — e o Go sorteia a ordem de um `map` a cada
`range`. Esta linha continha os dois fragmentos, então era reivindicada por
qualquer um dos dois, e só o teste daquele era procurado naquela execução.

Renomear um dos testes deixava `TestEveryInvariantHasATest` vermelho numa fração
das execuções e verde no resto. Medido: renomear
`TestUsageOnItsOwnFrameTerminatesWithoutRepeatingTheCall` reprovou 49 execuções
de 100; renomear `TestAFrameCarryingUsageStillYieldsItsToolCall`, 20 de 100. As
duas frações não somam cem porque a semente do `map` muda de processo para
processo: nem a chance de a guarda ver o erro era fixa.

A guarda agora exige que **cada linha seja reivindicada por exatamente um
fragmento**, e reprova o fragmento que não reivindica linha nenhuma. Linha com
duas promessas vira duas linhas — o "um teste por linha" do `.i`, lido nos dois
sentidos.
