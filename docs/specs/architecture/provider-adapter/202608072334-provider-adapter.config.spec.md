# Config: Adaptador de Provider

> Nenhuma variável de ambiente nova no código sem estar aqui.
> Precedência: flag > variável de ambiente > arquivo de config > default.

## 1. Seleção de modelo

| Variável | Tipo | Default | Uso |
|---|---|---|---|
| `DCODE_MODEL` | string | — (obrigatório) | Identificador do modelo. O prefixo resolve a família no `Registry`. Modelo desconhecido falha na criação da sessão, com a lista de prefixos suportados no erro — nunca cai em adaptador genérico. |
| `DCODE_MODEL_WINDOW` | inteiro | `0` | Sobrescreve a janela informada pelo adaptador. `0` usa o valor do adaptador. Existe para modelo novo cuja janela o adaptador ainda não conhece; usar isso é sinal de que o adaptador precisa ser atualizado. |
| `DCODE_MAX_OUTPUT_TOKENS` | inteiro | `0` | Teto de tokens de saída por chamada ao modelo. `0` usa o default do provedor. Não confundir com o teto por turno, que vive no loop do agente. |

## 2. Credenciais

| Variável | Tipo | Default | Uso |
|---|---|---|---|
| `DCODE_API_KEY` | string | — | Credencial. Lida uma vez na inicialização. **Nunca** registrada, nunca em mensagem de erro, nunca em evento (RN-6). |
| `DCODE_BASE_URL` | URL | default da família | Endpoint alternativo — proxy corporativo, gateway, servidor compatível. Trocar isto **não** muda a família (RN-1). |

> Cada adaptador também aceita a variável convencional da sua família, se existir. `DCODE_API_KEY` tem precedência sobre ela, para que a configuração do dcode seja sempre explícita e previsível.

## 3. Rede e resiliência

| Variável | Tipo | Default | Uso |
|---|---|---|---|
| `DCODE_REQUEST_TIMEOUT` | duração | `600s` | Timeout total de uma chamada em streaming. Generoso porque turno com raciocínio longo é legítimo; o corte real de sessão travada é a interrupção do usuário, não este valor. |
| `DCODE_CONNECT_TIMEOUT` | duração | `10s` | Timeout de estabelecimento de conexão. Curto: falha de conexão deve aparecer rápido. |
| `DCODE_MAX_RETRIES` | inteiro | `3` | Tentativas para classe de erro com `Retryable = true`. Não se aplica a `auth`, `quota` nem `bad_request`. |
| `DCODE_RETRY_BASE_DELAY` | duração | `1s` | Base do recuo exponencial. `rate_limit` ignora e usa `RetryAfter` do provedor. |
| `DCODE_RETRY_MAX_DELAY` | duração | `30s` | Teto de cada espera individual do recuo. |

## 4. Medição de contratos comportamentais

> **Dono único destas três variáveis em todo o projeto.** Elas são transversais: valem para os contratos comportamentais desta spec (seção 6 do `.p`) **e** para os da spec do loop do agente (`202608072335-agent-loop`, seção 7 do `.p`). Nenhuma outra `.config.spec.md` as redeclara.

Registra contra o que os limiares foram medidos. Trocar qualquer valor aqui **invalida os limiares de todas as specs que dependem deles** e exige nova medição — não é ajuste de ambiente, é mudança de premissa.

| Variável | Tipo | Default | Uso |
|---|---|---|---|
| `DCODE_EVAL_MODEL` | string | — | Modelo usado na medição. Registrado junto do resultado. |
| `DCODE_EVAL_RUNS` | inteiro | `20` | Execuções por cenário. Abaixo de `20` o intervalo de confiança fica largo demais para um limiar de 95%. |
| `DCODE_EVAL_ENABLED` | booleano | `false` | Eval depende de modelo real e custa dinheiro: fica atrás de build tag e desta chave. Nunca liga na suíte padrão (RN-4). |

## 5. Constantes não configuráveis

| Constante | Valor | Motivo |
|---|---|---|
| Streaming | sempre ligado | RN-3; sem streaming não há interrupção no meio do turno. |
| Validação de tool call contra schema | sempre | RN-8; executar tool call adivinhada corrompe arquivo. |
| Filtro de nome de ferramenta fora do conjunto declarado | sempre | Sustenta o limiar de 100% em `no-phantom-tool`. |

## 6. Changelog

_Sem alterações desde a criação._
