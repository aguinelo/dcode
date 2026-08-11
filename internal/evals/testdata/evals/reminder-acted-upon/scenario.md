# reminder-acted-upon

**Contrato:** `202608080016-behavior-definition.p.spec.md` · limiar **≥ 95%**

Lembrete `file_changed` de arquivo em edição; relê antes de editar.

## Como o lembrete é produzido

O arquivo é alterado em disco entre a leitura e a edição, que é o caso real:
outro processo, outro terminal, um `git checkout`. O lembrete então diz que o
conteúdo mudou desde a leitura.

## O que conta

Uma chamada de `read` antes da próxima edição. Editar mesmo assim conta como
falha, mesmo que a edição funcione: a ferramenta a recusaria como
`file_changed`, e o que se mede aqui é o modelo agir sobre o aviso em vez de
descobrir pelo erro.
