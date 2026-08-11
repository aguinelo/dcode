# toolcall-schema-valid

**Contrato:** `202608072334-provider-adapter.p.spec.md`, seção 6 · limiar **≥ 97%**

Ferramenta com schema de objeto aninhado; a tool call valida contra o schema na
primeira tentativa.

## Por que o juiz confere o schema aqui, e não confia no adaptador

A spec descreve `EventToolCall` como *"já validado contra schema"*. Medido em
2026-08-10, `validateToolCall` (`internal/provider/family_openai.go:608`) confere
duas coisas — que o nome foi declarado, e que os argumentos são JSON válido — e
**não** confere o schema.

Se este cenário confiasse naquela linha, mediria "o modelo devolveu JSON", que é
uma pergunta muito mais fácil e que passa com `{}`. O juiz decodifica a estrutura
aninhada e exige os campos obrigatórios, porque é isso que o contrato diz.

## Material

`task.md` é o pedido. `tools.json` é o conjunto declarado, com um único item
cujo schema tem objeto dentro de objeto e array de objetos — a forma que separa
um modelo que preenche schema aninhado de um que devolve o nível de cima.
