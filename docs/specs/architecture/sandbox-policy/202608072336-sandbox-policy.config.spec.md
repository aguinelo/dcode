# Config: Sandbox e Política de Permissão

> Nenhuma variável de ambiente nova no código sem estar aqui.
> Precedência: **config travada por administrador** > flag > variável de ambiente > arquivo de config > default.

> `HARNESS_SANDBOX_MODE`, `HARNESS_APPROVAL_POLICY` e `HARNESS_APPROVAL_TIMEOUT` são declaradas em `202608072240-client-server-protocol.config.spec.md`, seção 3, porque o protocolo as expõe em `CreateSessionRequest`. Esta spec **não** as redeclara — ela define o comportamento que elas controlam.

## 1. Mecanismo de sandbox

| Variável | Tipo | Default | Uso |
|---|---|---|---|
| `HARNESS_SANDBOX_BACKEND` | enum | `auto` | `auto`, `seatbelt`, `bubblewrap`, `none`. `auto` escolhe pelo sistema operacional. `none` **só é aceito** junto de `HARNESS_SANDBOX_MODE=full-access` — em qualquer outro modo é erro de inicialização, porque prometeria fronteira que não existe (RN-3). |
| `HARNESS_SANDBOX_BIN` | caminho | vazio | Caminho do binário do mecanismo. Vazio procura no `PATH`. Existe para instalação fora do padrão, não para substituir por outro programa. |
| `HARNESS_SANDBOX_PROFILE_DIR` | caminho | `$HARNESS_STATE_DIR/profiles` | Onde os perfis gerados são escritos. Perfil é gerado por sessão a partir do workspace e do modo. |

## 2. Rede

| Variável | Tipo | Default | Uso |
|---|---|---|---|
| `HARNESS_ALLOW_NETWORK` | booleano | `false` | Concede rede sem escalonar, dentro do sandbox. Equivale a tratar rede como dentro da fronteira. Ligar reduz atrito e reduz segurança — é exatamente o trade-off que a ADR-02 torna explícito em vez de escondido. |
| `HARNESS_NETWORK_ALLOWLIST` | lista | vazio | Hosts permitidos sem escalonar, mesmo com `HARNESS_ALLOW_NETWORK=false`. Vazio não permite nenhum. Aplicado pelo perfil do sandbox quando o mecanismo suporta; onde não suportar, a sessão falha na criação em vez de ignorar a lista. |

## 3. Governança

| Variável | Tipo | Default | Uso |
|---|---|---|---|
| `HARNESS_REQUIREMENTS_FILE` | caminho | vazio | Arquivo de política travada por administrador. Valores definidos aqui **não** são sobrescrevíveis por variável de ambiente nem por flag (RN-7). Ausente, não há travamento. |
| `HARNESS_ALLOW_FULL_ACCESS` | booleano | `true` | Quando `false`, `full-access` é recusado mesmo se pedido. Só faz sentido no arquivo de requisitos — em variável de ambiente, quem pode mudar a variável pode mudar esta também. |

> O arquivo de requisitos é o análogo do `requirements.toml` do Codex, elogiado na ADR-02 como o que torna o harness adotável em organização. Separar config de usuário de config travada é o ponto inteiro.

## 4. Diagnóstico

| Variável | Tipo | Default | Uso |
|---|---|---|---|
| `HARNESS_SANDBOX_TRACE` | booleano | `false` | Registra cada `Verdict` em nível `debug`: operação, caminho resolvido, fronteira, decisão. Verboso; para entender por que algo foi negado. |

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
