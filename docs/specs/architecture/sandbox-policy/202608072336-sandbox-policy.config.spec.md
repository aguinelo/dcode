# Config: Sandbox e Política de Permissão

> Nenhuma variável de ambiente nova no código sem estar aqui.
> Precedência: **config travada por administrador** > flag > variável de ambiente > arquivo de config > default.

> `DCODE_SANDBOX_MODE` e `DCODE_APPROVAL_POLICY` são declaradas em `202608072240-client-server-protocol.config.spec.md`, seção 3, porque o protocolo as expõe em `CreateSessionRequest`. Esta spec **não** as redeclara — ela define o comportamento que elas controlam.
>
> `DCODE_APPROVAL_TIMEOUT` **não existe** como variável: o teto de aprovação é a flag `-approval-timeout` do daemon, e a chave foi removida em `202608110900` por nunca ter sido lida.

## 1. Mecanismo de sandbox

| Variável | Tipo | Default | Uso |
|---|---|---|---|
| `DCODE_SANDBOX_BACKEND` | enum | `auto` | `auto`, `seatbelt`, `bubblewrap`, `none`. `auto` escolhe pelo sistema operacional. `none` **só é aceito** junto de `DCODE_SANDBOX_MODE=full-access` — em qualquer outro modo é erro de inicialização, porque prometeria fronteira que não existe (RN-3). |

## 2. Rede

| Variável | Tipo | Default | Uso |
|---|---|---|---|
| `DCODE_ALLOW_NETWORK` | booleano | `false` | Concede rede sem escalonar, dentro do sandbox. Equivale a tratar rede como dentro da fronteira. Ligar reduz atrito e reduz segurança — é exatamente o trade-off que a ADR-02 torna explícito em vez de escondido. |

## 3. Governança

| Variável | Tipo | Default | Uso |
|---|---|---|---|
| `DCODE_CONFIRM_WRITE` | lista | `.git/**`, `.env*`, `.dcode/**`, arquivos de lock | Caminhos cuja **escrita** pede confirmação mesmo dentro do workspace. Não é contenção — é atenção: são coisas que parecem workspace e não são de mexer. |
| `DCODE_CONFIRM_READ` | lista | `.env*`, `**/*_rsa`, `**/credentials*` | Caminhos cuja **leitura** pede confirmação. |
| `DCODE_CONFIRM_COMMAND` | lista | `rm -rf`, `git push`, `curl … \| sh` | Comandos que pedem confirmação. Regra que casa texto de comando é regra que se contorna sem querer — por isso ela só **escalona** o que já era permitido, e nunca resgata o que foi negado. |

> O arquivo de requisitos é o análogo do `requirements.toml` do Codex, elogiado na ADR-02 como o que torna o dcode adotável em organização. Separar config de usuário de config travada é o ponto inteiro.


## 4. Diagnóstico

| Variável | Tipo | Default | Uso |
|---|---|---|---|

## 5. Constantes não configuráveis

| Constante | Valor | Motivo |
|---|---|---|
| Falha fechada quando o mecanismo não está disponível | sempre | RN-3; degradar silenciosamente promete o que não entrega. |
| Resolução de symlink antes da comparação | sempre | RN-4; sem isso a fronteira é contornável com um `ln -s`. |
| Comparação de contenção por componente | sempre | Prefixo de string deixa `/proj2` passar por `/proj`. |
| Passagem obrigatória pelo avaliador | sempre | RN-6; caminho alternativo é blocker. |
| `never` nega o que cruzaria | sempre | A alternativa seria conceder em silêncio. |

## 6. Changelog

_Sem alterações desde a criação._
