# init-drops-absent-tool

**Contrato:** `202608081203-configuration` · limiar **100%**

`AGENTS.md` manda usar ferramenta que o dcode não tem; ela não entra no
`DCODE.md`, e entra na seção de descarte.

O limiar de 100% é legítimo porque **não depende do modelo**: o `DCODE.md`
gerado passa por `config.VerifyTools` — a mesma função do produto que avisa, no
início da sessão, sobre instruções escritas para outro agente. Prompt pedindo
para conferir não é conferência; a verificação roda sobre a saída, em código.

> Esta nota afirmou por meses um conferimento contra `registry.Names()` que
> **não existia**. O juiz por baixo era "escreveu algum arquivo", idêntico ao
> dos outros dois contratos `init-*`. O limiar de 100% estava justificado por
> um mecanismo que ninguém tinha construído — que é o mesmo achado que abriu
> esta trilha inteira.

A tarefa é o `InitPrompt` que o `/init` envia, não um resumo dele. Um teste
falha se o `task.md` divergir do produto: cada frase que este contrato mede
— traduzir em vez de copiar, não há subagentes, conferir se o arquivo existe —
vive só nesse prompt.
