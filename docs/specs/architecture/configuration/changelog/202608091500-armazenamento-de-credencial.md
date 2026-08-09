# Armazenamento de credencial

**Data:** 2026-08-09
**Specs afetadas:** `202608081203-configuration` (`.p`, `.config`)

## O que mudou

A spec recusava credencial no `config.toml` e parava aí. Agora existe onde
guardar: `internal/credential`, com keychain do SO onde houver e arquivo `0600`
na raiz de estado onde não houver, mais `dcode login` e `dcode config`.

**Uma credencial por família**, não uma global.

**A escolha do backend é configuração** (`credential.backend`), não flag do
comando que escreve.

## Por que mudou

**Recusar o lugar errado sem oferecer o certo não protege ninguém.** A
consequência observada: a chave vai para o `.zshrc` ou é colada num terminal —
e, na sessão que produziu esta mudança, acabou no transcript de uma conversa
com IA. O segredo apenas mudou para um lugar que não controlamos e não
auditamos.

**Por família porque `/model` já existe.** Ele troca o modelo e mantinha a
chave, então ir de MiniMax para Claude falhava com erro de auth. Chave única
teria exigido reconfigurar a cada troca, e o formato guardado mudaria depois de
qualquer forma.

**Configuração e não flag** porque a primeira versão tinha `--backend` só no
`login`: gravava no arquivo, e `config` e o app liam do keychain. Escrever num
lugar que nada lê é pior que não escrever — parece que funcionou.

## Sobre exibir a chave

A discussão que originou isto foi querer **ver o modelo e a chave para editar e
confirmar**. A necessidade é legítima: "configurada: sim" não ajuda a achar uma
chave colada da conta errada.

A resposta é máscara com início, fim e impressão digital — que responde a
pergunta inteira — mais revelação explícita para o caso de recuperar a chave.
Imprimir por padrão entregaria o segredo a screenshot, screen share, scrollback
e gravação de uma vez só, e a necessidade não exige isso.

## Impacto

- `DCODE_API_KEY` continua vencendo a store: explícito e por invocação.
- `provider.FamilyFor` passa a ser exportada — a família é necessária **antes**
  do provider existir, porque é ela que nomeia a credencial com que o transporte
  é construído. Duas implementações dessa correspondência acabariam discordando
  sobre qual chave um modelo usa, e o sintoma só apareceria como erro de auth.
- O backend macOS passa o segredo por argumento no `add-generic-password`, que
  é o único ponto onde este pacote não consegue evitar: `security` não oferece
  forma por stdin para escrita. Está documentado no código como exposição real e
  aceita apenas porque a alternativa é não guardar nada. O backend de arquivo e
  o `secret-tool` não têm esse furo.

## Alternativa descartada

Só arquivo `0600`, sem keychain. Descartada porque a raiz de estado no macOS é
`~/Library/Application Support`, que pode ser sincronizada — e sincronizar um
arquivo de segredos é o mesmo problema que motivou recusar o `config.toml`.
