# Lacunas de spec — DCode Desktop v1

O design assume os comportamentos abaixo, e nenhum deles está em
`docs/specs/`. **Não implemente nenhum item sem antes abrir o changelog da spec
correspondente.** Onde o item conflita com uma regra existente, isso está
indicado.

Status sugerido para cada item: `proposto → aceito | recusado | alterado`.

---

## A. Cliente desktop (nova área: `docs/specs/architecture/client-desktop/`)

A1. **O cliente desktop existe** e fala com o mesmo daemon da TUI e da CLI.
Sessões são compartilhadas entre clientes. *Precisa:* a decisão de framework e o
transporte (o mesmo protocolo da TUI?).

A2. **Vários projetos na mesma janela.** A lateral agrupa sessões por projeto.
⚠ **Conflita** com a regra da TUI "um workspace só — outro projeto é outro
DCode aberto naquele diretório". *Proposta:* cada sessão continua ancorada num
único workspace (sandbox, política e cadeia de instruções não mudam); a janela
só **lista** vários. Decidir se o daemon é um só para todos os workspaces ou um
por workspace.

A3. **Uma sessão visível por vez**, ocupando a área toda. As outras continuam
rodando no daemon. Não há split.

A4. **Projetos: renomear, reordenar, recolher.** Renomear muda só o rótulo, nunca
o diretório. Ordem, rótulos e estado recolhido são **preferências locais do
cliente**, e o daemon não sabe deles. *Decidir:* sincronizar entre máquinas ou
não.

A5. **Seletor `⌘K`** com escopos `todas` / `ativas` / `<projeto>` e prévia do
último evento. `⌘1…⌘9` abrem as sessões ativas.

A6. **Contagens na barra inferior** (`n rodando`, `n esperando você`) e na linha
de projeto recolhido.

A7. **Aprovação com três opções:** permitir uma vez / permitir nesta sessão /
negar. *Confirmar* se "nesta sessão" já existe na política de aprovação ou se é
novo.

A8. **Mensagem durante o turno entra na fila** (*Mensagem entra na fila depois
deste turno*). *Confirmar* com a spec da fila.

A9. **Tema claro.** Os tokens do tema claro são proposta. Falta validar o
contraste do âmbar e do `warn` em todos os estados.

## B. Loop (em `docs/specs/architecture/agent-loop/`)

Ver `LOOP.md` para o detalhe.

B1. **Modo loop:** repetir o turno até os checks do `done.toml` passarem. O fim
é decidido pelos checks, nunca pelo modelo.

B2. **Entrada:** `/loop <objetivo>` no composer; `⌘⇧↵`; "Continuar em loop" a
partir de um selo não verificado; `dcode loop "…"` na CLI.

B3. **Flags:** `--max` (100), `--tokens` (1M), `--tempo` (2h), `--parado` (3),
`--checks`. Tokens dos filhos contam para o pai.

B4. **Sem `done.toml` e sem `--checks`, o loop não começa.**

B5. **Regras de parada:**
1. todos os checks passam → conclui com `verified`;
2. N iterações sem progresso → pausa e pergunta;
3. um limite acaba → para, com o selo honesto.

B6. **Definição de progresso:** mais itens de checks passando do que na iteração
anterior.

B7. **Mensagem no loop entra no início da próxima iteração** e não interrompe a
atual.

B8. **Controles:** parar depois desta iteração (`⌘.`, roda os checks uma última
vez); interromper agora (`esc`); `⌘Z` desfaz a iteração inteira, filhos
incluídos.

B9. **Aprovação dentro do loop:** o relógio do loop para enquanto espera você.

## C. Protocolo do daemon

C1. **Resultado de check com contagem, por iteração** (ex.: `go test: 11/12`),
não só passou/falhou. Sem isso não existem o placar nem a regra B5.2. **O
cliente não deve parsear stdout.**

C2. **Evento de iteração:** início, fim, resumo, duração, tokens, diff.

C3. **Estado agregado por sessão** para a lateral e o `⌘K`: estado, branch,
diff, último evento, selo e idade, sem precisar reproduzir o log inteiro.

C4. **Aprovação pendente** com `desde` (para o relógio ao vivo) e os recursos
pedidos (rede, caminhos fora do sandbox).

---

## Fora do escopo desta versão (telas ainda não desenhadas)

Configurar loop · loop pausado · loop concluído · loop parado por limite ·
nova sessão / projeto vazio · seletor de modelo com a nota dos contratos ·
revisão de diff · daemon desconectado · configurações · abrir projeto ·
modo full-access (`danger`). A lista com a ordem sugerida está em `LOOP.md`.
