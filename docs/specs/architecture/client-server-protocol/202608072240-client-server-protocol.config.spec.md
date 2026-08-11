# Config: Protocolo Client-Server do dcode

> Nenhuma variável de ambiente nova pode aparecer no código sem estar aqui.
> Precedência: flag de linha de comando > variável de ambiente > arquivo de config > default.

## 1. Transporte e estado

| Variável | Tipo | Default | Uso |
|---|---|---|---|
| `DCODE_SOCKET` | caminho | `$XDG_RUNTIME_DIR/dcode.sock`, ou `$TMPDIR/dcode-$UID.sock` se ausente | Caminho do socket de domínio Unix. Criado com `0700`. Removido no encerramento limpo; socket órfão é detectado por tentativa de conexão e removido. |
| `DCODE_STATE_DIR` | caminho | `$XDG_STATE_HOME/dcode`, ou `~/.local/state/dcode` | Raiz de logs de sessão e dados persistentes. |

## 2. Retenção de eventos

| Variável | Tipo | Default | Uso |
|---|---|---|---|
| `DCODE_EVENT_SPILL` | caminho | `$DCODE_STATE_DIR/events` | Onde os eventos que saem da memória são guardados. Retenção **sem** transbordo é horizonte duro: um cliente que demorou demais para voltar recebe `events_expired`, e a sessão que ele acompanhava fica ilegível por um motivo que não tem nada a ver com a sessão. Vazio desliga e devolve o comportamento antigo. |

> O log de evento **é** a sessão — o cliente guarda um número e todo o resto vive no servidor, e é isso que torna reconectar indistinguível de ter acompanhado ao vivo. Um horizonte de retenção quebra exatamente essa propriedade, e quebra em silêncio: o usuário vê uma sessão que "sumiu".
>
> Nada é descartado se não puder ser guardado. Memória crescendo é visível e recuperável; buraco silencioso no meio da sessão não é nenhum dos dois.

## 3. Aprovação e sandbox

Valores espelham a ADR-02. O protocolo transporta a decisão; a fronteira é aplicada pelo sistema operacional.

| Variável | Tipo | Default | Uso |
|---|---|---|---|
| `DCODE_SANDBOX_MODE` | enum | `workspace-write` | `read-only`, `workspace-write`, `full-access`. Default por sessão; sobrescrevível em `CreateSessionRequest.SandboxMode`. |
| `DCODE_APPROVAL_POLICY` | enum | `on-request` | `untrusted`, `on-request`, `never`. Política de escalonamento, **ortogonal** ao modo de sandbox. |

> `DCODE_SANDBOX_MODE=full-access` combinado com `DCODE_APPROVAL_POLICY=never` remove toda fronteira. O servidor registra aviso na inicialização quando essa combinação está ativa.

> **Correção de nome.** Esta spec dizia `danger-full-access` e o `202608072336-sandbox-policy.p.spec.md` diz `full-access`, ambas marcadas `stable`, e o código sempre implementou `full-access`. Duas specs discordando de um valor que as duas declaram estável é pior que uma incompleta — quem lê uma delas escreve algo que a inicialização recusa. O nome do código vence, pelo mesmo motivo de `sandbox.policy` → `sandbox.approval_policy`: o nome interno passa a ser o que o usuário escreve.
>
> O prefixo `danger-` era um aviso embutido no valor, e é uma ideia boa que se resolve melhor onde o aviso é lido: o modo aparece em destaque permanente na linha de status, e a combinação com `never` avisa no boot.

## 4. Runtime Go

Consequência direta da ADR-01: o custo aceito da escolha por Go é pressão de GC sob swarm.

| Variável | Tipo | Default | Uso |
|---|---|---|---|
| `GOMEMLIMIT` | tamanho | não definido | Teto flexível de memória do processo. Recomendado em ambiente com muitas sessões; sem ele o GC só reage a `GOGC`. |
| `GOGC` | inteiro | `100` (padrão do Go) | Reduzir para `50` troca CPU por menor pico de heap — avaliar sob carga real de swarm, não por palpite. |

## 5. Observabilidade

| Variável | Tipo | Default | Uso |
|---|---|---|---|

## 6. Constantes não configuráveis

Fixas em código, documentadas porque afetam o contrato observável.

| Constante | Valor | Motivo |
|---|---|---|
| Intervalo de ping SSE | `20s` | Abaixo do timeout de inatividade comum em proxies. |
| Prefixo de versão | `/v1` | Muda apenas em incremento de major. |
| Permissão do socket | `0700` | Implementa RN-6; não afrouxar sem spec de autenticação. |
| Formato de ID de sessão | ULID | Ordenável por tempo e seguro para nome de arquivo. |

## 7. Changelog

_Sem alterações desde a criação._
