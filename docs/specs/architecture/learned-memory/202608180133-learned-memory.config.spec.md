# Config: Memória aprendida

Toda chave desta tabela é lida por código deste componente. Chave declarada e não
lida é defeito, e é o defeito que este projeto mais encontrou.

## 1. Chaves

| chave | variável | tipo | padrão | o que faz |
|---|---|---|---|---|
| `memory.enabled` | `DCODE_MEMORY_ENABLED` | bool | `true` | Lê e escreve memória aprendida. Desligado, o produto é o de antes deste componente. |
| `memory.max_entries` | `DCODE_MEMORY_MAX_ENTRIES` | int | `40` | Quantas memórias entram no prefixo, as mais recentes primeiro. |

## 2. Sobre o teto de 40

**É um valor inicial, não um número defendido.**

Não há observação atrás dele. Foi escolhido porque quarenta blocos curtos ficam
na ordem de grandeza da cadeia de instruções que já vai no prefixo, e porque um
teto tem de existir antes que haja uso — sem teto, a primeira sessão que gravar
demais come a janela e ninguém descobre por quê.

O que fazer com ele: **medir e mexer**, com changelog dizendo o que foi observado.
Fixar um teto por raciocínio e nunca revisitá-lo é o erro que `EVAL_TIMEOUT`
cometeu duas vezes, matando uma corrida em 180m e outra em 480m.

## 3. Constantes não configuráveis

Estas não são chaves, e a ausência é a decisão.

**A ordem da fonte aprendida.** `learned` abaixo de toda fonte humana, sempre. Uma
garantia que uma configuração pode desligar não é garantia — o mesmo raciocínio
que mantém `Safety` fora da sobreposição de doutrina.

**A lista de tipos.** `gotcha`, `decision`, `convention`. Tipo novo é mudança de
spec, com o motivo escrito, e não uma linha em `.toml` que ninguém revisa.

**O caminho do arquivo.** `<workspace>/.dcode/memory.md`. Caminho configurável é
memória que uma máquina lê e outra não, para a mesma checagem do mesmo repo.

**Não existe chave de decaimento.** RN-10 fecha isso: frequência de acesso mede o
que o agente por acaso precisou, não o que é verdade.

## 4. Changelog

- [202608180133 — Memória aprendida](changelog/202608180133-memoria-aprendida.md)
