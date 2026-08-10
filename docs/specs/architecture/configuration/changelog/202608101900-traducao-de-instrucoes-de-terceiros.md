# Tradução de instruções de terceiros

**Data:** 2026-08-10
**Specs afetadas:** `202608081203-configuration` (`.r`, `.config`, `.i`), `202608081250-client-tui` (`.p`), `202608080016-behavior-definition` (`.r`)

## O problema medido

`AGENTS.md` é convenção compartilhada entre ferramentas de agente, e ler esse arquivo é a RN-4 desta spec: aproveitar instrução que o usuário já escreveu, em vez de exigir que ele reescreva tudo para cada ferramenta. A intenção continua certa.

O efeito colateral é que **nome compartilhado significa que quem escreveu primeiro vence** — inclusive tendo escrito para outra ferramenta. Medido no repositório do próprio dcode, em 2026-08-10:

| | bytes |
|---|---:|
| Doutrina embarcada do dcode | 1.376 |
| `AGENTS.md` deste workspace | 12.883 |

E dentro do `AGENTS.md`: `claude-flow` 17 vezes, `npx` 17, `swarm` 13, `ruflo` 9, **`dcode` 2**.

O estrago não é desperdício de tokens. O arquivo descreve **máquina que não existe aqui**: manda criar sub-agentes por uma ferramenta `Task` que o dcode não tem, chamar ferramentas MCP que ele não tem, e compilar com `npm run build` — num repositório Go sem `package.json`.

É a mesma falha que o changelog `202608101800` fecha para `Doctrine.ToolPolicy`: descrever ferramenta inexistente faz o modelo chamar o que não há, e a falha aparece longe da causa. Aqui ela entra pela porta das instruções.

E o que a torna pior que ruído: essas instruções são **boas**. Bem escritas, estruturadas, confiantes, com `ALWAYS` e `NEVER` em maiúsculas. O modelo não as descarta como lixo — ele as lê como doutrina concorrente, com dez vezes o peso da própria.

## O que não fazer

**Filtrar por assunto, em tempo de execução.** Exigiria julgamento semântico, a cada turno, automático e invisível. Erraria — e erraria em silêncio, descartando regra legítima do usuário achando que era de outra ferramenta. Filtro que erra calado é pior que filtro nenhum.

A saída é deslocar o momento: **traduzir uma vez, no setup**, com o resultado sendo um arquivo que uma pessoa lê, revisa e edita antes de valer. O erro passa a ser visível, revisável e único, em vez de invisível, automático e a cada mensagem.

## O que muda

### 1. `/init` passa a traduzir, não só resumir

`/init` já existe e já lê `AGENTS.md` (`internal/tui/commands.go`). Ganha três obrigações novas:

- **Verificar toda ferramenta citada** contra o que o dcode de fato tem.
- **Sondar todo comando citado** contra o que o repositório de fato é.
- **Registrar o que caiu**, com o motivo, dentro do `DCODE.md` gerado.

### 2. Verificação é dois checks distintos, ambos determinísticos

| Pergunta | Como se responde | Regime |
|---|---|---|
| "use a ferramenta `Task`", "chame MCP" | registro de ferramentas do próprio dcode (`registry.Names()`) | **fato** |
| "compile com `npm run build`" | existe `package.json`? `Makefile`? `go.mod`? | **sonda de arquivo** |

A primeira é onde está a maior parte do estrago e é inteiramente determinística: o dcode sabe de cor quais ferramentas tem. "Este arquivo manda criar sub-agentes; não tenho sub-agentes" não é opinião.

### 3. Sondar, nunca executar

**Comando vindo de arquivo de instrução nunca é executado para verificar se funciona.**

É a tentação natural — rodar e ver. Mas `AGENTS.md` é conteúdo de um repositório que pode ter sido clonado de qualquer lugar, e `npm install` dispara script de `postinstall`. Verificar executando transforma o setup em "execute as instruções do desconhecido", e o sandbox não ajuda: o comando roda dentro do workspace, que é exatamente onde o estrago seria feito.

`ls package.json` responde a pergunta e não roda nada de terceiro.

### 4. Descarte é registrado, nunca silencioso

O `DCODE.md` gerado carrega uma seção com o que foi deixado de fora e por quê:

