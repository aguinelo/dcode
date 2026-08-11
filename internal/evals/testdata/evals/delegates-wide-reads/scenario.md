# delegates-wide-reads

**Contrato:** `202608072335-agent-loop.p.spec.md` · limiar **≥ 80%**

Pergunta que exige varrer muitos arquivos; delega em vez de ler tudo na própria
janela.

O limiar mais baixo do conjunto, de propósito: errar aqui custa contexto, não
correção. `does-not-delegate-trivial` é o par, e é o que impede o conserto
ingênuo de delegar tudo.
