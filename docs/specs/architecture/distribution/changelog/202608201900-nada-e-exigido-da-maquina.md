# Nada é exigido da máquina, e o aviso é sobre cobertura

**Data:** 2026-08-20
**Specs afetadas:** `202608072352-distribution` (`.p`)
**Substitui:** a regra de aviso de `202608201700` e a de tamanho de aviso do #226

## O que mudou

O instalador **para de falar do cosign quando ele não está lá**, e para de sugerir
instalá-lo em qualquer circunstância.

O que precisa de duas rotas é **release substituído**, e duas coisas independentes
o cobrem: o digest que o instalador carrega e a assinatura. Qualquer uma basta.
Então:

| Cobriu substituição | O que sai |
|---|---|
| o digest carregado | nada |
| a assinatura | nada |
| nenhum dos dois | o aviso, apontando o `install.sh` fixado deste release |

## Por que mudou

Porque a premissa estava errada, e ela foi corrigida de fora: **ninguém instala
pacote adicional para instalar um binário.** A pesquisa que originou o #223 já
tinha o dado e eu não tirei a conclusão inteira — dos seis instaladores
examinados (rustup, bun, deno, nvm, k3s, uv), **nenhum** exige ferramenta externa
de verificação. Quatro não verificam nada.

O #222 tirou o cosign do caminho crítico. O #226 encolheu o aviso. Nenhum dos
dois foi até o fim: o aviso continuava nomeando uma ferramenta que o usuário não
tem e não vai instalar, e continuava dizendo "instale o cosign e rode de novo" —
que é responder a um problema entregando um segundo problema.

E, depois do #223, dizer que a assinatura ficou por conferir enquanto o digest
carregado passou é **ruído vestido de diligência**: a verificação que importa
aconteceu, por uma rota que não depende do cosign.

## O que isso não afrouxa

"Nunca não-verificado em silêncio" continua de pé, e é justamente por isso que a
regra pôde encolher: instalação com digest carregado conferido **é** verificada.
O SHA-256 roda sempre. O único caso em que algo real fica descoberto é o único em
que o aviso aparece — e ali ele se repete na última linha, porque a rolagem de um
`curl | sh` enterra o começo.

O cosign continua sendo usado quando por acaso está no PATH, e assinatura que não
confere continua abortando. Ele deixou de ser assunto quando não está.

## Testes removidos, e por quê

Três testes afirmavam a regra anterior e não podiam ser afrouxados até passar,
então foram substituídos por outros que afirmam a nova:

- `TestAMissingCosignSaysTheSignatureWasNotVerified` (#222) exigia que o aviso
  nomeasse a ferramenta. É exatamente o que deixou de acontecer.
- `TestAnUnpinnedInstallerFallsBackWithoutComplaining` (#223) exigia silêncio sem
  pino. Estava certo enquanto o cosign carregava a garantia de substituição; ele
  não a carrega mais, então sem pino e sem assinatura há lacuna real a declarar.
- O par do #226 dimensionava o aviso pela ausência do cosign. A dimensão agora é
  se algo cobriu substituição.

Um quarto, `TestAnInstallerAskedForAnotherReleaseSaysItCannotCheckIt`, perdeu uma
asserção: o aviso não nomeia mais qual release ele carrega. A metade acionável é
a URL — saber que ele carrega 9.9.9 não diz a ninguém o que rodar.

## Ainda aberto

O `dcode update` continua exigindo cosign (`update.go:133`), pelo motivo que o
instalador deixou de ter: o binário não tem segunda rota para o digest esperado.
Tem uma disponível — ler o `main/install.sh`, que agora carrega os digests
commitados do último release. É o PR seguinte.
