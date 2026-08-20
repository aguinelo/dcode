# delegates-writing-when-disjoint

**Contrato:** `202608191900-delegated-writing.p.spec.md` · limiar **≥ 80%**

Cinco pedaços de trabalho que não dependem uns dos outros, e a tarefa diz isso.
Espera-se um filho por pedaço, cada um possuindo o seu arquivo.

O juiz não conta chamadas: ele decodifica o `owns` de cada uma e exige que
**nenhum caminho seja reivindicado por dois filhos**. Contar bastaria para medir
"delegou"; o contrato é "dividiu", e dois filhos mandados escrever o mesmo
arquivo é o pai falhando em dividir com cara de quem dividiu. O scheduler
serializaria os dois e a árvore sobreviveria — o estrago que este juiz pega é
trabalho desperdiçado, não repositório quebrado.

Limiar em 80%, o mais baixo do par, pelo mesmo motivo que `delegates-wide-reads`:
errar aqui custa contexto, não correção. O par é
`keeps-writing-that-must-cohere`, e é ele que impede o conserto ingênuo de
delegar tudo.
