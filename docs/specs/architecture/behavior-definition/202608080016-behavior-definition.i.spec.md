# Implementing: Definição de Comportamento

> Siga a ordem. Se algum passo contradisser o `.r.spec.md`, **pare** — o `.r` tem precedência.
> Todo código novo nasce de teste que falhou primeiro (`docs/conventions/TESTING.pt-BR.md`).

## Pré-requisitos

| Componente | Estado mínimo |
|---|---|
| `202608072333-context-engine` | Passo 3 — `Assemble` consome a saída de `Build` |
| `202608072334-provider-adapter` | Passo 1 — a interface `Family` fornece a formulação |
| `202608072337-tool-suite` | Passo 1 — `ToolDef` entra no prefixo |

## Dependência resolvida

Os Passos 3 e 6 dependem de `202608081203-configuration` (Passo 3 daquela spec), que define caminhos, nomes de arquivo e hierarquia de descoberta.

O localizador continua sendo **injetado por interface**, não caminho literal — agora por desenho, não por bloqueio: é o que mantém este pacote puro e testável sem tocar o sistema de arquivos.

## Ordem de execução

### Passo 1 — Tipos e doutrina

`internal/behavior/types.go`

- [ ] `Doctrine`, `Instruction`, `InstructionSource`, `SkillIndexEntry`, `Prompt`.
- [ ] Texto inicial da doutrina base, com `Safety` numa seção própria e isolada.

**Teste obrigatório:** `Doctrine.Safety` não é alcançável por nenhum caminho de configuração — varredura de todos os pontos que montam `Doctrine`.

> `Safety` isolado desde o primeiro commit. Depois que texto de segurança se mistura ao resto, separar vira arqueologia.

### Passo 2 — `Build`, puro

`internal/behavior/build.go`

- [ ] Ordem dos blocos exatamente conforme a seção 2 do `.p`.
- [ ] Bloco ausente é omitido por inteiro, sem marcador vazio.
- [ ] Recebe `Family` para a formulação; o conjunto de regras vem de `Prompt`.

**Testes obrigatórios:**
- Pureza byte-a-byte para a mesma entrada.
- Guarda: o pacote não importa `os`, `net`, `time` nem `math/rand`.
- Varredura da saída por timestamp, contador, ID de sessão e caminho absoluto variável.
- Golden file por combinação de blocos presentes e ausentes.
- **Duas famílias produzem prompts distintos a partir do mesmo `Prompt`, e ambos contêm todas as regras de `Safety`** — é o teste que garante que RN-8 não vira porta para regra divergente.

### Passo 3 — Descoberta e precedência de instruções

`internal/behavior/instructions.go`

- [ ] Localizador **injetado por interface**, implementado por `internal/config`.
- [ ] Resolução de precedência conforme a seção 4 do `.p`.
- [ ] Empilhamento, não substituição: menor autoridade primeiro.
- [ ] Truncamento em `InstructionsMaxBytes` **com aviso**.
- [x] Instrução que tenta afrouxar segurança é **registrada**. Não é descartada: o resto do arquivo é legítimo, e jogar fora um arquivo inteiro por uma frase é a falha de filtro silencioso que este projeto recusa em todo lugar. A garantia é estrutural e está noutro lugar — o sandbox é aplicado pelo sistema operacional, aprovação é consentimento, e `Safety` não é campo da sobreposição. O que faltava era a tentativa ser **visível**..

**Testes obrigatórios:**
- Uma asserção por par de fontes conflitantes da tabela de precedência.
- Diretório vence projeto; projeto vence usuário; travada vence tudo.
- Instrução mandando ignorar aprovação é descartada, e o registro acontece.
- Truncamento produz aviso; nunca silencioso.

> Instrução truncada em silêncio é pior que instrução ausente: o usuário acredita que a regra está valendo.

### Passo 4 — Fixação do prefixo

`internal/behavior/session.go`

- [ ] Prefixo resolvido na criação da sessão e congelado (RN-5).
- [ ] Nenhum caminho de código permite adicionar instrução depois.

**Teste obrigatório:** tentativa de mutar instruções após a criação retorna erro; o prefixo previamente produzido permanece byte-a-byte idêntico.

### Passo 5 — Canal de lembrete

`internal/behavior/reminder.go`

- [ ] `Reminder`, `ReminderKind` e `Emit` puro.
- [ ] Texto constante por `Kind`, interpolando só dados já presentes no histórico.
- [ ] Sempre anexado; nenhum caminho o coloca no prefixo.
- [ ] Doutrina base descreve o canal ao modelo.

**Testes obrigatórios:**
- `Emit` é puro: mesmo estado, mesmo conjunto.
- Nenhum lembrete aparece na saída de `Build` — varredura.
- Emissões repetidas do mesmo `Kind` com os mesmos dados produzem texto idêntico.
- Nenhum texto de lembrete contém horário ou contador.

> Migrar aqui a nota de execução concorrente da seção 4.3 de `202608072335-agent-loop.p.spec.md`. Ela é lembrete, não texto solto no resultado da ferramenta — e centralizar impede que uma segunda formulação apareça mais tarde.

