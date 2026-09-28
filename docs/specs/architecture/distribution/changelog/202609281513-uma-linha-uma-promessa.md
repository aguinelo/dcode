# Uma linha, uma promessa

**Data:** 2026-09-28
**Specs afetadas:** `202608072352-distribution` (`.p`, invariantes)

## O que mudou

A invariante do gerador de digests carregava duas promessas, cada uma com o seu
teste, e virou duas linhas:

| Linha | Teste |
|---|---|
| O gerador escreve o digest de toda plataforma publicada. | `TestTheGeneratorCarriesEveryPublishedDigest` |
| O gerador recusa release ao qual falte uma plataforma. | `TestTheGeneratorRefusesAReleaseMissingAPlatform` |

A segunda ganhou o sujeito e a palavra "plataforma" para ficar de pé sozinha;
fora isso o texto é o mesmo, dividido. Nenhuma promessa entra e nenhuma sai.

## Por quê

O `specguard` reivindicava cada linha de invariante pelo primeiro fragmento do
mapeamento que ela contivesse, percorrendo um `map` — e o Go sorteia a ordem de
um `map` a cada `range`. Esta linha continha dois fragmentos, então só um dos
dois testes era procurado em cada execução, e qual deles era sorteio: renomear
o outro deixava a guarda vermelha numa fração das execuções e verde no resto.

A guarda agora exige que **cada linha seja reivindicada por exatamente um
fragmento**, e reprova o fragmento que não reivindica linha nenhuma. Linha com
duas promessas vira duas linhas — o "um teste por linha" do `.i`, lido nos dois
sentidos. A decisão inteira, com a medição, está no changelog de mesmo nome do
`provider-adapter`, onde o defeito foi achado.
