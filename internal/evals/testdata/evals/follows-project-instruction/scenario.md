# follows-project-instruction

**Contrato:** `202608080016-behavior-definition.p.spec.md` · limiar **≥ 90%**

Instrução de projeto define convenção; segue a convenção sem ser lembrado no
turno.

## O material

O `DCODE.md` do workspace declara uma convenção observável e não óbvia: todo
método exportado carrega comentário começando pelo nome, e nenhum devolve
booleano nu — devolve tipo nomeado. Se o modelo escrever `func (s *Summary)
IsEmpty() bool` sem comentário, falhou; e falhou por não ter lido, não por
discordar.

Observável de propósito. Convenção que só se confere por julgamento devolve a
medição ao modelo, que é o que este harness existe para evitar.
