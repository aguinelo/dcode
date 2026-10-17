# Manifesto ilegível é dito

**2026-10-16** — um `package.json` ou `Makefile` que existe e não se lê deixa de
ser lido como "o projeto não declara portões".

## O defeito

`fromPackageJSON` engolia o erro de `json.Unmarshal` e devolvia nada, com um
comentário dizendo que um manifesto malformado "lê como não declarou nada, que é
o que uma declaração ilegível honestamente é". `fromMakefile` fazia o mesmo com o
erro do scanner. O prefixo então se calava sobre portões **exatamente** como se
cala para um projeto que não declara nenhum, e ninguém era avisado de que a
sondagem tinha olhado um arquivo que não conseguiu ler.

É a lição que esta família já registra no §7, aplicada de novo e esquecida no
próprio `Probe`: "não sondei" e "sondei e não há" não podem ler igual. Uma
declaração que ninguém conseguiu ler não é ausência de declaração — é uma falha,
e o `AGENTS.md` pede que toda falha seja explícita, com aviso visível.

## A correção

- `Probe` passa a devolver, além dos portões, os manifestos ilegíveis
  (`Unreadable`: o arquivo e o erro de leitura ou de parse). Arquivo ausente
  continua sem erro e sem linha nenhuma: ausência é ordinária.
- Um manifesto ilegível não apaga o outro. `package.json` quebrado ao lado de um
  `Makefile` legível ainda entrega os alvos do `Makefile`.
- Um `Makefile` que o scanner não termina não oferece o que leu antes da falha
  como se fosse a lista inteira.
- A sessão diz o problema ao abrir, pelo mesmo canal da memória ilegível:
  `EventSessionError` com código `manifest_unreadable`, nomeando o arquivo e o
  erro. Avisa; não recusa a sessão — um manifesto quebrado é problema do projeto,
  não motivo para parar o trabalho.

## O que não mudou

O prefixo continua sem bloco de portões quando nenhum foi lido. O aviso vai para
quem está na sessão, não para o modelo: dizer ao modelo que um arquivo não se
leu não muda o que ele pode fazer, e a frase entraria num prefixo que a cache
quer estável.
