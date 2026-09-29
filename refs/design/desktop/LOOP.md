# DCode Desktop — Loop de execução

Tela de referência: `DCode Desktop v2.dc.html` → "Loop de execução".

> **Tudo aqui é proposta.** Nada disso existe em `docs/specs/` hoje. O comando,
> as flags e as regras de parada precisam de changelog em
> `docs/specs/architecture/agent-loop/` antes de codar. Onde este documento e a
> spec divergirem, a spec vence.

## O que é um loop

É uma sessão que repete o turno até a **definição de pronto** passar. A cada
iteração o modelo trabalha, depois o daemon roda os checks do `done.toml` na
árvore inteira. Se os checks não passam, começa a próxima iteração. Se passam, o
loop termina com o selo `✓ verified`.

Uma sessão comum para quando o modelo responde. Um loop para quando **os checks**
dizem que acabou, nunca quando o modelo diz que acabou.

## Como entrar

### 1. Pelo composer: `/loop` (entrada principal)

```
/loop faz os 12 testes de internal/integration passarem sem mexer nos fixtures
```

- Tudo depois de `/loop` vira o **objetivo** (o cabeçalho da tela).
- Os checks de "pronto quando" vêm do `done.toml` do workspace.
- `↵` abre a folha **Configurar loop** (tela a desenhar). `⌘↵` pula a folha e
  começa direto com os valores padrão.

Flags opcionais, na mesma linha:

| Flag | Padrão | O que faz |
|---|---|---|
| `--max 100` | 100 | teto de iterações |
| `--tokens 1M` | 1M | teto de tokens (os tokens dos filhos contam para o pai) |
| `--tempo 2h` | 2h | teto de tempo total |
| `--parado 3` | 3 | iterações sem progresso antes de pausar e perguntar |
| `--checks "go test ./..."` | `done.toml` | substitui os checks só neste loop |

Sem `done.toml` e sem `--checks`, o `/loop` **não começa**. A folha abre pedindo
pelo menos um check, porque um loop sem critério de parada é só um teto de gasto.

### 2. Atalho: `⌘⇧↵`

Com texto no composer, `⌘⇧↵` envia **como loop**. É o mesmo que `/loop <texto>`.
O botão de enviar ganha o glifo `↻` enquanto `⇧` estiver pressionado.

### 3. Continuar uma sessão em loop

Quando um turno termina com `⚠ unverified` ou `!! NOT verified !!`, o selo
oferece **"Continuar em loop até ficar pronto"**. A sessão vira loop a partir da
próxima iteração, e a conversa anterior conta como contexto, não como
iteração.

### 4. Pela linha de comando

```
dcode loop "faz os testes de integração passarem" --max 50
```

O desktop e a TUI falam com o mesmo daemon, então um loop aberto no terminal
aparece na coluna lateral do desktop com `↻ n/max`, e o contrário também vale.

## Onde o loop aparece

- **Coluna lateral:** ponto âmbar pulsando + `↻ 8/100` à direita da sessão.
- **Seletor de sessões (`⌘K`):** entra em "Ativas", com a mesma marca.
- **Barra inferior:** `<verbo>… iteração 8 · 15m22s`.

## Controles durante o loop

| Ação | Tecla | Efeito |
|---|---|---|
| Orientar a próxima iteração | `↵` no composer | a mensagem entra no **início** da próxima iteração; não interrompe a atual |
| Parar depois desta iteração | `⌘.` | termina a iteração atual, roda os checks uma última vez e para |
| Interromper agora | `esc` | corta a iteração no meio; `⌘Z` desfaz a iteração inteira, filhos incluídos |
| Ver uma iteração | clique na coluna, `←/→` | mostra resumo, ferramentas e checks daquela iteração |

## Regras de parada

1. **Todos os checks passam** → conclui com `✓ verified`.
2. **N iterações seguidas sem progresso** (`--parado`, padrão 3) → **pausa e
   pergunta**. Progresso = mais checks passando que na iteração anterior. A
   coluna sem progresso leva `=` e cor `warn`.
3. **Algum limite acaba** (iterações, tokens, tempo) → para com o estado atual e
   o selo honesto (`⚠ unverified` ou `!! NOT verified !!`).

Pedido de protocolo: o daemon precisa emitir, por iteração, **quantos itens de
cada check passaram** (por exemplo `go test: 11/12`), não só passou ou falhou.
Sem isso a regra 2 e o placar não existem. Não parsear stdout no cliente.

---

## Telas que faltam

### Do loop (para o fluxo ficar completo)

1. **Configurar loop:** a folha que abre no `/loop` + `↵`. Mostra o objetivo, os
   checks do `done.toml` (editáveis só neste loop), limites, `--parado`,
   sandbox e política. Tem o botão "Começar loop".
2. **Loop pausado por falta de progresso:** mostra as 3 colunas com `=` e o que
   foi tentado em cada uma. Ações: continuar, orientar e continuar, parar.
3. **Loop concluído:** selo `✓ verified`, placar completo, diff total, "Criar PR"
   e "Desfazer loop".
4. **Loop parado por limite ou interrompido:** qual limite acabou, o selo
   honesto e "Continuar com mais N iterações".
5. **Aprovação dentro de uma iteração:** o loop espera você. A marca lateral
   passa de âmbar para o losango `warn`, e o relógio do loop **para**.
6. **Delegação dentro de uma iteração:** o card de filhos aparece no detalhe da
   iteração; filho que não respondeu é nomeado ali.

### Do app (fora do loop)

7. **Nova sessão / projeto vazio:** o que aparece em `⌘N`, com a entrada
   `/loop` visível.
8. **Seletor de modelo** com a nota dos contratos (pedido do brief). Trocar de
   modelo continua a sessão.
9. **Revisão de mudanças:** diff do turno ou do loop, por arquivo, com `⌘Z`.
10. **Daemon desconectado / erro:** o que acontece com sessões rodando quando o
    daemon cai e volta.
11. **Configurações:** tema (claro/escuro/sistema, proposta a validar),
    política e sandbox padrão, `tui.activity_verbs`.
12. **Abrir projeto:** primeiro uso e adicionar workspace à coluna lateral.
13. **Modo full-access:** o único lugar com `danger` (branco sobre vermelho),
    e o texto muda além da cor.
14. **Todas as telas em claro:** hoje o tema claro é só o tweak `theme`; falta
    conferir o contraste do âmbar e do `warn` em cada estado.

Ordem sugerida: 1 → 2 → 3 → 5, porque sem elas o loop não começa nem termina.
Depois 7 e 8, que o brief pede.
