# Processo longo e permissão

**Data:** 2026-08-13
**Origem:** uso real — "roda esse projeto", `npm start`, a sessão travou
**Estado:** pauta aberta, nada decidido

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

## As perguntas que decidem, antes de qualquer tipo

1. **O processo morre com a sessão, ou sobrevive?** As duas respostas são
   defensáveis e levam a produtos diferentes. Sobreviver exige um registro
   durável de processos e um jeito de matá-los; morrer junto exige que o
   usuário aceite que fechar a TUI derruba o servidor.

2. **A aprovação de um processo longo é a mesma de um comando?** Uma aprovação
   que vale enquanto o processo viver não é a mesma coisa que uma que vale para
   uma chamada. Se for a mesma, o usuário aprovou uma duração sem saber.

3. **O modelo consulta a saída depois, ou recebe?** Consultar precisa de outra
   ferramenta (`logs`, ou `bash` com um handle). Receber precisa de um canal do
   servidor para o turno que não existe no protocolo hoje.

## Relacionado

Recuperar sessão — mesma família, e as duas se cruzam na pergunta 1. Um processo
que sobrevive à sessão só faz sentido se a sessão puder ser reencontrada.
