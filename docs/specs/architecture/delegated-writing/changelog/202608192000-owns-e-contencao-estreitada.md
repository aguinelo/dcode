# `owns` e a contenção estreitada

PR 1 da §9 do `.p`. Um filho delegado passa a escrever, dentro do que declarou
possuir, e a contenção é quem responde por isso.

## O que mudou

`explore` ganha um campo, e só um:

```jsonc
{ "task": "...", "path": "repos/billing", "owns": ["repos/billing/ARCHITECTURE.md"] }
```

- **ausente** — o filho somente-leitura que já existia, sem diferença nenhuma;
- **presente** — pedido de filho escritor, herdando o modo do pai;
- **vazio** — erro de declaração, nunca permissão total.

Continua não havendo campo de modo. O que o modelo passa é tarefa e caminhos, e
ambos só estreitam.

## Onde a garantia mora

`Resolver.Owning(paths)` devolve **outro** resolvedor, com as escritas confinadas
ao conjunto. `InWorkspace` — o mesmo predicado que a política já consulta —
passa a responder não para escrita fora dele. Não é verificação nova ao lado da
antiga: é a antiga, com o conjunto menor.

Leitura fica inteira de propósito. Um filho que cataloga um repositório lê o
repositório todo e escreve um arquivo; estreitar leitura junto tornaria a
capacidade inútil para o caso que a justifica.

Quatro propriedades caíram de graça, porque a contenção já as tinha:

- posse é por **componente de caminho**, nunca por prefixo — `docs2` não está
  dentro de `docs`;
- possuir caminho **fora** do workspace não o traz para dentro;
- possuir **nada** não é possuir tudo;
- estreitar um filho **não** estreita o pai, porque `Owning` devolve valor novo.

## O que o filho escritor não ganhou

**`bash`.** Comando de shell é opaco — o scheduler já roda um sozinho por isso —
então nada pode ser provado sobre o que ele tocaria, e contenção estreitada
estaria discutindo com um processo em vez de com uma declaração. Se um filho que
escreve pode rodar comando é pergunta em aberto no `.r`; excluir é o lado errado
barato de errar enquanto ela não tem resposta.

**Delegar.** Continua impossível por ausência, como já era.

**Conferir.** As instruções do filho escritor dizem, com todas as letras, para
não rodar a suíte: a árvore ainda vai mudar e conferi-la não é trabalho dele. A
definição de pronto é do pai, uma vez, sobre a árvore inteira.

## O que este PR não entrega

`Wrote` no relatório, as fotos do filho no desfazimento do turno do pai, e o
filho mudo nomeado — todos no PR 2. Os contratos comportamentais, no PR 3. A
§8.1 do `.p` é a lista do que falta.

## Uma nota de processo

`childConfig` foi separada de `Delegate` para que as garantias possam ser
asseridas sem rodar um turno contra um modelo. Garantia que só se confere
rodando a coisa que ela protege é garantia que ninguém confere.

E uma guarda cobrou o autor no meio do caminho: `Narrowed()` foi escrito, usado
só pelo teste, e o teste de nome não usado reprovou. Estava certo — o teste
já provava a coisa real duas linhas abaixo, e o método era conveniência.