### Passo 5.5 — Estado de verificação

> Acrescentado por [202608102000](changelog/202608102000-verificacao-antes-da-afirmacao.md). Depende do Passo 5; independente dos Passos 3, 6 e 7.

`internal/behavior/verification.go`

- [ ] `Verification` e os três `ReminderKind` novos.
- [ ] Derivação **pura** a partir do registro de escrita e do registro de execução da sessão. Sem julgamento, sem heurística sobre o texto do modelo.
- [ ] Uma frase acrescentada a `Doctrine.Style` — e só ela; nenhuma outra seção muda.

**Testes obrigatórios:**
- Edição sem execução posterior → `stale`.
- Execução do comando após a última edição, saída zero → `passed`.
- Execução com saída diferente de zero → `failed`, mesmo tendo rodado.
- Sessão só de leitura → `clean`, e **nenhum** lembrete de verificação emitido.
- Mudança sem comando configurado → `unavailable`.
- Nenhum lembrete de verificação aparece no prefixo — varredura de `Build`.

`internal/loop` — a condição do passo 4 da RN-1 de `agent-loop`:

- [ ] Sem tool call, com `stale` ou `failed`, e ainda não forçado neste turno → anexa lembrete e volta ao passo 2.
- [ ] Forçado **uma vez por turno**; persistindo, encerra com `StopUnverified`.
- [ ] `unavailable` **não** força continuação.

**Testes obrigatórios:**
- A continuação dispara exatamente uma vez, nunca duas — asserção contra o laço patológico.
- `StopUnverified` não é tratado como erro em nenhum caminho.
- `unavailable` encerra sem forçar.

`internal/tui` — o selo:

- [ ] Estado de verificação exibido ao fim do turno.

> O selo é **a garantia**. A frase da doutrina e os contratos comportamentais são reforço: o texto do modelo pode afirmar sucesso, e o selo o contradiz. Se o selo não for exibido, esta mudança inteira volta a depender de o modelo obedecer.

### Passo 6 — Índice de skills

`internal/behavior/skills.go`

- [ ] Índice com **uma linha** por skill; nenhum corpo no prefixo.
- [ ] Teto de `SkillsMaxIndex`.
- [ ] Corpo anexado quando o gatilho bate.
- [ ] Descoberta pelo mesmo localizador injetado do Passo 3.

**Testes obrigatórios:**
- Nenhum corpo de skill aparece na saída de `Build`.
- Índice acima do teto trunca e avisa.
- Carregar corpo não altera o prefixo.

### Passo 6.5 — Sobreposição de doutrina

> Acrescentado por [202608101800](changelog/202608101800-doutrina-editavel-por-camada.md). Depende do Passo 1 e do Passo 2; independente dos Passos 3, 5 e 6.

`internal/behavior/doctrine_overlay.go`

- [x] `DoctrineOverlay` com exatamente três campos: `Identity`, `Style`, `ToolsMore`. **`Safety` não é campo** — é a trava (RN-12).
- [x] `Doctrine.Apply(o DoctrineOverlay) Doctrine`, pura. `ToolsMore` concatena; nunca substitui.
- [x] `LoadDoctrineOverlay(dir string, maxBytes int)` — **um** diretório, nunca uma lista (RN-11).
- [x] `Notice` para: truncamento por teto, nome de arquivo não reconhecido, `safety.md` presente.
- [x] `SectionOrigins` e `DoctrineOverlay.Origins()`. `Safety` é `OriginBuiltin` sem passar por condicional: não há campo que o mude.

**Testes obrigatórios**, um por invariante da seção 8 do `.p`:

- [x] `Apply(o).Safety == DefaultDoctrine().Safety` para toda entrada, incluindo a tabela de casos hostis.
- [x] `Apply(o).ToolPolicy` tem `DefaultDoctrine().ToolPolicy` como **prefixo**, sempre.
- [x] Os três arquivos em `<workspace>/.dcode/doctrine/` produzem prompt **byte-idêntico** ao default.
- [x] `safety.md` na raiz do usuário não muda nada **e** produz `Notice`.
- [x] Truncamento produz `Notice`; nenhum caminho trunca em silêncio.
- [x] Ordem dos `Notice` estável entre execuções — lista de aviso que embaralha é lista que ninguém consegue diffar.

`internal/app/app.go`

- [x] Resolver a sobreposição junto de instruções e skills, **uma vez**, na criação da sessão (RN-5).
- [x] Passar **apenas** `roots.Config` — a raiz do usuário. Contraste deliberado com `LoadSkills`, logo acima, que recebe duas raízes.
- [x] A decisão do diretório sai numa função própria, `doctrineDir(override string, roots config.Roots)`. O workspace **não é parâmetro dela**: não há ramo que possa alcançá-lo porque não há argumento. Mesma forma de trava da RN-12, um nível acima.

