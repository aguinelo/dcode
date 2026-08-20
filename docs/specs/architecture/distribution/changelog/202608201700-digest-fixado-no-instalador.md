# O instalador carrega o digest do release que instala

**Data:** 2026-08-20
**Specs afetadas:** `202608072352-distribution` (`.p`, `.i`)

## O que mudou

O `install.sh` passa a ter um bloco entre `# BEGIN PINNED` e `# END PINNED` com
os SHA-256 dos quatro artefatos publicados, escrito por `scripts/installer.sh` a
partir do `checksums.txt` **já assinado**. Quando o artefato baixado tem digest
fixado, é contra ele que se compara — e divergência aborta, mesmo que o
`checksums.txt` do release concorde com o download.

O bloco nasce vazio, e o vazio é silencioso: o instalador sem pino cai no
`checksums.txt` sem reclamar. Instalador fixado num release e chamado para outro
avisa e aponta o instalador daquele release, que é o que carrega os digests
certos.

Este changelog cobre o **mecanismo**. A geração no pipeline vem a seguir.

## Por que mudou

Porque o `checksums.txt` viaja do **mesmo host** que o tarball. Sozinho ele pega
download corrompido ou truncado; não pega release substituído, porque quem troca
um troca o outro e o par continua coerente consigo mesmo. Só a assinatura cobria
substituição — e o #222 acabou de tornar a assinatura opcional, por motivo bom
(exigi-la deixava quem não tem cosign sem binário e sem verificação nenhuma).

Opcional não pode virar decorativa. O digest fixado é o que fecha a janela que o
#222 abriu, e ele fecha **por construção**: o valor esperado passa a viver no
histórico do git. Um asset de release pode ser trocado sem deixar rastro
público; uma linha num arquivo versionado só muda por commit — visível no log,
no diff, atribuído. São duas rotas independentes, e o atacante precisa das duas.

É a doutrina de camadas deste repositório aplicada onde ela vale mais: garantia
estrutural acima de frase. O #222 escreveu "nunca não-verificado em silêncio";
isto faz a frase ser verdade sem depender de ninguém a cumprir.

## De onde veio o desenho

Do `uv`, que é o único dos seis instaladores examinados (rustup, bun, deno, nvm,
k3s, uv) mais rigoroso que este. Ele embute o digest no próprio script, gerado
por release. Os outros quatro não verificam nada, e **nenhum dos seis exige
ferramenta externa de verificação**.

A separação do uv é melhor que a nossa: o instalador dele vem do `astral.sh` e o
artefato do `github.com` — dois hosts, duas credenciais. Sem domínio próprio, a
melhor separação disponível aqui é histórico do git contra asset de release. É
mais fraca, e é honesto dizer que é.

## Alternativa descartada

**Gerar o `install.sh` inteiro a partir de um template em heredoc**, como o
`scripts/formula.sh` faz com o `dcode.rb`. Descartada: o `install.sh` é cheio de
`$`, e aninhá-lo num heredoc de outro script shell exige escapar tudo — na
prática, uma segunda cópia do instalador para manter em sincronia com a primeira.
Este repositório já sabe o que acontece com a segunda cópia.

O bloco marcado mantém o `install.sh` sendo um script real, editável à mão e
executável direto da árvore, com apenas o pedaço derivado sendo derivado.

## Alternativa descartada, segunda

**Verificar que o digest fixado e o do `checksums.txt` concordam.** Chegou a ser
escrita e foi removida antes de virar código: os dois são comparados contra o
mesmo digest do arquivo baixado, então a checagem cruzada é inalcançável. A
proteção inteira está em comparar o fixado com o download, e é ali que ela está.

## Alternativa descartada, terceira

**`sed -i` ou `python3` para trocar o bloco.** A troca é feita em `awk`.

`sed -i` diverge entre GNU e BSD, e este repositório já perdeu uma noite com um
`sed ... t;` que funcionava no Ubuntu e falhava no macOS — numa matriz de duas
plataformas, é a metade dos jobs que reprova por motivo que não é o código.

`python3` seria uma dependência nova do pipeline de release. O
`scripts/formula.sh`, que faz o trabalho equivalente no outro canal, é bash puro,
e no macOS o `python3` nem sempre está lá. Ele chegou a ser usado e foi trocado
antes do merge.
