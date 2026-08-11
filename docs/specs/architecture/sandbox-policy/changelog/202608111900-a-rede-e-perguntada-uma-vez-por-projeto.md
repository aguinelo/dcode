# A rede é perguntada uma vez por projeto

**Data:** 2026-08-11
**Specs afetadas:** `202608072336-sandbox-policy` (`.r`, `.p`), `202608072337-tool-suite` (`.p`)

> **Regra:** um comando de shell declara que **pode** alcançar a rede, sempre. A
> pergunta é feita uma vez por projeto e a resposta é guardada na configuração
> do **usuário** — nunca na do projeto.

## O que estava errado

A US-2 diz que rede é decisão do usuário, e o produto tinha dois estados,
ambos ruins:

- `sandbox.allow_network=false`, o default: a rede era bloqueada pelo sistema
  operacional e **nenhuma pergunta era feita**. `curl localhost:3000` no próprio
  servidor que o agente acabou de subir falhava igual a `curl google.com`, sem
  nada explicando por quê.
- `sandbox.allow_network=true`: o `bash` passava a declarar rede em **todo**
  comando, e todo comando escalava. Aprovar vinte vezes por sessão é o padrão
  que ensina a aprovar sem ler.

Não havia estado em que a fronteira existisse e fosse utilizável.

## Por que a declaração muda

O `bash` declarava rede só quando o sandbox já permitia, e o raciocínio era
correto **naquele desenho**: com o SO bloqueando, aprovar não concedia nada e
negar derrubava o comando inteiro em vez do acesso à rede — o usuário respondia
uma pergunta que não era a da tela.

Essa premissa acabou. O sandbox passou a ser consultado **por comando**, e uma
concessão abre a fronteira para o comando que provocou a pergunta. Aprovar
agora significa o que aparenta significar.

E a declaração honesta é *sempre*: um comando de shell é opaco. Um build resolve
dependências, uma suíte baixa uma imagem, um formatador checa se há versão nova.
Não existe leitura da string que responda se **este** sai para a rede.

## O escopo da resposta

A pergunta é sobre o **projeto**, e o texto do modal diz isso. Dizer "permitir
este comando" prometeria algo mais estreito do que a resposta faz.

| Resposta | Vale para | Guardada |
|---|---|---|
| `[a]` uma vez | esta sessão | não |
| `[P]` este projeto | este workspace, sempre | sim |
| `[G]` sempre | todo workspace | sim |
| `[d]` não | esta sessão | não |

As duas que ficam gravadas exigem maiúscula: mais difícil de apertar sem querer,
que é o que a consequência maior merece.

**Negar vale pela sessão e não é gravado.** Uma recusa que sobrevivesse ao
reinício seria uma fronteira que o usuário fechou uma vez e depois não acharia
mais — e cuja única saída seria editar um arquivo que ele não sabe que existe.

## Onde a concessão mora

Na raiz de configuração do **usuário**, nunca no workspace. Um registro dentro
do projeto deixaria um repositório chegar pré-aprovado: clonar alguma coisa a
autorizaria antes de qualquer pessoa ler uma linha dela, que é o oposto de
perguntar.

O arquivo é escrito `0600`, o mesmo modo das credenciais e pelo mesmo motivo:
registra o que o usuário permitiu, e permissão que qualquer um na máquina edita
não é permissão que o usuário deu.

## O que não muda

A avaliação de política. Rede continua sendo `escalate` em workspace-write e
`deny` em read-only; o sandbox continua contendo. O que mudou é **com que
frequência** uma pessoa é perguntada sobre uma decisão que já tomou.
