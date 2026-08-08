# Config: Definição de Comportamento

> Nenhuma variável de ambiente nova no código sem estar aqui.
> Precedência: config travada por administrador > flag > variável de ambiente > arquivo de config > default.

> Os **caminhos concretos** — raiz de configuração, nomes de arquivo, hierarquia de descoberta — são definidos em `202608081203-configuration`. Aqui ficam apenas as chaves que controlam **como o comportamento é montado**, não onde os arquivos moram.

## 1. Instruções

| Variável | Tipo | Default | Uso |
|---|---|---|---|
| `DCODE_INSTRUCTIONS_ENABLED` | booleano | `true` | Liga a leitura de arquivos de instrução. Desligar roda só com a doutrina base — útil para isolar se um comportamento vem da instrução do usuário ou do produto. |
| `DCODE_INSTRUCTIONS_MAX_BYTES` | inteiro | `65536` | Teto por arquivo de instrução. Excedido, trunca e **avisa** — instrução truncada em silêncio faz o usuário achar que a regra está valendo quando não está. |
| `DCODE_INSTRUCTIONS_MAX_DEPTH` | inteiro | `8` | Níveis de diretório percorridos acima do workspace na descoberta. Limita o custo em monorepo profundo. |
| `DCODE_INSTRUCTIONS_PATH` | caminho | vazio | Sobrescreve a descoberta e usa um arquivo específico. Vazio usa a descoberta definida em `202608081203-configuration.p.spec.md`, seção 4.1. |

## 2. Skills

| Variável | Tipo | Default | Uso |
|---|---|---|---|
| `DCODE_SKILLS_ENABLED` | booleano | `true` | Liga o disclosure progressivo. Desligar remove o índice do prefixo e nenhum corpo é carregado. |
| `DCODE_SKILLS_MAX_INDEX` | inteiro | `64` | Máximo de skills no índice. Cada entrada custa uma linha no prefixo de **todo turno**; acima disso o índice compete por atenção com a tarefa. |

> `DCODE_SKILLS_DIR` é declarada em `202608081203-configuration.config.spec.md`, seção 3, junto das demais raízes de descoberta.

## 3. Lembretes

| Variável | Tipo | Default | Uso |
|---|---|---|---|
| `DCODE_REMINDERS_ENABLED` | booleano | `true` | Liga o canal de lembrete. Desligar é apenas para depuração: sem ele o agente age sobre estado obsoleto e não sabe disso. |
| `DCODE_REMINDER_KINDS` | lista | todos | Restringe os tipos emitidos. Existe para isolar qual lembrete causa um comportamento observado. |

## 4. Doutrina

| Variável | Tipo | Default | Uso |
|---|---|---|---|
| `DCODE_DOCTRINE_STYLE` | string | vazio | Ajusta a seção de estilo da doutrina — tom e formato de saída. **Não** alcança `Doctrine.Safety` (RN-10). |
| `DCODE_DOCTRINE_DUMP` | booleano | `false` | Imprime o prompt montado e sai. Ferramenta de depuração e de auditoria: é como se verifica o que de fato vai ao modelo. |

> `DCODE_DOCTRINE_DUMP` é também a resposta à pergunta "o que exatamente está indo para a LLM". Um harness que não permite inspecionar o próprio prompt pede confiança cega em um programa com acesso a shell.

## 5. Constantes não configuráveis

| Constante | Valor | Motivo |
|---|---|---|
| `Doctrine.Safety` sobrescrevível | **nunca** | RN-10; a fronteira real é estrutural, e o prompt não pode sugerir o contrário. |
| Ordem dos blocos do prefixo | seção 2 do `.p` | Alterar invalida o cache de toda sessão viva; é mudança de contrato. |
| Lembrete anexado, nunca prefixado | sempre | RN-6; prefixar invalidaria o cache a cada emissão. |
| Corpo de skill fora do prefixo | sempre | RN-7; carregar tudo é o caminho para prompt de dezenas de milhares de tokens pago em todo turno. |
| Prefixo montado uma vez por sessão | sempre | RN-5; instrução tardia invalidaria o prefixo inteiro. |
| Precedência da seção 4 do `.p` | sempre | Instrução mais específica vence; travada vence tudo. |

## 6. Changelog

_Sem alterações desde a criação._
