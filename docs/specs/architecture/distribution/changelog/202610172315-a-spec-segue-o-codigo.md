# A spec segue o código

**Data:** 2026-10-17
**Specs afetadas:** `202608072352-distribution` (`.r` RN-1 e RN-2, `.p` §2, §3, §4.1, §4.1.1, §5 e §6, `.i` passos 3 e 7, `.config` §4)

Correção de documentação. Nenhum comportamento muda; o texto passa a dizer o que
o código já faz.

## O que mudou

- **RN-2 do `.r`.** Dizia que o instalador "falha se a verificação falhar —
  nunca instala com aviso". O `.p` §3 e o `install.sh` fazem outra coisa desde
  o `202608201900`: SHA-256 sempre; substituição coberta pelo digest fixado
  **ou** pela assinatura, qualquer uma basta; e, quando nenhuma cobriu, instala
  dizendo isso — nunca não-verificado em silêncio. O changelog daquela decisão
  nunca chegou ao `.r`, e o `.r` tem precedência: um `.i` lido ao pé da letra
  mandaria "parar" diante do próprio instalador publicado. Agora o `.r` diz a
  regra que o `.p` segue.
- **RN-1 do `.r`.** Listava quatro canais, com o tap do Homebrew que saiu no
  `202608202100` (que marcava o `.r` como afetado e não o mudou), e dizia que
  nenhum canal compila localmente — o `go install` compila. São três canais; o
  script e o download direto entregam o mesmo artefato, e o `go install`
  compila o mesmo código da tag e se apresenta como build local.
- **`.p` §2.** Ainda publicava `dcode_<versão>_windows_<arch>.zip`. O
  `release.yml` constrói só `darwin/linux × amd64/arm64`, as plataformas com
  backend de sandbox. O formato de nome de artefato é `stable` (§1): tirar o
  Windows da spec **não** quebra nada publicado, porque nenhum release jamais
  carregou um `.zip` — a spec prometia um artefato que nunca existiu. O formato
  dos `.tar.gz` não muda.
- **`.p` §4.1.1.** O texto de versão local dizia `0.0.0-dev+<commit>`. O
  `Makefile` nomeia o build pela versão para a qual vai: `<próxima>-dev+<commit>[.dirty]`,
  por exemplo `0.24.0-dev+abc1234`.
- **Números de passo.** O §4.1 dizia "passo 4 antes do 5" sobre extrair e
  conferir antes do `rename`, que hoje são os passos 6 e 7; o §5 dizia "os
  passos 10 e 11", e o 11 era o do tap que saiu.
- **Invariante e `.i` passo 7 dos canais.** "Todos os canais entregam o mesmo
  SHA-256" não vale para o `go install`. A invariante passa a falar dos canais de
  release; o fragmento que o `specguard` reivindica não muda.
- **`.i` passo 3 e `.config` §4.** Diziam que a ausência do cosign "avisa e
  segue". Avisa só quando o digest fixado também não cobriu.
- **Índices de changelog.** O `.r`, o `.i` e o `.config` diziam "sem alterações
  desde a criação" com oito entradas na pasta; o `.p` listava três.

## Por quê

Spec que contradiz o código ensina a regra errada a quem lê primeiro a spec, e o
`.r` é o primeiro que se lê.
