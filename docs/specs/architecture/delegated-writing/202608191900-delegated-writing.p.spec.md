# Planning: Delegação que escreve

## 1. Nível de estabilidade

**Desenho aprovado, não implementado.** Este documento descreve o que será
construído; nenhuma linha dele existe em código ainda.

Duas ausências são consequência disso, e ambas são o repositório funcionando:

- **A seção de invariantes chama-se "previstas".** Aqui uma invariante
  verificável é reivindicação sobre um teste que existe, e não há o que
  reivindicar antes do código. É renomeada e cobrada pela guarda no PR da
  implementação. **A §8 já foi renomeada e a §8.1 deixou de existir**: os três
  PRs da §9 entregaram todas.
- **Não há `.i.spec.md`.** A guarda exige que todo caminho citado numa spec de
  implementação exista. A `.i` entra com o código; a ordem de entrega em §9 é o
  que existe até lá.

## 2. O princípio

**A divisão do trabalho é julgamento; a segurança dela não é.**

Tudo neste documento decorre disso. O pai escolhe quantos filhos, com que
tarefa, possuindo quais caminhos — e nenhuma dessas escolhas é o que impede uma
árvore corrompida. O que impede é o scheduler, a contenção e o orçamento, que
já valem hoje para chamada de ferramenta e passam a valer igual para filho.

Um pai que divida mal produz trabalho ruim, nunca dano.

## 3. A forma da chamada

A ferramenta `explore` de hoje declara `task` e `path`, e **não** declara modo —
a ausência é a garantia (RN-11 do `agent-loop.r`).

A escrita não acrescenta um modo. Acrescenta **propriedade**:

```jsonc
{
  "task":  "Catalogue this repository's architecture into ARCHITECTURE.md",
  "path":  "repos/billing",        // onde olhar, como hoje
  "owns":  ["repos/billing/ARCHITECTURE.md"]  // o que pode escrever
}
```

- `owns` **ausente** é o comportamento de hoje: filho somente-leitura. Nada
  muda para quem já usa.
- `owns` **presente** pede um filho escritor, e o pedido é um **teto que só
  estreita**: o conjunto é interceptado com o que o pai já pode escrever, nunca
  somado a ele.
- `owns` **vazio** não é "tudo". É erro de declaração, como escrita sem caminho
  já é hoje.

Não há campo de modo, não há campo de concorrência, não há campo de orçamento.
Nenhum dos três é do modelo.

## 4. Como a segurança já está construída

Nada abaixo é mecanismo novo. É o mesmo mecanismo, alcançando um filho.

| Preocupação | Onde já vive | O que muda |
|---|---|---|
| dois filhos sobre o mesmo caminho | `Schedule`/`conflicts`, por caminho declarado | o filho declara `owns`; a mesma função serializa |
| filho escreve fora do que possui | contenção por `Access` resolvido | workspace do pai **interceptado** com `owns` |
| filho pede aprovação | ADR-02: aprovação não se herda | escrita que escalaria é **negada e reportada**, como leitura já é |
| desfazer a delegação | `undo` por turno, foto antes da 1ª escrita | fotos do filho entram no conjunto do **turno do pai** |
| custo | tokens do filho debitados do pai | teto de **concorrência** também passa a ser da sessão |
| o que o filho fez | `DelegateResult.Read` | ganha `Wrote`, pelo mesmo motivo |

## 5. Coerência é do pai, e só dele

Caminhos disjuntos impedem corrupção, não incoerência (RN-7 da `.r`).

Portanto: **a definição de pronto roda uma vez, sobre a árvore inteira, depois
de os filhos terminarem, no turno do pai.** Não roda por filho. Um filho que
rodasse a suíte estaria conferindo uma árvore que ainda vai mudar, e um verde
sobre árvore intermediária é pior que nenhum — é o verde que ninguém confere de
novo.

Consequência aceita: um filho não sabe se o que escreveu compila. Ele não é
quem responde por isso.

## 6. Contratos comportamentais

