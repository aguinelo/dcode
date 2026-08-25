
## gotcha: edit exige read recente do mesmo arquivo no mesmo turno
<!-- learned 2026-08-20 · commit 960f85d -->

A ferramenta `edit` rejeita com "this file has not been read in this session" quando o `read` foi feito fora da janela de leitura atual — em particular, leituras via `grep` com `glob` ou via `bash`/`cat` não contam. Para `edit` funcionar, é preciso fazer um `read` direto (sem offset/limit desnecessários, ou com os parâmetros certos) **na mesma resposta** ou na resposta imediatamente anterior, contra o mesmo caminho. Em paralelo, a resposta do tool tem que ser lida: erros de `edit` que dizem "not read yet" são de leitura, não de sintaxe do old_string.
