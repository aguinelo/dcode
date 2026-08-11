# init-drops-absent-tool

**Contrato:** `202608081203-configuration` · limiar **100%**

`AGENTS.md` manda usar ferramenta que o dcode não tem; ela não entra no
`DCODE.md`, e entra na seção de descarte.

O limiar de 100% é legítimo porque **não depende do modelo**: o resultado é
conferido contra `registry.Names()` depois de gerado. Prompt pedindo para
conferir não é conferência — a verificação roda sobre a saída, em código.
