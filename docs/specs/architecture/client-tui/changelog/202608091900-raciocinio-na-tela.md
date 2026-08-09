# Raciocínio do modelo na tela

**Data:** 2026-08-09
**Specs afetadas:** `202608081250-client-tui` (`.p`, `.config`), `202608072240-client-server-protocol` (`.p`)

## O que mudou

Novo evento `message.reasoning` no protocolo, e `KindReasoning` no fluxo do
cliente. Nova **RN-18**: o pensamento aparece na tela e **nunca** no histórico.

Nova chave `behavior.show_reasoning`, default ligado.

## Por que agora

Quando a RN-10 foi escrita, o raciocínio era descartado no loop — corrigia-se um
vazamento, e mostrar era feature. Esta é a feature.

O valor é ver **por que** o agente está prestes a fazer algo, antes de ele
fazer. É o que transforma "ele travou?" em "ele está lendo o handler, faz
sentido".

## O desenho, e o número que o decidiu

Medido nos fixtures reais antes de projetar:

| turno | raciocínio | resposta |
|---|---|---|
| pergunta em prosa | 258 car. | 444 car. |
| com chamada de ferramenta | 96 car. | 19 car. |
| outro com ferramenta | 143 car. | 13 car. |

Em turno com ferramenta o raciocínio é **5 a 11 vezes** a resposta. Isso decidiu
tudo:

- **Aberto transmite só a cauda** (4 linhas). O que ele pensa agora é o que
  importa, mesmo motivo pelo qual o fluxo acompanha o próprio fim.
- **Fechado é uma linha.** Deixado aberto, o pensamento enterra o resultado a
  que levava.
- **Desligável.** Não por preferência de tela: cada fragmento é um evento, e o
  log tem teto por sessão. Raciocínio compete por orçamento com tudo que vale
  replay.

## Por que evento, e não só stream ao vivo

Considerado transmitir sem registrar, para poupar o log. Descartado: a
igualdade entre **replay e observação ao vivo** é invariante do protocolo, e
quebrá-la por economia de espaço é o tipo de exceção que torna um protocolo não
confiável. Desligar é a forma honesta de economizar — desligado, ninguém vê nos
dois casos, e a igualdade continua valendo.

## Impacto

- A RN-10 do provider é reforçada, não afrouxada: o loop **encaminha e nunca
  anexa**. Um teste trava as duas metades na mesma execução.
- `Entry` ganha `StartedAt` e `Closed`; a duração do pensamento é medida pelo
  cliente, como o tempo do turno e pelo mesmo motivo — precisa avançar entre
  eventos.
- Os defaults de `behavior.*` passam a viver na camada de configuração, e não só
  no código que os lê, de modo que `--config behavior.show_reasoning` responde
  valor **e** procedência.

## Encontrado ao construir

Duas atribuições no wiring falharam em silêncio: o campo existia em `Options`,
nada o preenchia e nada o passava adiante. A suíte inteira passava, porque cada
camada estava correta e nenhum teste atravessava as três. O sintoma foi zero
evento `message.reasoning` no fio, e quem apontou foi a verificação contra o
modelo real, não os testes.
