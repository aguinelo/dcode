## en

- **A request the daemon cannot read answers `400 invalid_input`, on every
  route.** A malformed body, an image that is not base64 and a `from` that is
  not a number answered `500 internal` on most session routes (and
  `workspace_invalid` on create), which tells a client the daemon broke and
  that retrying might help. The git and loop routes already answered
  `invalid_input`; now all do. The error table in the protocol spec also lists
  `no_active_turn` and `invalid_input`, which the daemon already answered.

## pt-BR

- **Pedido que o daemon não consegue ler responde `400 invalid_input`, em toda
  rota.** Corpo malformado, imagem que não é base64 e `from` que não é número
  respondiam `500 internal` na maior parte das rotas de sessão (e
  `workspace_invalid` na criação), o que diz ao cliente que o daemon quebrou e
  que tentar de novo pode adiantar. As rotas de git e de loop já respondiam
  `invalid_input`; agora todas respondem. A tabela de erros da spec do
  protocolo passa a listar `no_active_turn` e `invalid_input`, que o daemon já
  respondia.
