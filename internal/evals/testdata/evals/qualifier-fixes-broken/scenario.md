# qualifier-fixes-broken

**Contrato:** `202608261730-done-qualifier.p.spec.md` · limiar **85%**

A mesma pasta de `qualifier-proposes-commands`, uma volta depois: o `done.toml`
que a qualificação anterior escreveu está lá, e o único critério dele voltou
`broken` com saída 127 — a ferramenta que ele nomeia não existe na máquina.

## Este cenário só é alcançável por causa de uma correção

Antes dela, um critério quebrado era gravado **declarado**. O arquivo é o que a
próxima execução carrega, então a sessão de trabalho passava a ser medida contra
um comando que não existe — vermelho para sempre, laço que nunca termina — e a
pasta passava a declarar um critério, então nunca mais voltava para a
qualificação. Dois becos sem saída de uma linha só.

Gravado comentado, ele fica visível para quem revisa, fica fora da `DoneSet`, e
a pasta segue pendente sem declarar nada — que é o que a manda de volta para
esta fase. O `done.toml` desta pasta é saída literal do `Render` do produto, e
um teste o reconstrói e compara.

## O que se mede

Que o nome sobreviva e o comando mude. As duas formas de errar são opostas:
apagar o critério deixa a spec medindo menos do que media, e propô-lo igual
redeclara um comando que já se mostrou incapaz de rodar qualquer coisa.

## O que este cenário ainda NÃO pega

Se o comando novo roda. Mesmo limite de `qualifier-proposes-commands`, pela
mesma razão.
