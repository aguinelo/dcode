## en

- **`generated.ts` carries the loop's new fields.** The core's `LoopRun` gains
  `args` and each `LoopSpecRun` what the survey found (`criteria`, `unmet`,
  `unavailable`, `error`) and what its proposal wrote (`proposal_criteria`,
  `proposal_summary`), so the loop sheet and the review card draw them without
  measuring again. Regenerated, and the loop decoder accepts them as optional
  (an absent `criteria` is zero); nothing in the app draws them yet.

## pt-BR

- **O `generated.ts` traz os campos novos do loop.** O `LoopRun` do núcleo
  ganha `args`, e cada `LoopSpecRun` o que a pergunta achou (`criteria`,
  `unmet`, `unavailable`, `error`) e o que a proposta escreveu
  (`proposal_criteria`, `proposal_summary`), para a folha do loop e o card de
  revisão os desenharem sem medir de novo. Regenerado, e o decodificador do
  loop os aceita como opcionais (`criteria` ausente é zero); nada no app os
  desenha ainda.
