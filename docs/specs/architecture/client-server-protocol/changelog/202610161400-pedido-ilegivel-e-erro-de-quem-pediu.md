# Pedido ilegível é erro de quem pediu

**Data:** 2026-10-16
**Specs afetadas:** `202608072240-client-server-protocol` (`.p` §8 e §9).

## O que mudou

Corpo que o daemon não consegue decodificar, imagem em base64 inválido e
`from` fora do formato no fluxo de eventos passam a responder
`400 invalid_input`, com o que estava errado na mensagem, em toda rota:

| Rota | Antes |
|---|---|
| `POST /sessions` | `400 workspace_invalid` |
| `POST /sessions/{id}/name` | `500 internal` |
| `POST /sessions/{id}/turns` (corpo e imagem) | `500 internal` |
| `POST /sessions/{id}/exec` | `500 internal` |
| `POST /sessions/{id}/steer` | `500 internal` |
| `POST /sessions/{id}/mode` | `500 internal` |
| `POST /sessions/{id}/approvals/{approvalID}` | `500 internal` |
| `GET /sessions/{id}/events?from=` | `500 internal` |

As rotas de git e de loop já respondiam assim; agora todas respondem igual.

A tabela da §8 ganha as duas linhas que faltavam: `no_active_turn` (409) e
`invalid_input` (400). Os dois códigos já existiam em
`internal/protocol/errors.go` e já eram respondidos; a spec não os listava.

## Por quê

A §8 define `internal` como "falha não classificada". Um corpo malformado é
classificado — é erro de quem pediu — e `500` diz ao cliente que o daemon
quebrou e que tentar de novo pode adiantar, as duas coisas falsas.
`workspace_invalid` na criação mandava o cliente olhar o caminho do
workspace, quando o problema era o JSON.

`experimental`, abaixo de 1.0. Mudar o código de uma resposta é mudança de
contrato, **MINOR**: um cliente que tratava `500` como "tente de novo" passa a
receber `400` e deve mostrar a mensagem.
