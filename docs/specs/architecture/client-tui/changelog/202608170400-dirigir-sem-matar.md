# Dirigir o turno sem matá-lo

**Data:** 2026-08-17

## O que mudou

Palavra digitada durante um turno ativo **dirige** o turno: chega na próxima
rodada do laço, como a pessoa falando. Antes ficava na fila até o turno inteiro
acabar.

Caminho novo: `POST /sessions/{id}/steer` → `Session.Steer` → fila da sessão →
`loop.Config.Steer`, drenada no topo de cada rodada, antes da compactação. Sai
como `turn.steered` para o cliente e para a gravação.

## A invariante que mudou, e por quê

Saiu:

> Entrada durante turno ativo entra na fila, nunca é submetida ao servidor.

Ela existia porque o protocolo recusa turno concorrente. **Esse motivo continua
verdadeiro** — e não se aplica: uma correção não é um segundo turno, é a mesma
conversa continuando enquanto o trabalho acontece.

O que a fila custava: ver o turno indo pro lado errado na rodada 3 de 20 e ter
duas opções — deixar terminar errado, ou matar e perder tudo que ele aprendeu.
Gente assiste turno ruim até o fim por causa disso.

Entraram quatro no lugar, todas cobradas por teste:

- Palavra digitada durante turno ativo dirige; nunca abre um segundo turno.
- Embutido que custa um turno (`/init`, `/plan <texto>`) **continua na fila**.
- Imagem anexada segura a mensagem na fila.
- Correção recusada porque o turno acabou vira fila, nunca sumiço.

## As três distinções que o desenho carrega

**Embutido não é correção.** `/init` no meio de um turno não corrige o trabalho
em curso — é outra tarefa. Dirigir carrega o que a pessoa quer **mudado** no que
está acontecendo; embutido carrega outra coisa a fazer depois.

**Imagem segura o par.** O caminho de direção leva texto só. Mandar metade agora
seria perguntar ao modelo sobre algo que ele não pode ver.

**Correção não é lembrete.** Ela entra no histórico como `RoleUser`, sem a marca
de lembrete. Lembrete é o produto falando; arquivar a correção da pessoa como um
seria atribuir mal a coisa mais importante que alguém diz durante um turno — o
modelo pesaria como maquinaria em vez de como a pessoa mudando de ideia.

## O que se perde, dito em voz alta

**Corrida real e inevitável:** o turno pode acabar entre a tecla e a requisição.
O cliente trata a recusa transformando em fila — que é o que teria acontecido se
a pessoa tivesse digitado um instante depois.

**O que não foi entregue morre com o turno.** Uma correção ainda na fila quando o
turno termina era para aquele turno; levá-la adiante entregaria "não, faça do
outro jeito" sobre trabalho que já acabou. É descartada e **dita**, via
`session.error`, nunca em silêncio.

## Duas recusas, não uma

`no_active_turn` é o espelho exato de `turn_already_active`: uma diz "espere", a
outra diz "não há o que corrigir — mande como mensagem". Juntar as duas deixaria
o cliente adivinhando qual delas era.
