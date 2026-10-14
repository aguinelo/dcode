# A proposta escrita é anunciada

**Data:** 2026-10-14
**Specs afetadas:** `202608261730-done-qualifier` (`.p` §6.1, §9). Decisão Q1 C
da proposta `desktop/docs/proposals/loop.md`, passo 2 da §8, aprovada pelo dono
em 2026-10-13.

## O que mudou

`done.proposed` ganha emissor. O commit da proposta — `POST
/v1/sessions/{id}/done`, que o laço chama quando o turno qualificador termina —
mede os critérios, escreve o `done.toml` e **só então** emite `done.proposed`
no log da sessão qualificadora, com o `DoneProposal` medido.

O `DoneProposal` muda de forma para dizer o que é:

| Campo | Antes | Agora |
|---|---|---|
| `path` | — | o `done.toml` escrito |
| `turn_id` | o turno que esperava a assinatura | sai: o turno já terminou |
| `round` | a rodada do ida-e-volta | sai: não há rodada |
| `expires_at` | o prazo que recusava | sai: nada espera resposta |
| `proposal_id`, `criteria`, `protected`, `no_acceptance` | — | ficam, e são o que o arquivo carrega |

Duas invariantes novas na §9: escrever a proposta emite o aviso uma vez, com a
medição e o caminho; proposta que não chegou a ser escrita não é anunciada.

## Por quê

O desktop precisa desenhar a revisão (Figura 3 da proposta): a classe de cada
critério e a discordância entre o que o proponente disse e o que a medição
achou, que é onde o olho deve cair (RN-2, RN-3 da `.r`). Hoje isso só existe em
dois lugares — no arquivo, que o renderer não lê (o desktop não toca o disco),
e no `Summary`, que é prosa. Um cliente que parseasse prosa seria um segundo
dono do formato.

É **aviso** e não pergunta porque a assinatura é o arquivo revisado
(`202610140900-a-assinatura-e-o-arquivo`). Um evento que esperasse resposta
reabriria o ida-e-volta que acabou de sair.

Emitido **depois** de escrever, e só se escreveu: o aviso diz "existe um
arquivo para revisar". Anunciar antes, ou anunciar uma proposta que falhou ao
gravar, mandaria a pessoa ler um arquivo que não existe.

## O que isto não muda

- `CommitDoneResponse` continua igual: o laço da TUI lê o `Summary` como lia.
- O protocolo é `experimental`; acrescentar um emissor e mudar a forma de um
  payload que nunca foi emitido é **MINOR** como feature.
