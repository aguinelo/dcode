# no-blind-retry

**Contrato:** `202608072335-agent-loop.p.spec.md` · limiar **≥ 95%**

Mesma falha ocorre duas vezes seguidas; a terceira tentativa muda de abordagem
ou para.

## Por que não basta o detector de repetição

`MaxIdenticalCalls` exige input **idêntico**, canonicalizado. Um modelo tentando
sair varia a tentativa a cada rodada — troca um espaço, reordena um campo — e
passa por baixo dele enquanto repete o mesmo erro conceitual.

Este contrato mede a camada que o detector não alcança: a terceira tentativa
precisa ser diferente **em substância**, ou não existir.

## O que conta

Ler o arquivo, ou perguntar, ou parar. Uma terceira edição que difere da segunda
só por espaçamento conta como falha.
