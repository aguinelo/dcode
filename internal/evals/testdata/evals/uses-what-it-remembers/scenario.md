# uses-what-it-remembers

## O que mede

Que o agente **age** sobre uma memória que já está no prefixo, em vez de
redescobrir a mesma coisa ao mesmo custo.

## O material

`.dcode/memory.md` traz uma `gotcha`: `make build` quebra sem `make generate`
antes. A tarefa pede para adicionar um método e garantir que o pacote ainda
compila.

O bloco chega ao prefixo pelo **leitor do produto** — `memory.Read` e
`memory.Render` —, nunca por um texto copiado para a fixture. Fixture que copia
texto do produto é fixture que diverge dele, e esta suíte já pagou isso quatro
vezes; a pior foi um lembrete cuja cópia truncada tinha perdido justamente a
cláusula que o juiz media.

## O juiz

`CalledWith("bash", "generate")`.

Agir sobre a memória **é** uma chamada de shell. O harness recusa executá-la, e
isso não muda o que o modelo escolheu fazer — a escolha é o que se mede.

## Por que este oferece shell

É a exceção declarada em `shellIsPartOfTheTask`, com motivo: aqui a chamada de
shell é a medição, não ruído em volta dela.

## Limiar

Declarado como ≥ 0% — que significa **"meça e me diga"**, não "qualquer coisa
serve". O primeiro número honesto vem da primeira medição, e limiar antes de
medição é limiar que a medição depois justifica.
