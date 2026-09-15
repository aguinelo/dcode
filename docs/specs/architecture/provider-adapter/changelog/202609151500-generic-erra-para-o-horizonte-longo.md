# `generic` erra para o horizonte longo

**Data:** 2026-09-15
**Specs afetadas:** `202608072334-provider-adapter` (`.p`, seções 2.1 e 2.2)
**Fonte:** o mesmo relato de campo do turno mudo — depois de corrigido o
silêncio (`docs/specs/architecture/client-tui/changelog/202609151400-o-turno-que-parava-mudo.md`),
o usuário perguntou o que o teto de rodadas regula e qual o impacto de
subir bastante; a resposta expôs que `generic` herdava 50 sem nunca ter
sido a decisão certa pra o que essa família serve.

## O que mudou

`Generic.DefaultLimits().MaxIterations`: 50 → **5.000**. E `models.toml`
ganha um campo a mais no pacote de um perfil: `max_iterations`, mesma
semântica de `limits.max_iterations` — zero é "pergunte à família",
maior que zero sobrescreve.

```toml
[profile.qwen-local]
model          = "qwen3.5-9b"
family         = "generic"
max_iterations = 5000
```

## Por que 50 estava errado

`generic` herdava o mesmo default de `claude` — dimensionado pelo caso de
um refactor cruzando dez arquivos. Certo pra Claude, que É medido contra
esse horizonte. Errado pra `generic`, que na prática é usado por um modelo
que dcode nunca viu — e, mais frequentemente, um modelo **local**, rodado
especificamente por alguém fugindo de um teto de nuvem. Uma tarefa real de
código (migração de i18n num projeto Next.js, várias dezenas de arquivos)
passa de 50 chamadas de ferramenta com folga, e o turno cortava no meio,
silenciosamente — o defeito que a entrada irmã desta corrige do lado do
cliente.

## Por que 5.000 é seguro

Perguntado diretamente: "qual o impacto de desabilitar e deixar infinito?"
A resposta, lendo o código: **hoje não dá pra deixar infinito de verdade**
— zero ou ausente sempre cai no default da família (`internal/loop/limits.go`,
`withFamily`), nunca em "sem teto". E o teto de iteração **não é** a defesa
real contra loop patológico — é o detector de repetição
(`MaxIdenticalCalls`), que dispara em N chamadas idênticas seguidas,
independente de quantas rodadas já rolaram. O teto de iteração é só
backstop de custo e tempo de parede; um modelo local não paga o primeiro em
dólar de API nenhum, o que torna um número alto seguro especificamente pra
essa família.

## O que não mudou

`MiniMax-M3` continua em 2.000 — número medido contra uma execução real
(1.959 tool calls), não um chute, e não havia motivo relatado pra revisar
uma medição. `claude` continua em 50 — dimensionado pelo próprio caso que o
justifica, e nada neste relato era sobre Claude. Só `generic`, cujo número
nunca tinha sido uma decisão — era herança.

`Family.DefaultLimits()`, `Limits.withFamily` e a interface declarada em
`.p` §2 seguem intocadas: a mudança é só o literal dentro de
`family_generic.go`, e o novo campo é aditivo em `Profile`
(`internal/config/models.go`) e em `applyModelRequest`
(`internal/app/daemon.go`) — nenhum dos dois muda de forma, só ganha um
campo a mais que zero não altera nada.
