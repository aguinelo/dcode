# O id fica na pasta em toda porta

**Data:** 2026-10-13
**Specs afetadas:** `202608072240-client-server-protocol` (`.p`: duas linhas
novas em §9)
**Fonte:** o que ficou de fora de 202610131800: nomear ainda juntava o id ao
caminho sem passar por `ValidID`. Olhando a mesma linha, o `resume` também.

## O problema

Um id de conversa vira nome de arquivo — `<pasta>/<id>.jsonl` — em três
lugares do daemon. A leitura de conversa gravada passou a segurá-lo na pasta;
os outros dois não:

- **Nomear** (`POST /sessions/{id}/name`), para conversa que não está
  carregada, **acrescenta** uma linha ao arquivo. O mux decodifica `%2F` no
  valor do caminho, então `..%2Fsecret` chegava ao handler como `../secret`, e
  um `session.renamed` era escrito num `.jsonl` um nível acima da pasta. Reproduzido:
  o teste vermelho mostra a linha lá.
- **Continuar** (`resume` no pedido de criação) **lê** o arquivo duas vezes
  (`Rebuild` e `Carry`). `../secret` continuava um registro fora da pasta: o
  conteúdo ia para o contexto do modelo e era copiado para o registro da sessão
  nova.

## O que mudou

- `session.Rename` chama `ValidID` antes de montar o caminho. Id fora do
  alfabeto fechado é `invalid_input`, e a rota já repassa erro de protocolo
  como é. Id vazio, que antes caía em `ErrNoSuchSession`, agora é
  `invalid_input` também: é um id malformado, não uma conversa que falta.
- O construtor de sessão do daemon chama `ValidID(req.Resume)` antes de juntar
  o caminho. Antes, um id assim respondia `session_not_found` com o caminho
  inteiro na mensagem — quando o arquivo não existia — ou continuava o que
  achasse, quando existia.

Validado na função que monta o caminho, e não só na rota, porque é ali que o id
vira arquivo: qualquer outro chamador herda a regra sem saber dela.

## As invariantes

As duas linhas novas em §9 são reivindicadas por
`TestNamingAnIdThatLeavesTheRecordDirectoryIsRefusedByTheDaemon`
(`internal/server`, pela rota HTTP, com `%2F` e `%2E` escapados) e
`TestContinuingAnIdThatLeavesTheRecordDirectoryIsRefused` (`internal/app`).
`TestNamingAnIdThatLeavesTheRecordDirectoryIsRefused` (`internal/session`)
cobre a função sozinha.

Correção: PATCH. Nenhum id que o daemon gera fica fora do alfabeto, então
nenhum cliente que usa ids que recebeu muda de comportamento.
