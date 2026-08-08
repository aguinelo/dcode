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
- [ ] Instrução que afrouxa segurança é descartada e registrada em `warn`.

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

### Passo 7 — `DCODE_DOCTRINE_DUMP`

`cmd/dcode/dump.go`

- [ ] Imprime o prompt montado e sai.
- [ ] Saída idêntica ao que seria enviado ao modelo — sem reformatação de conveniência.

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
- **Caminho literal de arquivo de instrução** — acopla este pacote ao sistema de arquivos e destrói a pureza de `Build`. Use o localizador injetado.

## Changelog

_Sem alterações desde a criação._
