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

## O que o juiz lê, e o que ele ignora

Só a parte **carregada** do `DCODE.md` — tudo antes da seção
`## Not carried over from AGENTS.md`.

O `InitPrompt` **exige** essa seção, e exige que ela liste o que ficou de fora e
por quê: sem ela ninguém distingue descarte correto de regra do usuário perdida
por engano. Então nomear a ferramenta ausente **ali** é o contrato cumprido.

> O juiz lia o arquivo inteiro e reprovava exatamente a frase que prova o
> trabalho ter acontecido: 4% em 50 execuções, com as transcrições mostrando o
> modelo lendo o `AGENTS.md`, reconhecendo ferramental Node num módulo Go e
> traduzindo certo. Foi defeito meu, do mesmo tipo que este contrato existe para
> pegar.

Arquivo sem a seção é julgado inteiro: não há o que excluir, e presumir uma
seção que não está lá seria parar de julgar em silêncio.
