## en

- **Four specs and the roadmap are brought back in line with the code.**
  Documentation only; no behaviour changes. `provider-adapter` now names its
  four families and their transports, the stream events and decoder as they
  are (reasoning, opened/progress, `Truncated`, `Decoder.Close`), and the
  registry's real signatures (`RegisterFamily` returns an error;
  `ResolveFamily` and `FamilyFor` exist). Its promise that each family also
  reads its conventional credential variable is marked not implemented, and
  the undated Claude aliases falling to the defaults are recorded as a known
  gap. `recoverable-cycle` carries the reminder the code actually sends and
  fixes its cross-references. `sandbox-policy` copies the real `Evaluate`,
  `Access` and `Sandbox` signatures. `tool-suite` gains sections for `process`
  and `fetch`, says which tools in the package belong to other families, fixes
  its error-code table (`bad_input` in, the nonexistent `is_binary` out) and
  moves the two fetch keys out of the non-configurable constants. The roadmap
  stops listing writing sub-agents as not done, corrects the stale story about
  the eval harness refusing delegated calls, and records the task ledger as
  approved and unbuilt. Each was a spec stating something the code no longer
  does, which is the one kind of documentation that is worse than none.

## pt-BR

- **Quatro specs e o roadmap voltam a bater com o código.** Só documentação;
  nenhum comportamento muda. A `provider-adapter` passa a nomear as quatro
  famílias e seus transportes, os eventos de stream e o decoder como são
  (raciocínio, opened/progress, `Truncated`, `Decoder.Close`) e as assinaturas
  reais do registro (`RegisterFamily` devolve erro; `ResolveFamily` e
  `FamilyFor` existem). A promessa de que cada família também lê a variável
  convencional de credencial fica marcada como não implementada, e os aliases
  Claude sem data caindo nos defaults ficam registrados como lacuna conhecida.
  A `recoverable-cycle` traz o lembrete que o código de fato envia e conserta
  as referências cruzadas. A `sandbox-policy` copia as assinaturas reais de
  `Evaluate`, `Access` e `Sandbox`. A `tool-suite` ganha seções para `process`
  e `fetch`, diz quais ferramentas do pacote são de outras famílias, conserta a
  tabela de códigos de erro (entra `bad_input`, sai o inexistente `is_binary`)
  e tira as duas chaves de `fetch` das constantes não configuráveis. O roadmap
  deixa de listar subagentes que escrevem como não feitos, corrige a história
  desatualizada do harness de eval recusando chamada delegada e registra o
  ledger de tarefas como aprovado e não construído. Cada uma era uma spec
  afirmando o que o código já não faz, que é o único tipo de documentação pior
  que nenhuma.
