# A verificação não é um segundo mecanismo

**Data:** 2026-08-11
**Specs afetadas:** `202608072335-agent-loop` (`.r`, `.p`), `202608080016-behavior-definition` (`.p`, `.config`, `.i`)

> **Regra:** a reentrada por verificação é a reentrada da RN-10 — enquanto houver
> progresso, até `MaxStallCycles` ciclos parados. Não existe teto de "uma vez
> por turno".

## O que estava contraditório

A RN-9 nasceu com um teto próprio: *"o lembrete é anexado e o ciclo continua —
uma vez por turno"*. Fazia sentido quando ela era o único mecanismo.

`202608102100` generalizou o caso de um critério para N, e a própria RN-9 ganhou
a nota que registra isso:

> **Esta é a instância unitária da RN-10.** A verificação é uma lista de
> critérios com um item, e os dois mecanismos não coexistem — a reentrada, a
> medida de progresso e o encerramento honesto são os da RN-10.

O que não foi feito na mesma passagem: apagar o teto antigo do corpo da regra e
das linhas que o repetiam. A spec passou a afirmar as duas coisas, em parágrafos
vizinhos — uma vez por turno, e enquanto houver progresso.

O código nunca implementou o teto antigo. `checkDone` conta ciclos sem progresso
e encerra em `StopIncomplete`, que é a RN-10. Então a contradição não estava
entre spec e código: estava dentro da spec, e o efeito é pior que estar errada
por inteiro. Uma spec que se contradiz não é conferível — quem lê escolhe a
metade que confirma o que já achava, e as duas metades têm respaldo textual.

Foi encontrada exatamente assim: o mapeamento de invariantes buscou o teste que
sustenta *"a continuação dispara no máximo uma vez por turno"* e o teste que
existe assere o contrário — reentrada até `MaxStallCycles`.

## O que muda

Cinco lugares deixam de repetir o teto revogado: o corpo da RN-9, um invariante
do `.p` do agent-loop, um invariante do `.p` de behavior-definition, uma linha
da tabela do `.config` e um passo do `.i`.

Nenhuma linha de Go muda, porque nenhuma linha de Go implementava o que sai.

## O que não muda

O teto continua existindo — é `MaxStallCycles`, e a diferença entre os dois não
é de grau. "Uma vez por turno" conta **tentativas**; "enquanto houver progresso"
conta **resultado**. O primeiro corta um turno que está avançando a cada ciclo;
o segundo deixa avançar e corta quando parou de avançar.

E o motivo do teto antigo continua valendo, atendido pelo novo: sem teto algum,
projeto cuja verificação não roda gira até o teto de iterações da RN-2 — a
defesa errada para o problema errado.
