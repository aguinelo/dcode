# qualifier-proposes-commands

**Contrato:** `202608261730-done-qualifier.p.spec.md` · limiar **95%**

Uma pasta de spec com prosa e nada mais: nenhum `tasks.md`, nenhum `done.toml`.
É o estado em que o `/loop` abre uma sessão **qualificadora** em vez de uma de
trabalho.

## O turno é o do produto, inteiro

Nada aqui é escrito à mão. A instrução vem de `tui.LoopTask` com `Qualify`
ligado — a mesma frase que o cliente manda —, a ferramenta é a `done_propose` do
registro do produto, a resposta que o modelo lê de volta é a de
`app.QualifyingTool`, e a fronteira sai de `app.QualifyMode`: leitura apenas,
porque descobrir como você vai ser medido é leitura.

Um `task.md` nesta pasta é **erro**, não alternativa. Duas instruções para o
mesmo turno é uma que diverge, e este pacote já foi mordido quatro vezes por
cópia de texto do produto.

## O que se mede

Que todo critério proposto seja um **comando**. "Lighthouse >= 95" é o que uma
pessoa escreve num quadro; `pnpm lhci --assert` é o que decide.

## O que este cenário ainda NÃO pega

Se o comando **roda**. O arcabouço não executa o que um modelo escreveu — essa
recusa é mais velha que este contrato e é deliberada —, então um critério que
nomeie um script inexistente passa aqui e voltaria 127 da medição do próprio
produto. O que o juiz pega é a falha que dá nome ao contrato: uma frase no campo
onde vai um comando.