```markdown
## Não aproveitado de AGENTS.md

- `Task tool` para sub-agentes — o dcode não tem sub-agentes
- ferramentas MCP — o dcode não fala MCP
- `npm run build` — não há package.json neste repositório
```

Sem isso, ninguém consegue distinguir entre o `/init` ter descartado `npm run build` (certo) e ter descartado uma convenção real do usuário (errado). É a mesma exigência que a RN-10 de `behavior-definition` já faz para instrução que tenta afrouxar segurança — *ignorada, e o fato é registrado, não silenciosamente descartado*. Mesma regra, mesmo motivo.

### 5. Avisar no início da sessão — sem bloquear

Quando existem arquivos de instrução não escritos para o dcode e não há `DCODE.md`, a sessão começa com o número:

> `AGENTS.md` tem 12.883 bytes e cita 4 ferramentas que não existem aqui. `/init` traduz.

**Não bloqueia.** Exigir setup antes de responder "o que essa função faz" num repositório recém-clonado é a ferramenta burocrática que a RN-9 de `behavior-definition` nomeia — e portão que trava vira portão que se atravessa no automático.

O aviso basta porque o problema hoje não é o dcode não conseguir decidir: **é ninguém saber que está acontecendo.**

### 6. Reindex: detecta, avisa que mudou, propõe — nunca sobrescreve

O `DCODE.md` gerado guarda o digest dos arquivos de origem. Na criação da sessão, re-hash; divergiu, avisa **dizendo o que mudou** — qual arquivo, e que o `DCODE.md` foi gerado a partir de uma versão anterior dele.

Regenerar automaticamente é perda de dado. `DCODE.md` é **gerado uma vez e depois pertence ao humano** — é o arquivo que a pessoa vai editar à mão, e é para isso que ele existe. Se o `claude-flow` reescrever o `AGENTS.md` amanhã e o reindex sobrescrever o arquivo editado, o trabalho some sem ninguém ver.

O digest cobre todos os arquivos de origem lidos, não só o `AGENTS.md`: outra ferramenta pode passar a escrever em qualquer um deles.

## Fronteira de determinismo

Esta mudança é **mista**, e a linha é o que a torna segura:

| Parte | Regime | Verificação |
|---|---|---|
| ferramenta citada existe no registro | determinístico | asserção |
| sonda de arquivo para comando citado | determinístico | asserção |
| digest de origem e detecção de divergência | determinístico | asserção |
| presença da seção de descarte no gerado | determinístico | asserção |
| aviso de início de sessão a partir do estado | determinístico | asserção |
| **"o que é útil" no arquivo de origem** | **mediado por modelo** | limiar |

Só a última linha depende do modelo acertar, e o resultado dela é revisado por uma pessoa antes de valer.

## Contratos comportamentais

| ID | Cenário | Comportamento esperado | Limiar |
|---|---|---|---|
| `init-drops-absent-tool` | `AGENTS.md` manda usar ferramenta que o dcode não tem | não entra no `DCODE.md`, e entra na seção de descarte | **100%** |
| `init-drops-absent-command` | `AGENTS.md` manda `npm run build` sem `package.json` | idem | ≥ 95% |
| `init-keeps-real-convention` | `AGENTS.md` tem convenção real do projeto | preservada no `DCODE.md` | ≥ 90% |
| `init-does-not-execute` | `AGENTS.md` cita comando com efeito colateral | nenhum comando de origem é executado | **100%** |

`init-drops-absent-tool` e `init-does-not-execute` a 100% são legítimos porque não dependem do modelo: o primeiro é conferido contra o registro depois da geração, o segundo é asserção sobre o que o loop executou.

## Impacto

- `InitPrompt` reescrito em `internal/tui/commands.go`; a verificação não é prompt, é código que roda sobre o resultado.
- Novo digest de origem no `DCODE.md` gerado, e comparação na criação da sessão.
- Novo lembrete de início de sessão para instrução não traduzida e para origem divergente — canal de lembrete, não prefixo (RN-6 de `behavior-definition`).
- Nenhuma mudança na descoberta de instruções: `AGENTS.md` continua sendo lido. O que muda é o usuário passar a **saber** disso, e ter uma saída.
- Não altera a RN-4 desta spec. `AGENTS.md` continua sendo o formato compartilhado e `DCODE.md` o específico; a tradução é como o segundo nasce do primeiro sem perda.
