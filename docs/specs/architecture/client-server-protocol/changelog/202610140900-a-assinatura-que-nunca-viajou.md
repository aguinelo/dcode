# A assinatura que nunca viajou sai do fio

**Data:** 2026-10-14
**Specs afetadas:** `202608072240-client-server-protocol` (tipos do pacote
`internal/protocol`; o `.p` nunca listou os dois na §5.1). Decisão Q1 C da
proposta `desktop/docs/proposals/loop.md`, aprovada pelo dono em 2026-10-13.

## O que mudou

Saem do protocolo:

- o evento `done.signed` (`EventDoneSigned`);
- o tipo `SignDoneRequest`.

Os dois nasceram com a `done-qualifier`, para uma assinatura que seria um
ida-e-volta bloqueante dentro do turno. Nenhum emissor, nenhuma rota, nenhum
leitor além do desktop, que os decodificava por estarem no `generated.ts`.

`done.proposed` e `DoneProposal` ficam: o passo seguinte da proposta os religa
como aviso, depois que o `done.toml` é escrito.

## Por quê

A assinatura passou a ser o arquivo revisado
(`done-qualifier/changelog/202608270700-o-laco-le-as-ferramentas.md`), e o que
sobrou no fio era promessa versionada que nenhum código cumpria. Um cliente que
lesse os tipos acreditaria numa rota que não existe. O detalhe da decisão está
em `done-qualifier/changelog/202610140900-a-assinatura-e-o-arquivo.md`.

Nível `experimental`, abaixo de 1.0: contrato removido é **MINOR**. Como nunca
houve emissor, nenhum cliente dependia da forma — o desktop só deixa de
decodificar um evento que nunca chegou.
