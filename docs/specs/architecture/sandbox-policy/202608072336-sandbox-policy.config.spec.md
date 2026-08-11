# Config: Sandbox e Política de Permissão

> Nenhuma variável de ambiente nova no código sem estar aqui.
> Precedência: **config travada por administrador** > flag > variável de ambiente > arquivo de config > default.

> `DCODE_SANDBOX_MODE`, `DCODE_APPROVAL_POLICY` e `DCODE_APPROVAL_TIMEOUT` são declaradas em `202608072240-client-server-protocol.config.spec.md`, seção 3, porque o protocolo as expõe em `CreateSessionRequest`. Esta spec **não** as redeclara — ela define o comportamento que elas controlam.

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
