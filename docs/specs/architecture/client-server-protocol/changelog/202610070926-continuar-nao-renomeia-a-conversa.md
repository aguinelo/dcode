# Continuar não renomeia a conversa

**Data:** 2026-10-07
**Specs afetadas:** `202608072240-client-server-protocol` (`.p`: o comentário de
`Title` e um parágrafo na §5.2, duas linhas novas na §9)
**Fonte:** visto no desktop depois da troca de modelo do #424 (D28): a linha da
lateral e o título da conversa aberta discordavam.
**Revê:** "Uma conversa continuada sem pergunta própria herda o título da que ela
continua, seguindo a cadeia alguns passos", em
[202609302328 — Uma lista de conversas, vivas e gravadas](202609302328-uma-lista-de-conversas.md).

## O defeito

Uma conversa continuada numa sessão nova (`CreateSessionRequest.Resume`)
aparecia com dois títulos. A linha da lateral, que vem de `GET /v1/conversations`,
tomava a primeira pergunta feita na sessão nova — "e no qwen, onde fica?". O
painel aberto, que lê os eventos da sessão com o histórico carregado desde o
começo, mostrava a primeira pergunta da conversa — "onde fica a fila de
webhooks?".

A lista resumia cada sessão pelo próprio registro, e o registro de uma
continuação guarda o marcador (`session.resumed`), não o histórico carregado. A
herança do título valia só para a continuação que ainda não tinha perguntado
nada, e acabava na primeira pergunta dela; seguia a cadeia no máximo oito passos;
e o fluxo da lista olhava um passo só, entre os registros que alguma listagem já
tivesse lido. Quando a troca de modelo fechava a sessão deixada logo depois de
abrir a nova, o fluxo perdia a origem até o próximo retrato.

## A decisão

O título é da conversa, não da sessão. A sessão que continua outra é a mesma
conversa — é o gesto do D21 e do D28, e é o que a janela mostra —, então:

- o título derivado é a primeira pergunta feita em qualquer das sessões dela, não
  a primeira feita na sessão em que ela está agora;
- o nome em vigor é o último dado em qualquer delas: o da própria sessão quando
  ela deu ou apagou um, senão o da mais próxima antes dela que deu. Apagar o nome
  numa continuação devolve a primeira pergunta da conversa, não o nome de antes —
  é o que a janela faz, lendo o último `session.renamed` do histórico inteiro.

É o que o histórico lido pelo `Carry` diz, e o teste compara as duas leituras em
cada formato de cadeia.

Descartados:

- **A janela tomar o título da lista.** As duas concordariam no título errado: a
  conversa passaria a se chamar pela pergunta feita depois de trocar de modelo.
- **Gravar o título herdado no marcador `session.resumed`.** Cada registro se
  bastaria, mas mudaria a carga de um evento, e os registros de antes
  continuariam errados.

## Como

- Cada linha guarda do que o título é feito — a primeira pergunta da sessão, o
  último nome dado nela e se ela deu ou apagou algum —, e não o título. O título
  é decidido na vista da linha, que o retrato e o fluxo usam igualmente, seguindo
  `continued_from` pelas conversas vivas e pelas gravadas como o `Carry` segue: até
  o começo, uma volta no máximo num ciclo, e parando num registro podado.
- Conversa que termina fica guardada como terminou, para a continuação achar a
  origem sem esperar uma listagem reler o registro — o caso da troca de modelo.
- A vista não lê disco nem pega a trava do cache de registros, que fica presa
  enquanto arquivos são lidos. Antes, uma continuação sem pergunta própria podia
  esperar uma leitura de disco sob a trava do log da sessão; agora nenhuma das
  duas travas é pega sob a outra.

## As invariantes

Duas linhas novas na §9. "Continuar não renomeia a conversa" é reivindicada por
`TestAContinuedConversationKeepsItsTitle` (`internal/app`, pelo daemon, nos dois
gestos — reabrir a terminada e trocar de modelo —, comparando o retrato e o fluxo
com o que a janela lê dos eventos da sessão). "O que o histórico inteiro dela
diz" é reivindicada por `TestAContinuationIsTitledByItsWholeConversation`
(`internal/session`: origem nomeada, nome dado e apagado depois de continuar,
registro podado, ciclo e uma cadeia de onze sessões, cada uma contra o `Carry`).
`TestAContinuationKeepsItsTitleWhenWhatItContinuesEnds` segura o fluxo quando a
origem termina sem listagem no meio.

Nenhum tipo nem rota mudou; muda o que `title` diz de uma conversa continuada,
corrigindo o que a lista mostrava: PATCH.

## O que fica de fora

- **A listagem da TUI** (`internal/session/browse.go`, do `-r` e do
  `dcode sessions`) continua titulando cada sessão pelo próprio registro. É
  mudança da TUI, que passa a usar a rota da lista no PR dela.
- **Renomear a origem de uma continuação viva** não manda a continuação pelo
  fluxo: o título novo aparece no próximo retrato. Renomear uma conversa gravada
  já não avisava a lista nem da linha dela.
- **Os números de uma continuação** — turnos, selo, diff — continuam sendo os da
  sessão, não os da conversa.
