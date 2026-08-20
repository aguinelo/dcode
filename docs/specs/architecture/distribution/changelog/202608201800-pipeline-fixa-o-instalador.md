# O pipeline fixa o instalador e o leva para a `main`

**Data:** 2026-08-20
**Specs afetadas:** `202608072352-distribution` (`.p`)
**Continua:** `202608201700-digest-fixado-no-instalador.md`

## O que mudou

O release passa a executar, nesta ordem: verifica a assinatura que acabou de
produzir → **fixa o `install.sh`** com `scripts/installer.sh` → publica o release
com o instalador fixado entre os artefatos → **leva o `install.sh` fixado para a
`main`** com `scripts/publish-installer.sh`.

O `scripts/version.sh` passa a ignorar o commit que o passo final escreve.

## Por que a ordem é o conteúdo

Cada uma das três relações existe por um motivo distinto, e nenhuma é estética.

**Fixar depois de verificar.** Digest tirado de um `checksums.txt` que ninguém
atestou é precisamente a falha que a fixação existe para impedir — reproduzida
dentro do pipeline que a implementa.

**Publicar depois de fixar.** Senão o asset publicado é o instalador sem pino, e
o release inteiro carrega uma promessa que o arquivo nele não cumpre.

**Escrever na `main` depois de publicar.** Daí em diante toda falha é
irrecuperável no sentido que importa: o release já é público. Reprovar o job não
despublica nada — pinta de vermelho um release que deu certo, que é o relatório
sobre o qual ninguém consegue agir. É a mesma razão que o `publish-tap.sh` já
carrega, e por isso o novo script tem a mesma forma: toda condição recuperável
sai com zero e avisa alto, e a única irrecuperável é o arquivo fixado não
existir — pois aí a `main` ficaria com os digests do release **anterior** enquanto
o release reporta sucesso, e todo install cairia no `checksums.txt` em silêncio.
Que é o comportamento correto de um instalador sem pino, e portanto invisível.

A ordem é asserida lendo o `release.yml` como dado. Nada em `make check` executa
o workflow, então um passo que deixa de fazer o que diz deixa em silêncio — foi
o que obrigou a existir o teste do carimbo de build.

## Por que a `main`, e não só o asset

Porque a URL que o README publica é `main/install.sh`. Um pino que fica só no
release nunca alcança quem instala.

E é ali que mora o argumento de segurança inteiro. Asset de release se substitui
sem rastro público; linha de arquivo versionado não, porque mudá-la é um commit
— visível no log, no diff, e atribuído.

## O acoplamento que isso criou, e que teve de ser resolvido junto

O commit de pino cai na `main` **depois** da tag, porque os digests só existem
depois dos artefatos construídos. Sem mais nada, toda consulta de versão
pós-release responderia "há commits desde a tag" com nada humano tendo mudado, e
o `version.sh` subiria PATCH sozinho.

É a forma que este repositório não para de encontrar: **automação deixando um
rastro que outro mecanismo lê como sinal.** O `version.sh` passa a ignorar o
assunto exato que o pipeline escreve — `chore(release): pin the installer to
vX.Y.Z` —, nunca o prefixo. Isentar `chore(release):` inteiro daria a qualquer
pessoa uma forma de não ser contada, e há teste para os dois lados disso.

O `scripts/version.sh` não tinha teste nenhum até aqui.

## Alternativa descartada

**Tag apontando para o commit de pino**, para a `main` e a tag não divergirem.
Exigiria construir antes de taguear, e o release é disparado **pela** tag. Sair
disso significa release por `workflow_dispatch`, que é exatamente o que o
comentário no topo do `release.yml` recusa: *"um release que precisa que alguém
se lembre de algo é um release que um dia sai sem assinatura"*.

A consequência aceita é que a tag `vX.Y.Z` aponta para um commit cujo
`install.sh` carrega os digests do release **anterior**. Quem instala por
`git checkout vX.Y.Z && ./install.sh` cai no `checksums.txt`, com aviso. O asset
do release é a cópia correta para instalação fixada por versão.
