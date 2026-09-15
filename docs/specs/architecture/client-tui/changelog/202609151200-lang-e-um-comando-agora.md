# `/lang` é um comando agora

**Data:** 2026-09-15
**Specs afetadas:** `202608081250-client-tui` (`.p`, seção 8)
**Fonte:** pedido do usuário — depois de confirmar que a interface já é
bilíngue (`internal/tui/lang.go`) e que o modelo já responde na língua de
quem escreveu (doutrina, `internal/behavior/behavior.go:328`), pediu
explicitamente a troca ao vivo: *"quero /lang"*.

## O que mudou

`/lang [en|pt-BR]` — sem argumento mostra o idioma em vigor; com um
argumento válido, troca; com um inválido, recusa nomeando o que foi digitado
(mesma forma de `/mode`). Aceita qualquer grafia que o ambiente aceitaria em
`DCODE_LANG` — `pt_BR.UTF-8`, `pt-br`, `en_US` — porque reaproveita a mesma
`parseLang` que já resolvia a variável de ambiente.

## Por que foi pequeno

`Lang` nunca foi lido por `internal/app` nem por `internal/behavior` — é
puramente como este processo cliente desenha o próprio cromo. O comando
troca `p.model.Lang` e retorna, sem `newSession`, sem tocar o daemon. `/model`
e `/mode` existem porque mudam o que o daemon faz; `/lang` existe porque muda
o que a TELA diz, e por isso não precisa de nenhum dos dois.

A confirmação sai na língua **para a qual** se trocou, não na de origem — a
mesma razão pela qual uma negação nomeia o que foi negado em vez de repetir o
pedido.

## O que já existia, e não foi tocado

O sistema inteiro — catálogo de ~250 strings em duas línguas
(`internal/tui/lang.go`, 784 linhas), resolução por `DCODE_LANG`/`LC_ALL`/`LANG`,
`ui.lang` em `config.toml`, e a regra na doutrina que faz o modelo responder
na língua de quem escreveu — já existia antes deste pedido. `/lang` não é a
função bilíngue; é só o jeito de trocar sem editar arquivo nem exportar
variável. Essa spec nunca documentou o sistema (nenhuma seção o menciona antes
desta linha da tabela) — lacuna anterior a esta mudança, registrada aqui e não
fechada: documentar o catálogo inteiro é trabalho à parte.
