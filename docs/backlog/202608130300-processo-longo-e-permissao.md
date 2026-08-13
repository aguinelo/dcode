# Processo longo e permissão

**Data:** 2026-08-13
**Origem:** uso real — "roda esse projeto", `npm start`, a sessão travou
**Estado:** três decisões tomadas; o desenho ainda não foi escrito

## O que aconteceu

Pedido para subir o servidor da aplicação. O modelo chamou `bash npm start`,
que é a coisa certa a chamar, e a sessão parou de responder até o comando ser
cortado.

Não é bug de implementação. É o desenho encontrando um caso que ele não tem:

```go
runCtx, cancel := context.WithTimeout(ctx, timeout)   // 120s por default
out, code, err := b.Runner.Run(runCtx, b.Workdir, in.Command)
```

`Run` é síncrono. Um servidor não termina — é essa a definição de servidor — então
o turno fica preso até o teto, e o teto existe justamente porque um comando que
não volta travaria a sessão para sempre. O timeout não é o problema; é o único
motivo pelo qual a sessão volta a responder.

## Por que não é só "adicionar uma flag"

Um `background: true` no `BashInput` resolve a chamada e cria quatro problemas
que o produto hoje não tem:

**Quem é dono do processo.** O turno acaba, a sessão acaba, o daemon reinicia — e
o `npm start` continua de pé. Um agente que deixa processos órfãos na máquina de
alguém é pior que um que trava, porque travar é visível.

**Onde a saída vai parar.** O valor de subir um servidor é ver o log quando ele
quebra. Saída que ninguém coleta é um servidor que subiu e não se sabe se
funcionou — o mesmo modo de falha que o selo de verificação existe para impedir.

**O que a política diz.** `Declare` reporta o que a chamada tocaria e a política
decide antes. Um processo que fica vivo depois do veredito pode fazer coisas que
o veredito não cobriu: abrir porta, escrever arquivo, alcançar rede. A aprovação
foi dada para um instante e valeria para uma duração.

**Como o modelo sabe que subiu.** Hoje ele lê `exit 0`. Um processo em segundo
plano não tem código de saída até morrer, e "subiu" é uma pergunta diferente de
"terminou bem" — normalmente respondida sondando uma porta ou um log.

## O que já existe e serve de peça

- `Runner` é injetado (`internal/tools/exec.go:24`), então trocar a execução não
  toca o resto da suíte.
- `sandbox.Runner` já confina, e um processo longo continua confinado.
- `policy.Evaluate` já separa fronteira de autorização; falta o eixo **duração**.
- O harness de eval não tem contrato sobre isso, e precisaria de um: "sobe um
  servidor e relata o endereço, sem travar" é comportamento, não asserção.

## As três decisões

Tomadas em 2026-08-13. Escritas aqui porque decisão que não fica escrita vira
discussão de novo.

**1. O processo morre com a sessão.** Fechar a TUI derruba o servidor, e isso é
aceito. A alternativa exigiria um registro durável de processos e um jeito de
matá-los depois — e produziria órfãos na máquina de alguém, que é pior que
travar porque travar é visível.

Isso resolve um problema que parecia separado: `kill` em PID alheio é negado
pelo Seatbelt, e continuará sendo. Quem mata é o daemon, de fora do sandbox,
sobre processos que ele mesmo registrou. O modelo nunca ganha autoridade de
sinal sobre a máquina — e não precisa, porque nada fica órfão.

**2. Aprovar um processo longo é o mesmo ato que aprovar um comando.** Sem
pergunta extra sobre duração. É defensável porque o processo não sobrevive à
sessão: a autorização e o processo têm o mesmo tempo de vida, então não existe
a janela em que alguém aprovou um instante e concedeu uma duração.

**3. O modelo consulta a saída; não a recebe.** `bash` devolve um identificador
na hora, e uma ferramenta de leitura entrega o que saiu quando o modelo pedir.

Receber exigiria um caminho do servidor para dentro do turno que o protocolo não
tem, e um servidor tagarela comeria a janela de contexto sem ninguém ter pedido.
Consultar é coerente com o ciclo de vida curto que a decisão 1 estabelece.

## O que ainda não foi decidido

- A forma da ferramenta de leitura: outra ferramenta (`logs`), ou o mesmo `bash`
  recebendo um identificador.
- Como o modelo sabe que o servidor **subiu**, que é pergunta diferente de
  "terminou bem". Sondar porta e ler log são as duas respostas óbvias e nenhuma
  é obviamente melhor.
- O contrato de eval que mede isso. "Sobe um servidor e relata o endereço, sem
  travar" é comportamento, não asserção, e hoje não existe.

## Relacionado

Recuperar sessão — mesma família, e as duas se cruzam na pergunta 1. Um processo
que sobrevive à sessão só faz sentido se a sessão puder ser reencontrada.
