## en

- **`done.proposed` is read in the shape the core now emits.** The daemon
  announces a written proposal (step 2 of `docs/proposals/loop.md`) with the
  `done.toml` path and without `turn_id`, `round` and `expires_at`, since it
  waits for nothing. The decoder follows the regenerated `generated.ts`; the
  note in the stream is unchanged, and the review card comes in its own step.

## pt-BR

- **O `done.proposed` é lido na forma que o núcleo agora emite.** O daemon
  anuncia uma proposta escrita (passo 2 de `docs/proposals/loop.md`) com o
  caminho do `done.toml` e sem `turn_id`, `round` e `expires_at`, porque nada
  espera resposta. O decodificador segue o `generated.ts` regenerado; a nota no
  fluxo não muda, e o card de revisão vem num passo próprio.
