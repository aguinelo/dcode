# Uma fronteira aninhada é detectada, não adivinhada

**Data:** 2026-08-19

## O que mudou

O backend do macOS passa a **provar** que consegue aplicar um perfil antes de se
declarar disponível, como o do Linux já fazia. Sem fronteira, os testes que
pedem uma pulam dizendo o motivo, em vez de falharem.

## Por quê

Uma sessão do dcode rodando **dentro** da própria sandbox falhava seis testes de
fronteira de uma vez. O kernel do macOS recusa aplicar um perfil a partir de um
processo já confinado — `sandbox_apply: Operation not permitted` — e nada nessa
falha distingue "este ambiente não aninha" de "o trabalho está errado".

O agente que leu isso gastou a sessão inteira consertando o harness. O
diagnóstico dele estava certo e o conserto era razoável; o problema é que
ninguém pediu, e ele chegou lá porque a única saída aparente era essa.

**Uma mensagem de erro que não distingue ambiente de trabalho é uma mensagem que
manda o leitor para o lugar errado.**

## A assimetria que existia

O `bubblewrap.Available()` sondava desde que foi escrito, porque várias
distribuições restringem user namespaces e a falha resultante é opaca. O
`seatbelt.Available()` só olhava o `PATH`. Mesma classe de problema, um lado
tratado e o outro não — e o não tratado só apareceu quando alguém rodou o dcode
dentro do dcode.

## O risco que isto cria, e como fica coberto

**Teste de fronteira que pula lê como teste que passou.** É a razão pela qual
este job de CI existe, e agora há mais um caminho para pular silenciosamente.

Por isso a CI ganhou, no macOS, a mesma prova que já fazia no Linux: aplica um
perfil trivial e falha o job se não conseguir. Um runner que não consegue confinar
pularia todos os testes de fronteira e reportaria verde — exatamente o silêncio
que este job existe para quebrar.

## O que isto destrava

`make check` volta a ser executável de dentro de uma sessão do dcode neste
repositório. Sem isso, um agente aqui não conseguia rodar a definição de pronto
do próprio projeto, e trabalho entregue sem conferência é o que a definição de
pronto existe para impedir.
