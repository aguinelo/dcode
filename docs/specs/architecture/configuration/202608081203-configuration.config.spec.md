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

## 3. Descoberta de instruções

Fornece os defaults que `202608080016-behavior-definition.config.spec.md` deixou em aberto.

| Variável | Tipo | Default | Uso |
|---|---|---|---|

> `DCODE_INSTRUCTIONS_PATH`, `DCODE_INSTRUCTIONS_MAX_BYTES` e `DCODE_INSTRUCTIONS_MAX_DEPTH` são declaradas em `202608080016-behavior-definition.config.spec.md`, seção 1. Esta spec fornece os **defaults** que lá estavam pendentes.

| `DCODE_CREDENTIAL_BACKEND` | enum | vazio | Escolhe o armazenamento da credencial: chaveiro do sistema ou arquivo `0600`. Vazio escolhe pelo que existe. Configuração e não flag de comando: uma flag no comando que **escreve**, e nada nos que **leem**, guarda o segredo onde nada procura. |
## 3.1 Tradução e reindex

| Variável | Tipo | Default | Uso |
|---|---|---|---|
| `DCODE_INSTRUCTION_NOTICE` | booleano | `true` | Avisa, no início da sessão, quando há instrução compartilhada não traduzida ou origem divergente (RN-6.2). Desligar silencia o aviso; **não** muda o que é lido. |
| `DCODE_INSTRUCTION_FOREIGN` | lista | `AGENTS.md` | Arquivos tratados como **formato compartilhado** — candidatos a tradução, e origem do digest do reindex. `DCODE.md` nunca entra: é o destino, não a origem. |

> Não há chave para desligar a verificação de ferramenta nem a de comando. Gerar `DCODE.md` sem conferir produziria exatamente o arquivo que esta mudança existe para evitar, com a aparência de ter sido conferido.

## 4. Diagnóstico

| Variável | Tipo | Default | Uso |
|---|---|---|---|

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
| Comando citado em arquivo de instrução **nunca é executado para verificação** | sempre | RN-6.1; o arquivo pode ter vindo de repositório clonado, e sondar por presença de arquivo responde a mesma pergunta sem rodar nada de terceiro. |
| Descarte na tradução é registrado no arquivo gerado | sempre | RN-6.1; sem o registro ninguém distingue descarte correto de descarte de regra legítima. |
| Reindex nunca sobrescreve o arquivo gerado | sempre | RN-6.2; ele pertence ao usuário a partir da geração, e sobrescrever apaga edição manual sem aviso. |
| Instrução não traduzida avisa, nunca bloqueia | sempre | RN-6.2; portão que trava vira portão atravessado no automático. |
| Permissão `0700` nas raízes criadas | sempre | Config e estado contêm instruções e histórico do usuário. |
| Travamento é visível | sempre | RN-9; ignorar em silêncio faz o usuário achar que a mudança funcionou. |

## 6. Changelog

- [202608101900 — Tradução de instruções de terceiros](changelog/202608101900-traducao-de-instrucoes-de-terceiros.md)
