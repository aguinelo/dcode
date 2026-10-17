## en

- **The pre-push gate picks its gates from what the branch changes against
  `main`.** It compared with the branch's old remote head, so a branch brought
  up to date with `main` ran the desktop gate for protocol changes that came
  from `main`, and refused the push where `desktop/node_modules` was missing.

## pt-BR

- **O gate de pre-push escolhe os gates pelo que a branch muda em relação à
  `main`.** Ele comparava com o head antigo da branch no remoto, então uma
  branch atualizada com a `main` rodava o gate do desktop por mudanças de
  protocolo que vieram da `main`, e recusava o push onde faltava
  `desktop/node_modules`.
