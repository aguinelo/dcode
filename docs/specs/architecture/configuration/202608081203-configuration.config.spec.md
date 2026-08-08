# Config: Configuração e Descoberta de Arquivos

> Nenhuma variável de ambiente nova no código sem estar aqui.
> **Esta spec é a dona da cadeia de precedência** que todas as outras `.config.spec.md` citam no cabeçalho: config travada por administrador > flag > variável de ambiente > config do projeto > config do usuário > default.

## 1. Raízes de diretório

| Variável | Tipo | Default | Uso |
|---|---|---|---|
| `DCODE_HOME` | caminho | vazio | Colapsa as quatro raízes sob um único diretório (RN-1). Vazio mantém a separação XDG, que é o default. Existe para quem prefere um diretório só e aceita ter log junto da config. |
| `DCODE_CONFIG_DIR` | caminho | `$XDG_CONFIG_HOME/dcode`, ou `~/.config/dcode`; no macOS `~/Library/Application Support/dcode` | `config.toml`, instruções globais, `skills/`, `commands/`. É a raiz feita para ser versionada. |
| `DCODE_DATA_DIR` | caminho | `$XDG_DATA_HOME/dcode`, ou `~/.local/share/dcode` | Artefatos de longa vida criados pelo usuário. |
| `DCODE_CACHE_DIR` | caminho | `$XDG_CACHE_HOME/dcode`, ou `~/.cache/dcode` | Descartável sem perda: consulta de versão, temporários. |

> `DCODE_STATE_DIR` — a quarta raiz — é declarada em `202608072240-client-server-protocol.config.spec.md`, seção 1, porque o socket e o log de sessão nascem lá. Esta spec define seu **papel** no layout; aquela, seu valor.

## 2. Arquivo de config

| Variável | Tipo | Default | Uso |
|---|---|---|---|
| `DCODE_CONFIG_FILE` | caminho | `$DCODE_CONFIG_DIR/config.toml` | Sobrescreve o caminho do arquivo do usuário. Arquivo inexistente **não** é erro — config é opcional por inteiro. |
| `DCODE_PROJECT_CONFIG` | caminho | `<workspace>/.dcode/config.toml` | Config do projeto. Vence a do usuário, perde para ambiente e flag. |
| `DCODE_CONFIG_STRICT` | booleano | `true` | Chave desconhecida é erro. Desligar transforma em aviso — só para migrar entre versões, nunca como estado permanente: erro de digitação silenciosamente ignorado é a classe de bug mais frustrante que existe. |

## 3. Descoberta de instruções

Fornece os defaults que `202608080016-behavior-definition.config.spec.md` deixou em aberto.

| Variável | Tipo | Default | Uso |
|---|---|---|---|
| `DCODE_INSTRUCTION_FILES` | lista | `AGENTS.md,DCODE.md` | Nomes procurados em cada nível, **na ordem**: o último tem maior precedência no mesmo diretório. Reordenar muda quem vence; remover `AGENTS.md` abre mão da compatibilidade entre ferramentas (RN-4). |
| `DCODE_SKILLS_DIR` | caminho | `$DCODE_CONFIG_DIR/skills` | Raiz de skills do usuário. Skills de projeto ficam em `<workspace>/.dcode/skills`, sempre consideradas. |
| `DCODE_COMMANDS_DIR` | caminho | `$DCODE_CONFIG_DIR/commands` | Raiz de comandos do usuário. Comandos de projeto ficam em `<workspace>/.dcode/commands` e vencem os do usuário no caso de nome igual. |

> `DCODE_INSTRUCTIONS_PATH`, `DCODE_INSTRUCTIONS_MAX_BYTES` e `DCODE_INSTRUCTIONS_MAX_DEPTH` são declaradas em `202608080016-behavior-definition.config.spec.md`, seção 1. Esta spec fornece os **defaults** que lá estavam pendentes.

## 4. Diagnóstico

| Variável | Tipo | Default | Uso |
|---|---|---|---|
| `DCODE_CONFIG_TRACE` | booleano | `false` | Registra, em `debug`, cada chave resolvida com valor, camada e origem. Verboso; use quando um valor efetivo não bate com o esperado. |

> O caminho normal para isso é `dcode config get <chave>`, que responde valor **e** procedência (RN-8). O `TRACE` é para quando se quer a resolução inteira de uma vez.

## 5. Constantes não configuráveis

| Constante | Valor | Motivo |
|---|---|---|
| Cadeia de precedência | RN-7 | Ordem única para todo o produto; caso especial por chave tornaria config inexplicável. |
| Recusa de credencial em `config.toml` | sempre | RN-3; arquivo de config é feito para ser versionado e sincronizado. |
| Descoberta não sobe acima da raiz do workspace | sempre | Instrução fora do workspace exige caminho explícito, nunca descoberta acidental. |
| Cadeia congelada na criação da sessão | sempre | RN-5; instrução tardia invalidaria o prefixo inteiro. |
| Instrução fora da cadeia vira lembrete | sempre | RN-6; a única saída que não ignora o usuário nem quebra a imutabilidade. |
| Comando não executa nada | sempre | RN-10; execução fora do avaliador de política está proibida pela spec de sandbox. |
| Permissão `0700` nas raízes criadas | sempre | Config e estado contêm instruções e histórico do usuário. |
| Travamento é visível | sempre | RN-9; ignorar em silêncio faz o usuário achar que a mudança funcionou. |

## 6. Changelog

_Sem alterações desde a criação._
