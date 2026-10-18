# A spec alcança o código

**Data:** 2026-10-17
**Specs afetadas:** `202608301200-recoverable-cycle` (`.r`, cabeçalho e RN-4;
`.p`, §1, §2, §5, §6, §8 e §9)
**Fonte:** revisão das specs contra o código. Só documentação: nenhum
comportamento muda.

## O que estava desatualizado

- **O lembrete do `.p` §5 não era o que o código emite.** O texto da spec
  (*"The last cycle was undone…"*) era um rascunho; `cycleUndone`, em
  `internal/behavior/reminders.go`, emite o texto que o changelog
  [202608301400](202608301400-o-laco-desfaz.md) já registrava —
  *"Your last set of changes was undone…"* — e só escreve as frases de
  restaurados e de deixados quando houve algum. A seção agora traz esse texto.
- **`Progressed` aparecia como se existisse.** O `.p` §1 dizia que ele passava
  a devolver três respostas, e o §9 que ele seria "escrito sobre" `Moved`. Ele
  saiu, e o próprio §2 já dizia isso. As duas frases agora falam de `Moved`, e
  a RN-4 da `.r`, que descrevia o código de agosto no presente, diz que
  descreve o de agosto.
- **Duas referências cruzadas apontavam para a etapa errada.** O §6 dizia que
  o teto de ciclos parados "é a etapa 4 da §9" — a etapa 4 é a medição riscada,
  e subir o teto era a etapa 5 da `failure-feedback`, que o §8 daqui tirou do
  plano. O §8 dizia que o contrato de qualidade de correção "é a etapa 2 da
  §9", que é `BeginCycle`.
- **O cabeçalho da `.r` dizia "sem `.p`, sem código".** As etapas 1 a 3 estão
  entregues.