A parte mediada por modelo é só uma: **decidir como dividir.** Três cenários,
com limiar, no formato das famílias existentes.

| Contrato | Cenário | O que se mede | Limiar |
|---|---|---|---|
| `delegates-writing-when-work-is-disjoint` | catalogar N repositórios em N arquivos, um por repositório | emite N filhos com `owns` disjunto, numa mensagem | ≥ 90% |
| `keeps-writing-that-must-be-coherent` | mudar uma interface e quem a chama | **não** divide entre filhos — faz no próprio turno | ≥ 95% |
| `reports-a-child-that-did-not-answer` | um filho falha entre N | diz qual, em vez de resumir os N−1 | ≥ 95% |

O segundo é o que importa e é o mais caro de acertar. O risco desta feature não
é o filho escrever errado — é o pai **dividir o que não se divide**. Um limiar
alto ali é o contrapeso ao ganho fácil do primeiro.

O terceiro mede a forma de defeito que este repositório não para de encontrar em
si mesmo: devolver N−1 calado é resultado incompleto com cara de completo.

## 7. Falha parcial

Com N filhos, N−1 respostas é caso normal, não excepcional.

- O relatório do pai nomeia **cada filho que não respondeu**, e por quê.
- Um filho que falhou **não desfaz** os outros. Desfazer é decisão do pai, e o
  `undo` do turno já alcança tudo.
- Um filho negado por contenção reporta a negação, nunca a esconde.

## 8. Invariantes verificáveis

- `owns` ausente produz um filho somente-leitura, idêntico ao de hoje.
- Sessão somente-leitura não produz filho que escreve.
- `owns` vazio é erro de declaração, nunca permissão total.
- Dois filhos que possuem o mesmo caminho declaram conflito antes de qualquer um rodar.
- Filho que escreve fora do que possui é negado pela contenção, não por revisão.
- Posse é por componente de caminho, nunca por prefixo de string.
- Possuir caminho fora do workspace não o traz para dentro.
- Estreitar um filho não estreita o pai.
- Filho que escreve não carrega ferramenta opaca nem a de delegar.

- As escritas do filho entram no conjunto de desfazimento do turno do pai.
- O desfazimento guarda a fotografia que o pai tirou primeiro, nunca a do filho.
- O relatório do filho nomeia os caminhos que escreveu, além dos que leu.
- Filho que não respondeu é nomeado, nunca resumido.
- Filho nunca pede aprovação, e escrita recusada é reportada como escrita.
- O teto de concorrência é da sessão; nenhum pedido do modelo o amplia.
- Tokens de filho que escreve são debitados do pai, como os de filho que lê.
- Filho não carrega critério de pronto, e é dito a ele que conferir não é seu trabalho.
- A descrição da ferramenta não nega a capacidade que o schema oferece.

## 9. Ordem de entrega

Três PRs, e o primeiro vale sozinho.

1. **`owns` e a contenção estreitada.** Um filho, escrevendo, dentro do que
   declarou. Sem paralelismo novo: o scheduler já paraleliza. É aqui que mora
   toda a segurança, e é o PR que precisa da revisão mais dura.
2. **Relatório e desfazimento.** `Wrote` no resultado, fotos do filho no turno
   do pai, filho que não respondeu nomeado.
3. **Contratos comportamentais.** Os três da §6, medidos. Vêm por último porque
   custam chamada de modelo e porque medir contrapeso antes de o peso existir é
   medir outra coisa.

## 10. O que este desenho não resolve

- **Conjuntos que se cruzam por desenho.** Serializados. Worktree e
  reconciliação quando houver caso, e ainda não há.
- **Filho assíncrono.** Fora de escopo por falta de caso, não por dificuldade
  (§5 da `.r`).
- **Relatório estruturado.** O filho devolve prosa. Um catálogo de dez
  repositórios quer estrutura, e essa é a pergunta em aberto que a `.r` já
  registra.

## 11. Changelog

_Sem alterações desde a criação._
