# Socket é alcançável onde já se escreve

A correção anterior (#196) fechou o socket do runtime de contêiner negando
**todo** socket unix e todo tráfego que não fosse de saída. Fechou junto duas
coisas que o produto precisa.

## O que quebrou

`(allow network-outbound ...)` sozinho não permite **escutar**. Toda porta
aberta dentro do sandbox passou a falhar em `bind: operation not permitted` —
`httptest.NewServer`, e com ele os testes de `internal/server`, `internal/tools`,
`internal/update`, `internal/app` e `pkg/client`. E negar todo socket unix
fechou também o socket do próprio daemon do dcode.

Ou seja: a regra escrita para manter um daemon de fora manteve a suíte de fora
junto. É a segunda vez nesta série que consertar a sandbox quebra rodar os
testes, e o motivo é o mesmo das duas vezes — a fronteira foi desenhada olhando
para o que se queria negar, não para o que já se tinha permitido.

## Quem encontrou

Uma sessão não assistida, na mesma tarefa da rodada anterior. Ela terminou em 86
rodadas, com o trabalho feito, e **recusou-se a declarar sucesso**: reportou
`incomplete`, nomeou os cinco pacotes, mostrou que a falha era `bind` e não
regressão — revertendo a própria edição e reproduzindo o mesmo erro — e escreveu
"I have not weakened the check, and I am not reporting success."

O diagnóstico estava certo e a causa era nossa.

## A regra que passou a valer

**Socket unix é alcançável exatamente onde já se pode escrever.**

Não é uma fronteira nova ao lado da que existia: é a mesma. Um socket que o
processo pode criar é um socket com que ele pode falar, e `/var/run` — onde um
runtime de contêiner escuta — não é lugar onde `workspace-write` escreve. O
conjunto gravável passa a ser montado uma vez e lido duas, pelas regras de
escrita e pelas de socket, para que não existam duas respostas sobre a mesma
fronteira.

`read-only` não escreve em lugar nenhum, então não alcança socket unix nenhum.

Escutar volta a ser permitido quando a rede é concedida, em IP local. Escutar
não é alcançar ninguém: é ficar disponível para quem já está dentro.

## O que continua fechado

O socket do runtime de contêiner. Medido, com o perfil de `workspace-write` e a
rede concedida: porta local abre, socket unix dentro do workspace responde,
socket unix fora de todo lugar gravável é negado, e o daemon do Docker é negado.
