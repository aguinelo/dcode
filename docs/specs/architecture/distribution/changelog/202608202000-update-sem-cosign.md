# O `dcode update` também não exige cosign

**Data:** 2026-08-20
**Specs afetadas:** `202608072352-distribution` (`.p`, `.config`)
**Continua:** `202608201900-nada-e-exigido-da-maquina.md`

## O que mudou

O `update` passa a ler os digests do **instalador publicado na `main`** — o mesmo
arquivo que o script de instalação carrega dentro de si — e aplica a regra do
instalador: substituição coberta pelo digest carregado **ou** pela assinatura,
qualquer uma basta.

`DCODE_UPDATE_INSTALLER_URL` sobrescreve de onde ele lê.

## Por que mudou

Era o último lugar que exigia um pacote, e a exigência era **pior** aqui que na
instalação: esta máquina já tem um dcode funcionando. Pedir que se instale outra
coisa para poder atualizar transforma a atualização num beco para todo mundo que
não tem cosign — que é praticamente todo mundo.

A rota já existia e não estava sendo usada. Desde o #223 o release commita os
digests dos artefatos no `install.sh` da `main`, onde substituir um asset não
deixa rastro público e mudar uma linha versionada é um commit. O binário sabe
buscar HTTP; era só ler o mesmo arquivo.

## A diferença deliberada em relação ao instalador

Onde o script de instalação **avisa**, o `update` **recusa**.

Não é inconsistência, é a assimetria da situação. Numa primeira instalação a
alternativa a instalar é não ter nada; aqui a alternativa é continuar com o
binário que já funciona. Parar custa uma versão e preserva tudo, então parar é
barato e é o certo.

O que **não** é assimétrico: assinatura que **falha** aborta o update, qualquer
que seja o digest carregado. Tornar uma verificação opcional não pode torná-la
decorativa, e as duas coisas — "não pôde ser conferida" e "não confere" — passam
a ser distinguíveis no código porque agora custam coisas diferentes.

## `ErrNoVerifier` deixou de ser um veredito

Ele dizia *"Install cosign, or download the artifact and verify it by hand —
dcode will not install something it could not check"*. Essa frase **era** a
exigência: o sentinela decidia o resultado, em vez de relatar um fato.

Agora ele diz que a assinatura não pôde ser conferida aqui, e quem decide é o
`Apply`, depois de saber sobre a outra rota.

## Toda falha da segunda rota é vazia, nunca erro

Host inalcançável, instalador fixado noutro release, bloco ilegível — nada disso
é um veredito sobre o artefato. Significa que a rota está indisponível, e o
`Apply` decide o que isso custa quando já sabe da outra. Devolver erro ali faria
uma falha de rede parecer adulteração.

## Só o que está entre os marcadores conta

`PinnedDigest` lê apenas entre `# BEGIN PINNED` e `# END PINNED`. Ler digest de
qualquer lugar do arquivo deixaria uma linha não relacionada — um comentário, um
exemplo, uma mensagem de erro — decidir o que se instala.

Dentro do bloco, ele procura um token de 64 hex em vez de interpretar a sintaxe
do `case`: a formatação em volta é assunto do gerador e pode mudar, o formato de
um digest não.
