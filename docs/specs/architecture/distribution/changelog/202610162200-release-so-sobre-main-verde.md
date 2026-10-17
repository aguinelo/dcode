# Release só sobre main verde

**Data:** 2026-10-16
**Specs afetadas:** `202608072352-distribution` (`.p` §5, `.i` passo 2)

## O que mudou

O pipeline de release ganhou um passo 0, o job `main-green`, antes de qualquer
build. Ele exige que a tag esteja na `main` e que a CI da `main` tenha passado
exatamente no commit da tag. Se a CI ainda roda — o caso comum, com a tag
empurrada logo depois do merge —, o pipeline espera até 40 minutos. Se terminou
vermelha, ou nunca rodou, o release para e diz qual dos dois.

O passo 1 deixa de prometer o gate de cobertura, que o workflow nunca rodou: a
cobertura é a união da matriz, e essa união é a da CI da `main` que o passo 0
exige.

## Por quê

O release nunca repetiu o gate de cobertura, de propósito: ele mede a união dos
perfis de macOS e Linux, e rodá-lo por plataforma reprovava por geografia. A
justificativa escrita no workflow era "uma tag só existe sobre um commit da
main, e todo commit da main passou pelo job `coverage`".

Isso deixou de ser verdade quando as pull requests pararam de rodar CI — o
gate local do pre-push as cobre, e a CI passou a confirmar a `main` depois do
merge. A partir daí um commit da `main` pode ter CI vermelha, ou ainda em
curso, e uma tag empurrada sobre ele seria publicada assinada. A suposição
virou verificação.

A revisão das specs também achou a §5 e o `.i` prometendo o gate de cobertura
no release, que o workflow não roda; o texto agora diz o que acontece.
