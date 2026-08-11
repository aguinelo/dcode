# Config: Definição de Comportamento

> Nenhuma variável de ambiente nova no código sem estar aqui.
> Precedência: config travada por administrador > flag > variável de ambiente > arquivo de config > default.

> Os **caminhos concretos** — raiz de configuração, nomes de arquivo, hierarquia de descoberta — são definidos em `202608081203-configuration`. Aqui ficam apenas as chaves que controlam **como o comportamento é montado**, não onde os arquivos moram.

## 1. Instruções

| Variável | Tipo | Default | Uso |
|---|---|---|---|
| `DCODE_BEHAVIOR_INSTRUCTIONS_ENABLED` | booleano | `true` | Liga a leitura de `AGENTS.md` e `DCODE.md`. Desligar roda só com a doutrina embarcada — é como se isola se um comportamento vem da instrução do usuário ou do produto. |

> O prefixo `BEHAVIOR_` casa com a seção `behavior.*` do `config.toml`, e a bijeção entre chave e variável é asserção de teste. A spec declarava `DCODE_INSTRUCTIONS_ENABLED` e o código sempre implementou este nome; o nome documentado é que estava errado. Mesmo precedente de `sandbox.policy` → `sandbox.approval_policy`: o nome interno passou a ser o que o usuário escreve, não o contrário.

## 2. Skills

| Variável | Tipo | Default | Uso |
|---|---|---|---|
| `DCODE_BEHAVIOR_SKILLS_ENABLED` | booleano | `true` | Liga a divulgação progressiva. Desligar tira o índice do prefixo, e nenhum corpo de skill é carregado. |

> `DCODE_SKILLS_DIR` **não existe**. O diretório de skills é derivado da raiz de configuração; a chave foi removida em `202608110900` por nunca ter sido lida.


## 3. Lembretes

| Variável | Tipo | Default | Uso |
|---|---|---|---|
| `DCODE_BEHAVIOR_REMINDERS_ENABLED` | booleano | `true` | Liga o canal anexado. Desligar remove todo lembrete — inclusive o de arquivo alterado em disco, que é o que impede editar a partir de conteúdo que o modelo não tem mais. |
| `DCODE_SHOW_REASONING` | booleano | `true` | Encaminha o raciocínio do modelo aos clientes. Ele **nunca** entra no histórico: é evento de exibição, não contexto. |

## 3.1 Verificação (RN-13)

| Variável | Tipo | Default | Uso |
|---|---|---|---|
| `DCODE_VERIFY_COMMAND` | string | vazio | O comando que **conta** como verificação. Explícito porque "rodou algum bash" contaria um `ls`. Vazio produz estado `unavailable` quando há mudança — o produto diz que não sabe conferir, em vez de fingir que conferiu. |

> **`DCODE_VERIFY_TIMEOUT` e `DCODE_VERIFY_FORCE_TURN` foram removidas**, e pelo motivo que o próprio `202608102100` dá: *"os dois mecanismos não coexistem"*. A verificação é a lista unitária da definição de pronto, então o teto por critério é `DCODE_DONE_TIMEOUT` e a chave que liga a reentrada é `DCODE_DONE_ENABLED`. Manter as duas seria manter duas formas de ajustar a mesma coisa — exatamente o que a remoção de `DCODE_DOCTRINE_STYLE` evitou.
>
> `DCODE_VERIFY_COMMAND` **fica**: ela não duplica nada. É o que nomeia o critério único, e é o valor que o `/init` propõe a partir de comando sondado.

> O comando vem da config ou do arquivo de instrução **específico**, revisado por uma pessoa. **Nunca** de formato compartilhado de terceiro: ele passaria a ser executado a cada turno, que é a RN-6.1 de `202608081203-configuration` violada em laço.

> `/init` propõe o valor de `DCODE_VERIFY_COMMAND` a partir de comando **sondado** — ver `202608101900`. A tradução descobre como se verifica; a RN-13 cobra o uso.

## 4. Doutrina