> A tentação aqui é reaproveitar a lista de raízes que já está montada duas linhas acima, para skills. Fazer isso abre exatamente o vetor que a RN-11 fecha, e o teste de workspace acima existe para pegá-lo.
### Passo 7 — `DCODE_DOCTRINE_DUMP`

`cmd/dcode/dump.go`

- [ ] Imprime o prompt montado e sai.
- [ ] Saída idêntica ao que seria enviado ao modelo — sem reformatação de conveniência.
- [ ] Marca a `Origin` de cada uma das quatro seções; `Safety` é sempre `builtin`.
- [ ] Lista os `Notice` acumulados na resolução da sobreposição.

> É a ferramenta de auditoria do produto. Um harness que não deixa inspecionar o próprio prompt pede confiança cega em um programa com acesso a shell. Também é o que torna depurável qualquer contrato comportamental que falhe.

### Passo 8 — Contratos comportamentais

`internal/behavior/evals/` — atrás de build tag `eval`.

- [ ] Fixture para os sete cenários da seção 7 do `.p`.
- [ ] Registro de modelo e versão junto do resultado.
- [ ] Fora da suíte padrão e da CI de PR.

> `safety-not-overridable` merece atenção especial: se ele falhar, o achado **não** é "ajustar o prompt". A garantia real é estrutural — a política do sandbox não consulta o prompt. Falha aqui significa que a doutrina está confundindo o modelo, não que a fronteira está fraca. Confirme a fronteira antes de mexer no texto.

### Passo 9 — Invariantes

`internal/behavior/invariants_test.go`

- [ ] Um teste por linha da seção 8 do `.p.spec.md`.
- [ ] `go test -race ./internal/behavior/...` limpo.
- [ ] Cobertura ≥ 90%, excluído o pacote de eval.

## Ordem de dependência

```
Passo 1 (tipos e doutrina)
  └─ Passo 2 (Build, puro)
       ├─ Passo 3 (instruções)      → internal/config, Passo 3
       │    └─ Passo 4 (fixação do prefixo)
       ├─ Passo 5 (lembretes)
       ├─ Passo 6.5 (sobreposição)  → internal/config, Passo 3
       └─ Passo 6 (skills)          → internal/config, Passo 3
            └─ Passo 7 (dump)
                 ├─ Passo 8 (contratos comportamentais)
                 └─ Passo 9 (invariantes)
```

## Armadilhas conhecidas

- **Regra nova indo direto para a doutrina base** — é o caminho de menor resistência e o mais caro. Passe pela árvore de decisão da seção 6 do `.p` antes; chegar na doutrina é último recurso.
- **Mensagem de erro de ferramenta escrita como diagnóstico** — perde a camada mais eficiente da pilha. Erro é onde a recuperação é ensinada.
- **Lembrete no prefixo** — invalida o cache a cada emissão e não quebra teste nenhum sem a varredura do Passo 5.
- **Formulação de família virando regra de família** — começa como ajuste de fraseado e termina com duas famílias se comportando diferente. O teste do Passo 2 existe para pegar isso.
- **Corpo de skill no índice** — cresce sem ninguém notar até o prefixo dobrar de tamanho.
- **Instrução do usuário afrouxando segurança sem registro** — o descarte silencioso esconde tentativa que deveria ser visível.
- **Estado de verificação inferido do texto do modelo** — "ele disse que rodou" não é fato. A derivação é do registro de execução, ou o selo não vale nada.
- **Continuação forçada sem teto** — projeto cuja verificação nunca passa gira até o teto de iterações, e o usuário paga por isso sem entender.
- **`StopUnverified` tratado como erro** — é estado honesto de trabalho entregue sem conferência. Virando erro, a saída fácil passa a ser desligar a checagem.
- **Verificação disparando em tarefa de leitura** — queima turno respondendo "o que essa função faz", e duas semanas assim é uma ferramenta desinstalada.
- **Caminho literal de arquivo de instrução** — acopla este pacote ao sistema de arquivos e destrói a pureza de `Build`. Use o localizador injetado.
- **`Safety` protegido por condicional em vez de por tipo** — um `if` se remove num refactor sem quebrar compilação, e o teste que o cobria pode ser removido junto. A ausência do campo em `DoctrineOverlay` é o que não compila.
- **Sobreposição lendo a mesma lista de raízes das skills** — está duas linhas acima no mesmo arquivo e é a colagem mais provável. Abre o vetor da RN-11 inteiro.
- **`ToolsMore` substituindo em vez de acrescentar** — passa despercebido enquanto o arquivo do usuário for parecido com o texto embarcado, e só aparece como alucinação de ferramenta muito depois.
- **Truncar sobreposição em silêncio** — o usuário acredita que a regra está valendo. É o mesmo defeito já previsto para arquivo de instrução, e o teto menor aqui torna-o mais provável.

## Changelog

- [202608101800 — Doutrina editável por camada](changelog/202608101800-doutrina-editavel-por-camada.md)
- [202608102000 — Verificação antes da afirmação](changelog/202608102000-verificacao-antes-da-afirmacao.md)
