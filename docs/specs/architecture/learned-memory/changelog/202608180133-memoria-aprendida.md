# Memória aprendida

**Data:** 2026-08-18

## O que mudou

Família de spec nova: `learned-memory`. **Nenhuma linha de código ainda** — é
desenho registrado antes de implementação, que é a ordem que este projeto segue.

## O buraco que ela fecha

Havia duas memórias e faltava a terceira:

| o que | responde | onde vive |
|---|---|---|
| gravação de sessão | o que **aconteceu** | `.dcode/state/*.jsonl` |
| cadeia de instruções | o que alguém **decidiu** | `AGENTS.md`, `DCODE.md` |
| **memória aprendida** | o que foi **descoberto** e vale guardar | `.dcode/memory.md` |

Sem a terceira, cada sessão redescobre que `make test` precisa de `go generate`
antes, que aquela decisão já foi tomada, que o repositório faz assim e não
assado.

## O princípio que resolve o desenho

**Escreve por ferramenta. Lê pelo prefixo.**

É a linha que decidiu o estado do repositório no prefixo (#171), aplicada de
novo. Fato que se precisa todo turno vai no prefixo; ato com consequência é
ferramenta. Lembrar é ato, ler é fato. Uma frase resolve as duas metades.

## As três decisões que carregam o resto

**A memória mora no repositório do usuário, versionada.** Não em banco, não em
serviço, não num wiki paralelo. Memória errada não fica parada — ela compõe: o
agente lê o próprio engano de volta como fato e age com mais confiança que na
primeira vez. Decaimento por desuso não conserta isso; **só alguém vendo no diff
conserta**. A revisão é o único portão de qualidade que este desenho tem, e todo
o resto do formato existe para não atrapalhá-la.

**Nada aprendido vence nada escrito por pessoa.** Fonte `learned` com autoridade
`0`, abaixo de `user`, sem chave de configuração que inverta. O mesmo raciocínio
que mantém `Safety` fora da sobreposição: garantia que uma configuração desliga
não é garantia. Sem isso, a memória é o caminho pelo qual o agente reescreve
devagar as próprias restrições.

**Sem consolidador ao fim da sessão.** O agente chama `remember` durante o turno,
à vista. Sem chamada de modelo que ninguém pediu, sem processo rodando depois que
a pessoa parou de olhar.

## O que foi olhado e recusado

O desenho de referência nesse espaço — daemon próprio, hooks de ciclo de vida,
MCP, consolidador LLM por sessão, decaimento por acesso — colide com três
decisões já registradas aqui: MCP fora, hooks de projeto fora, e nada que escreva
onde ninguém revisa.

A evidência mais forte contra veio do próprio autor daquele sistema: **seis
semanas de perda silenciosa de dados** por uma chave de hook errada. É o que
acontece quando o produto depende de um acoplamento que nenhum dos dois lados
verifica — a mesma forma de defeito que este repositório encontra desde o
começo.

O que vale copiar de lá, e já era doutrina aqui: armazenamento simples, índice
simples, nada de banco vetorial.

## O que a spec deixa em aberto de propósito

**O teto de 40 memórias no prefixo é valor inicial, não número defendido.** Não
há observação atrás dele, e o `.config` diz isso com todas as letras. Fixar teto
por raciocínio e nunca revisitar é o erro que `EVAL_TIMEOUT` cometeu duas vezes,
matando uma corrida em 180m e outra em 480m.

**Os limiares dos três contratos comportamentais não estão declarados.** O
primeiro número honesto vem da primeira medição. Limiar antes de medição é
limiar que a medição depois justifica.

**Se o modelo vai usar a ferramenta, ninguém sabe.** É o mesmo risco que fez o
estado do git virar prefixo em vez de ferramenta. O contrapeso previsto é
lembrete de Camada 2 — e ele é o **último** passo do `.i`, porque não se constrói
contrapeso antes de medir o peso.

## O contrato que mede a ausência

Dos três contratos previstos, o que mais importa é **`does-not-remember-activity`**:
uma sessão comum, que só leu e editou, não deve gravar nada.

Memória escrita por hábito é exatamente como esse mecanismo vira ruído, e um
contrato que mede a ausência vale mais que um que mede a presença.

## Duas coisas que os guardas de spec pegaram na hora

Escrever a spec fez dois guardas dispararem, e os dois estavam certos.

**`## Invariantes verificáveis` significa "há teste cobrando isto".** O
`specguard` lê exatamente esse título para exigir um guarda por família. Uma spec
sem código não tem o que verificar, então a seção se chama **`Invariantes a
garantir`** e renomeá-la é o passo 0 do `.i` — junto com o guarda, para que cada
invariante seja cobrada no mesmo commit que a implementa.

**Caminho entre crases num `.i` é afirmação de que o arquivo existe.** Os
caminhos aqui estão sem crase de propósito; ganham crase quando forem escritos.

Nenhum dos dois foi contornado. Os títulos e as crases é que estavam errados —
afirmavam garantia onde havia só intenção, que é a forma de defeito que esta
spec inteira existe para não repetir.