| Variável | Tipo | Default | Uso |
|---|---|---|---|
| `DCODE_DOCTRINE_ENABLED` | booleano | `true` | Liga a leitura da sobreposição de doutrina. Desligar roda só com a doutrina embarcada — é como se isola se um comportamento vem da sobreposição do usuário ou do produto. |
| `DCODE_DOCTRINE_DIR` | caminho | vazio | Sobrescreve o diretório da sobreposição. Vazio usa `doctrine/` sob a raiz de configuração do **usuário**, definida em `202608081203-configuration`. Nunca a raiz do workspace (RN-11). |
| `DCODE_DOCTRINE_MAX_BYTES` | inteiro | `16384` | Teto por arquivo de sobreposição. Excedido, trunca e **avisa**. Menor que o teto de instrução porque isto é camada base, paga em todo turno de toda sessão. |
| `DCODE_DOCTRINE_DUMP` | booleano | `false` | Imprime o prompt montado, com a origem de cada seção, e sai. Ferramenta de depuração e de auditoria: é como se verifica o que de fato vai ao modelo. |

Os arquivos da sobreposição, e o efeito de cada um, estão na seção 3.1 do `.p`. O conteúdo é documento de várias linhas, não valor de configuração — daí serem arquivos, e não chaves.

> `DCODE_DOCTRINE_STYLE` **foi removida**. Estava declarada aqui desde a criação da spec e nunca foi implementada. Um bloco de prosa não é valor de variável de ambiente, e manter duas formas de ajustar a mesma seção cria a dúvida de qual venceu. Ver o changelog de 202608101800.

> `DCODE_DOCTRINE_DUMP` é também a resposta à pergunta "o que exatamente está indo para a LLM". Um harness que não permite inspecionar o próprio prompt pede confiança cega em um programa com acesso a shell. Com sobreposição existindo, ele passa a ser obrigatório e não conveniente: substituição invisível seria pior que a imutabilidade que substitui (RN-12).

## 5. Constantes não configuráveis

| Constante | Valor | Motivo |
|---|---|---|
| `Doctrine.Safety` sobrescrevível | **nunca** | RN-10; a fronteira real é estrutural, e o prompt não pode sugerir o contrário. Garantido por ausência de campo em `DoctrineOverlay`, não por condicional (RN-12). |
| `Doctrine.Safety` acrescentável | **nunca** | RN-12; apêndice pode dizer "ignore o acima", logo aceitar apêndice é não ter trava. |
| `Doctrine.ToolPolicy` substituível | **nunca** | RN-12; descreve máquina que existe. Declarar ferramenta inexistente faz o modelo chamar o que não há, e a falha aparece longe da causa. |
| Raiz do workspace como origem de doutrina | **nunca** | RN-11; é o vetor da RN-10 por outra porta — um repositório clonado redefiniria quem o agente pensa que é. |
| Ordem dos blocos do prefixo | seção 2 do `.p` | Alterar invalida o cache de toda sessão viva; é mudança de contrato. |
| Lembrete anexado, nunca prefixado | sempre | RN-6; prefixar invalidaria o cache a cada emissão. |
| Corpo de skill fora do prefixo | sempre | RN-7; carregar tudo é o caminho para prompt de dezenas de milhares de tokens pago em todo turno. |
| Prefixo montado uma vez por sessão | sempre | RN-5; instrução tardia invalidaria o prefixo inteiro. |
| Precedência da seção 4 do `.p` | sempre | Instrução mais específica vence; travada vence tudo. |
| Estado de verificação derivado de fato | sempre | RN-13; se dependesse de julgamento, o selo do cliente não valeria nada. |
| Continuação forçada limitada por ciclos sem progresso | sempre | RN-13; sem teto, projeto cuja verificação não roda gira até o teto de iterações. |
| Comando de verificação vindo de formato compartilhado | **nunca** | RN-13; seria execução de instrução de terceiro a cada turno. |

## 6. Changelog

- [202608101800 — Doutrina editável por camada](changelog/202608101800-doutrina-editavel-por-camada.md)
- [202608102000 — Verificação antes da afirmação](changelog/202608102000-verificacao-antes-da-afirmacao.md)
