# A primeira vez que o trabalho foi dividido

Primeira medição da delegação com escrita ponta a ponta, contra modelo real.
**Funcionou.**

## O número, com as três formulações lado a lado

Mesma tarefa, byte por byte, nas três: cinco notas de arquitetura, uma por
pacote, cada uma no seu arquivo, dito na tarefa que não dependem entre si.

| | descrição negava a escrita | descrição corrigida | descrição liberada |
|---|---|---|---|
| `explore` | **0** | **0** | **5** |
| `write` no pai | 5 | 5 | **0** |
| iterações | 104 | 37 | **29** |
| tokens de entrada | 5,8 M | 1,8 M | **1,3 M** |

As cinco chamadas saíram em **linhas consecutivas do log** — emitidas numa
mensagem só, portanto agrupadas pelo scheduler e executadas lado a lado, porque
os `owns` eram disjuntos. É o desenho da §4 do `.p` acontecendo sem que ninguém
o orquestrasse.

## O que a sequência ensina

A descrição da ferramenta não é documentação, é **superfície de comportamento**.

- Enquanto ela **negava** a capacidade, o modelo não a usou — e não a mencionou
  uma única vez em duas rodadas.
- Corrigir a contradição melhorou a **eficiência** em 3× e não moveu a delegação.
- O que moveu foi tirar a orientação de custo escrita para a era somente-leitura:
  *"não delegue o que você já leu"*. Ler primeiro é como todo trabalho de escrita
  começa, então aquela frase desaconselhava exatamente o caso que a feature
  existe para servir.

Três formulações, três medições. Nenhuma linha de lógica mudou entre a segunda e
a terceira — só a frase que o modelo lê antes de decidir.

## A anomalia, registrada sem explicação

Cinco arquivos foram escritos por cinco filhos. **Quatro relatórios trazem
`wrote:`; um não.** O filho de `internal/sandbox` descreveu em prosa o arquivo
que escreveu, o arquivo existe, e o rodapé do relatório — `looked at:` e
`wrote:` — não saiu para ele.

O caminho do código foi conferido e está certo: o rodapé é acrescentado **depois**
da conclusão, então truncamento da conclusão não o remove. Sobra a hipótese de
que `ReadPaths()` e `Written()` daquele filho voltaram vazios, e ela não foi
confirmada.

Fica aberto, sem palpite. Um relatório que às vezes não diz o que foi escrito é
pior que um que nunca diz, porque o silêncio passa por resposta.

---

## Correção: a anomalia era a tela

Registrado acima como defeito sem explicação, e não era defeito.

`internal/app/app.go:1017` corta a **exibição** em 12 linhas e escreve
`… N more lines`. Os rodapés `looked at:` e `wrote:` vêm no fim do relatório do
filho, então qualquer relatório mais comprido perde o rodapé **na tela**.

`indent()` é chamada num lugar só, escrevendo na saída do CLI. **O modelo recebe
o relatório inteiro.** Uma rodada seguinte, com relatórios mais longos, perdeu os
cinco rodapés em vez de um — o que confirma a explicação em vez de aprofundar o
mistério.

Fica registrado porque a hipótese errada estava no `ROADMAP.md`, e roadmap que
manda alguém caçar um defeito que não existe custa mais que silêncio.
