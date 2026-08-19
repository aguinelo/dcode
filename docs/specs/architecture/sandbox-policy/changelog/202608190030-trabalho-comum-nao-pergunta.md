# Trabalho comum não pergunta, destruição pergunta sempre

**Data:** 2026-08-19

## O que mudou

Duas coisas, e são as duas metades da mesma decisão:

- **`sandbox.allow_network` passa a `true` por default.** Rede concedida deixa
  de ser pergunta.
- **`DefaultRules()` passa a trazer a lista de comandos destrutivos** que a spec
  declarava desde sempre e o código nunca teve.

## Por que a primeira

`bash` declara rede **sempre** — um comando é opaco, então declara-se o pior
caso. Essa declaração virava pergunta sozinha, e a consequência não estava
escrita em lugar nenhum: onde ninguém podia responder, o shell inteiro era
negado.

Medido numa execução autônoma real, com `approval.policy = never`: **120
chamadas de ferramenta, zero comandos executados.** O `make check` — que é a
definição de pronto deste repositório — nunca teve como rodar. O modelo
compensou conferindo por leitura: 73 `grep` para responder o que um `go test`
responderia.

Um agente que pode editar e nunca verificar produz mudança que ninguém conferiu,
que é pior que mudança nenhuma.

## Por que a segunda

A linha `DCODE_CONFIRM_COMMAND` da spec declarava defaults — `rm -rf`,
`git push`, `curl … | sh` — e `DefaultRules()` mandava lista vazia. A promessa de
confirmar antes de destruir existia na documentação e em nenhum outro lugar.

Sem ela, "deixa tudo liberado" seria liberar também o irreversível.

## A terceira mudança, que só apareceu ao testar

Regras eram **puladas** sob `never`. O raciocínio estava escrito e era bom: uma
regra pede a atenção de uma pessoa; sem pessoa não há pergunta, e transformar
pergunta impossível em negação faria `never` mais restritivo que `on-request` —
o oposto do que o nome diz.

Valia enquanto regras eram atenção sobre caminhos. Parou de valer quando
passaram a carregar destruição: `never` permitia `rm -rf /` direto. **A única
configuração sem ninguém olhando era a única que não parava.**

Agora `never` é mais restritivo que `on-request` onde uma regra dispara, e é o
que deveria dizer desde o início: ele não desliga as perguntas, ele as responde
— e a única resposta segura sem ninguém presente é não.

## O que NÃO mudou

**A contenção.** `workspace-write` continua o default e `full-access` continua
opt-in. Mudou o que é **perguntado**, não o que é **contido**. Conceder rede é
autorização e nunca contenção: em `read-only` não há rede para conceder, e dizer
sim não a materializa.

## O que a lista de comandos não é

Não é fronteira e não pode ser. Comando é texto, e a mesma destruição sempre pode
ser escrita de outro jeito — por script, por alias, por variável. O que ela
compra é atrito contra o acidente, que é o que de fato acontece. Contenção é o
sandbox, e só ele. Está escrito ao lado da lista, no código, para que ninguém a
leia como garantia.

## Como voltar atrás

`sandbox.allow_network = false` no arquivo de configuração devolve a pergunta da
rede. `DCODE_CONFIRM_COMMAND` substitui a lista de comandos. Nenhuma das duas
mexe no sandbox.
