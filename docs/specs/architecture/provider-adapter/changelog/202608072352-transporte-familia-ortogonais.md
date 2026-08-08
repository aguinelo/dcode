# Transporte e família como eixos ortogonais

**Data:** 2026-08-07
**Specs afetadas:** `202608072334-provider-adapter` (`.r`, `.p`, `.config`, `.i`), `202608072335-agent-loop` (`.config`)

## O que mudou

A RN-1 dizia *"adaptação é por família, não por endpoint"*. Correto, mas com uma dimensão a menos. Agora são **dois eixos ortogonais**:

- **Transporte** — formato de fio (`openai`, `anthropic`). Reusável entre famílias. Não carrega limiar.
- **Família** — prompt, schema de tool, estratégia de edição (`minimax-m3`, `claude`). Carrega os limiares dos contratos comportamentais e os limites padrão do turno.

Uma família declara os transportes com que é compatível, em ordem de preferência.

Nova **RN-9**: os limites padrão do turno — teto de iteração, principalmente — passam a ser declarados pela família, não fixados globalmente.

## Por que mudou

**O caso que forçou a mudança foi o MiniMax M3**, escolhido como modelo principal do projeto. M3 fala **os dois dialetos**: Anthropic-compatible e OpenAI-compatible. Com um eixo só, suportar ambos exigiria duplicar a família inteira — mesma adaptação, mesmos limiares, dois adaptadores.

A consequência mais importante é de segurança de qualidade, não de organização de código: **"OpenAI-compatible" descreve como a requisição é serializada, não como o modelo se comporta.** Dois modelos atrás do mesmo endpoint OpenAI-compatible podem ter comportamento de tool-calling radicalmente diferente. Tratar o formato de fio como se fosse família faria o dcode aplicar a um modelo os limiares medidos para outro — entregando tool-calling não validado com aparência de validado.

Sobre a RN-9: o default de 50 iterações foi dimensionado pelo caso do Claude Code — um refactor cruzando dez arquivos. **M3 foi treinado para *long-horizon agent loops*; a MiniMax demonstrou uma execução com 1.959 tool calls.** Um teto global serve mal aos dois perfis. O detector de repetição continua sendo o mecanismo real contra loop patológico; o teto é backstop, e backstop precisa acompanhar o horizonte do modelo.

## Impacto

- `Provider` deixa de ser interface única e passa a ser composição de `Transport` e `Family`.
- `Registry.Resolve` resolve modelo → família, depois escolhe o transporte preferido ou o sobrescrito por config.
- Nova variável `DCODE_TRANSPORT`, validada contra os transportes declarados pela família.
- `DCODE_MAX_ITERATIONS` passa a ter default **por família**; vazio usa o da família em uso.
- Nenhum limiar de contrato comportamental existente é invalidado — eles já eram por família na prática, apenas não estavam nomeados assim.

## Alternativa descartada

Manter um eixo só e criar as famílias `minimax-m3-openai` e `minimax-m3-anthropic`. Descartada porque duplicaria adaptação e limiares idênticos, e porque a duplicação divergiria na primeira manutenção — um lado receberia correção que o outro não.
