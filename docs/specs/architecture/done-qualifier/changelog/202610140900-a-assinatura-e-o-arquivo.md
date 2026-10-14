# A assinatura é o arquivo, e o ida-e-volta sai

**Data:** 2026-10-14
**Specs afetadas:** `202608261730-done-qualifier` (`.p` §1, §2, §6, §9, §12;
`.config` §1.2, §2). Decisão Q1 C da proposta `desktop/docs/proposals/loop.md`,
aprovada pelo dono em 2026-10-13.

## O que mudou

Saíram, sem nunca terem tido emissor nem chamador fora de teste:

| O quê | Onde morava |
|---|---|
| `qualifier.Sign`, `Asker`, `SignedAnswer`, `ErrRefused`, `MaxSignRounds` | `internal/loop/qualifier/sign.go` |
| `done.signed` (`EventDoneSigned`) e `SignDoneRequest` | `internal/protocol` |
| `DCODE_QUALIFIER_MAX_SIGN_ROUNDS`, `DCODE_QUALIFIER_SIGN_TIMEOUT` | `.config` §1.2 — declaradas, nunca lidas |

Com elas, nove invariantes da §9 que só o `Sign` cumpria: remedição de
critério editado, critério acrescentado que volta, edição que assenta, recusa
por prazo, teto ou canal, assinar sem runner, assinar vazio, critério quebrado
fora do congelamento, conjunto que fica vazio. A décima — a condição do conjunto
mostrada junto dele — fica, porque o arquivo a mostra; o teste que a reivindica
passa a ser o do arquivo (`TestASetWithNothingRedWarnsInTheFile`).

`done.proposed` e `DoneProposal` ficam por enquanto. O passo 2 da proposta os
religa como aviso, emitido depois que o arquivo é escrito; apagá-los aqui para
recriá-los no PR seguinte seria uma volta no fio sem motivo.

## Por quê

O `202608270700-o-laco-le-as-ferramentas` trocou a assinatura dentro do turno
pelo arquivo revisado: a medição roda **fora** do turno, o `done.toml` é escrito
com a classe e a discordância de cada critério, e o laço para ali. A pessoa
assina revisando o arquivo — e é o arquivo que o próximo turno lê.

O resto do desenho antigo ficou para trás: um `Sign` que só os testes chamavam,
dois eventos que nada emitia, duas chaves de configuração que nada lia, e o
`specguard` mantendo tudo vivo com um motivo escrito ("nothing emits
done.proposed yet"). O desktop decodificava os dois eventos. Era promessa num
fio versionado que nenhum código cumpria, e cada uma dessas peças custava a
alguém descobrir que era cenário.

Religar como o `.p` original previa — assinatura bloqueante dentro do turno,
com prazo e rodadas — contradiria o desenho que ganhou e reabriria a RN-6 com
um relógio. A revisão não tem prazo: o laço espera a pessoa.

## O que isto não muda

- A RN-6 da `.r` continua: sem revisão não há laço. Quem a cumpre é o laço que
  para com o arquivo escrito, não um prazo que recusa.
- O protocolo está em `experimental`. Ainda assim é contrato removido, e por
  isso **MINOR** (`docs/conventions/VERSIONING.md` §3), com changelog também em
  `client-server-protocol`.
