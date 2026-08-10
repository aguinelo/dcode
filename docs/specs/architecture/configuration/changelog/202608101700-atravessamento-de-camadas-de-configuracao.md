# Atravessamento de camadas de configuração

**Data:** 2026-08-10
**Specs afetadas:** `202608081203-configuration` (`.p`)

## O que mudou

Um novo invariante na seção 7 da spec: **toda chave em `KnownKeys` que tenha
campo correspondente em `app.Options` é lida por `FromEnv` via um acessor
`r.{Bool,String,Int}` para essa chave, e toda atribuição de
`Options.<campo>` que precise chegar a `loop.Config` é feita na construção
do engine em `app.New` — sem lacuna no atravessamento das camadas.**

A verificação está em `internal/app/wiring_test.go`. A suíte é declarativa
— uma tabela em código lista cada elo da cadeia (`Options.<campo>`,
`KnownKeys.<chave>`, acessor, `loop.Config.<campo>` quando houver), e os
testes estruturais e de runtime percorrem a tabela para confirmar que cada
elo existe em código.

## Por que mudou

A suíte inteira passava e `behavior.show_reasoning` simplesmente não
funcionava: a chave existia no esquema, o campo existia em `Options`,
nenhum teste atravessava as três camadas, e o sintoma só apareceu na
verificação contra o modelo real. O mesmo formato de defeito é invisível
para qualquer teste que examine uma camada de cada vez — e como a
configuração tem três camadas exatas, é a classe de bug mais comum deste
componente.

A spec já dizia, em outra forma, que a configuração tem que se explicar
(RN-8). Não basta o valor estar correto: ele tem que **chegar** ao código
que o consome. Esta mudança fecha o furo entre "existe" e "funciona".

## Como o teste atravessa as camadas

Quatro pernas, cada uma cobrindo um elo da cadeia:

1. **Tabela existe e aponta para chaves reais.** Toda linha da tabela
   aponta para uma chave em `KnownKeys`; nenhuma chave fica sem linha
   entre as que têm campo em `Options`.
2. **O campo em `Options` existe e é exportado.** Reflexão confirma
   `reflect.TypeOf(Options{}).FieldByName(name)`.
3. **`FromEnv` lê a chave via `r.<Accessor>`.** Inspeção do texto-fonte
   do corpo da função, não dos valores em runtime — porque o zero value
   de um `bool` não-fio é indistinguível do valor correto até o ambiente
   sobrescrever. Foi exatamente o que aconteceu com `ShowReasoning`.
4. **`app.New` passa o valor para `loop.Config`.** O mesmo tipo de
   inspeção: a linha `loop.Config{<campo>: opts.<opts>}` tem que estar lá.

A quinta perna é de runtime: setar a variável de ambiente e confirmar via
reflexão que o valor chega em `loop.Config` através do wiring. Pega o caso
em que a linha existe sintaticamente mas referencia o campo errado.

## O que o teste não faz

Não valida o conteúdo semântico das opções — só que cada elo da cadeia
existe. "ShowReasoning tem que ser opt-in" continua sendo uma decisão de
produto, não de teste.

Não cobre chaves em `KnownKeys` que **não** têm campo em `Options` (por
exemplo, `behavior.instructions_enabled` e `behavior.skills_enabled` hoje).
Essas são um furo diferente — chave declarada sem consumidor — e o teste
apropriado para elas vive na cobertura da `KnownKeys` em si, não no
atravessamento das três camadas.

## Impacto

- `internal/app/wiring_test.go` é a nova morada do invariante. Quem tocar
  em `Options`, em `loop.Config` ou em `KnownKeys` precisa atualizar a
  tabela — uma linha por elo, em código.
- A inspeção de fonte vive acoplada ao arquivo `app.go` lido do mesmo
  diretório: se o nome do arquivo mudar, o teste falha com mensagem
  clara, e o conserto é explícito.
- Cobertura do `internal/app/` sobe marginalmente — o teste é
  declarativo, e a maioria das pernas é estática.

## Alternativa descartada

Gerar a tabela via reflection sobre `Options` e inferir o nome da chave
em `KnownKeys` por convenção (`section.field` → `Field`). Descartada
porque o mapeamento é irregular (`Policy` ≠ `sandbox.approval_policy`,
`Rules` ≠ `rules.confirm_write`, `Limits` é campo de sub-struct), e a
heurística esconderia exatamente o tipo de erro que o teste existe para
pegar: alguém renomeia `KnownKeys` sem renomear a tabela.


## Correção de 2026-08-10, ao aplicar

A primeira redação deste invariante dizia *"toda chave que **tenha campo
correspondente** em `app.Options`"*. A condicional desculpa exatamente o caso
que interessa: chave sem campo satisfaz o invariante por não ter nada a
satisfazer, e a suíte parte da tabela de fiação — chave sem linha na tabela não
produz asserção nenhuma.

Quatro chaves estavam nesse estado, declaradas e lidas por ninguém:

| Chave | Sintoma |
|---|---|
| `limits.max_turn_tokens` | `loop.Limits.MaxTurnTokens` existe e é usado em `turn.go`; `app.go` nunca o preenchia, logo o teto de token nunca disparava |
| `behavior.instructions_enabled` | default declarado, nenhum leitor |
| `behavior.skills_enabled` | idem |
| `update.channel` | mapeada para `DCODE_UPDATE_CHANNEL`; o código lia `DCODE_RELEASE_CHANNEL` |

A verificação passa a partir de **`KnownKeys`**, não da tabela. Chave nova falha
até alguém ligar as pontas ou declarar, com motivo, que ela não pertence a uma
sessão — e chave assim declarada precisa ser lida por algum comando **por aquele
nome**, que é o que teria pego a divergência de `update.channel`.

`update` passou a resolver pela cadeia comum em vez de `os.Getenv` direto: ler o
ambiente por fora é o que fazia `update.channel` ser chave que arquivo de config
nunca alcançava. `DCODE_RELEASE_CHANNEL` segue funcionando como fallback — era a
única grafia que funcionava, logo é a que quem configurou canal está usando.
