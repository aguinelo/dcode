# Uma linha, uma promessa

**Data:** 2026-09-28
**Specs afetadas:** `202608072336-sandbox-policy` (`.p`, invariantes)

## O que mudou

Duas invariantes carregavam duas promessas cada, com um teste para cada
promessa, e viraram quatro linhas:

| Linha | Teste |
|---|---|
| `operation not permitted` vindo do sandbox **diz como se abre**: nomeia `/mode auto` e `sandbox.writable`, e diz que pergunta nenhuma vem. | `TestAWallSaysHowItOpens` |
| Só EPERM, e nunca sob `full-access`, recebe a nota de como se abre. | `TestAnOrdinaryFailureGetsNoNote` |
| Nada nomeado esconde os cofres de credencial mesmo assim. | `TestUnreadableDefaultsToHidingCredentialStores` |
| O home inteiro nunca é um nome válido. | `TestUnreadableExpandsHomeAndRefusesIt` |

A segunda ganhou "recebe a nota de como se abre" para dizer sozinha de que fala.
A quarta começa em maiúscula, e o fragmento dela no mapeamento perdeu o "o"
inicial para continuar casando. Fora isso o texto é o mesmo, dividido: nenhuma
promessa entra e nenhuma sai.

## Por quê

O `specguard` reivindicava cada linha de invariante pelo primeiro fragmento do
mapeamento que ela contivesse, percorrendo um `map` — e o Go sorteia a ordem de
um `map` a cada `range`. Cada uma destas linhas continha dois fragmentos, então
só um dos dois testes era procurado em cada execução, e qual deles era sorteio:
renomear o outro deixava a guarda vermelha numa fração das execuções e verde no
resto.

A guarda agora exige que **cada linha seja reivindicada por exatamente um
fragmento**, e reprova o fragmento que não reivindica linha nenhuma. Linha com
duas promessas vira duas linhas — o "um teste por linha" do `.i`, lido nos dois
sentidos. A decisão inteira, com a medição, está no changelog de mesmo nome do
`provider-adapter`, onde o defeito foi achado.
