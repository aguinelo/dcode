# A primeira fatia, planejada

O `.r` aprovado em 2026-10-17 (#514) ganhou `.p`, `.config` e `.i` para a primeira fatia da §10
dele — o núcleo observável. Nada é construído aqui; o que segue são as decisões que o `.p` toma onde
o `.r` deixava a forma em aberto, cada uma com o porquê.

## O que o `.p` decide

- **A esteira é do harness, não do modelo.** O `.r` §6 listava `review`, `gate` e `merge` como
  ferramentas da orquestradora. Viraram passos que o harness roda em ordem ao fim `done` de um
  filho: commit e `owns`, revisão, atualização com merge da main, gate no commit exato, PR, pergunta,
  merge serial. O `.r` §2 põe do lado determinístico tudo que decide a entrada na main, e um modelo
  que chama o gate é um modelo que pode não chamar.
- **O filho não faz git que grava.** A worktree de um filho grava objetos no `.git` comum do
  repositório; abrir isso ao sandbox abriria também os hooks e a config — o mesmo `core.hooksPath`
  que sumiu três vezes em 2026-10-16. O harness faz commit (com caminhos nomeados, nunca `-A`),
  merge e push; o `bash` do filho tem esses comandos negados, dizendo o porquê.
- **A orquestradora não escreve na árvore.** Sessão em modo `plan` na raiz do projeto: só filhos
  escrevem, em worktree, e "merge só do que foi medido" fica sem exceção.
- **`POST /v1/projects` cria; o `GET` só lê.** O `.r` §5 dizia "criada na primeira vez" num `GET`;
  uma leitura que cria estado faria uma sonda abrir uma orquestradora.
- **`Role` no lugar de um booleano `orchestrator`.** O `.r` §5 listava `orchestrator` entre os campos
  novos; há três papéis (orquestradora, filho, revisor), e um booleano não diz o terceiro.
- **Reabrir sob o mesmo id.** O `Resume` de hoje cria sessão nova; a orquestradora e os filhos
  voltam do reinício com o id que o `parent_id` dos outros guarda, e o `Seq` continua.
- **Devolução com teste vermelho em duas fases.** "Confirmar que o teste novo falha sem o conserto"
  (RN-5) vira uma fase `red`, que termina com `red_test{command}` e só passa com saída diferente de
  zero, antes da fase `fix`.
- **Merge local por troca condicional.** Sem forge, `git update-ref` com o valor antigo é a trava do
  commit medido; e é recusado com a main em checkout em qualquer worktree, porque a integração nunca
  mexe na árvore de alguém.
- **Os oito estados da RN-9 bastam.** Revisão, atualização e gate cabem em `in_review`; o passo
  corrente se lê pelo que está preenchido.
- **Custo é tokens nesta fatia.** O produto não tem tabela de preços; dólares entram com o teto de
  custo, que precisa da mesma tabela.
- **O comando é `dcode decisions`**, e lê o arquivo sem daemon: o log tem de ser legível quando o
  que falhou foi o daemon.
- **As invariantes são previstas.** A seção do `.p` tem um título que o `specguard` não lê, até o
  primeiro passo com teste; uma invariante sem teste não pode reprovar nada, e fingir que reprova é
  pior.

## O que fica para a segunda fatia

`launchd`, supervisionado e autônomo, a lista "sempre perguntar" como negação, os tetos, o deploy,
as notificações e os sete contratos com modelo real. As chaves dos tetos, da autonomia e do deploy
estão declaradas no `.config` §5 e não são lidas.

## Ordem

O `.r` §10 tinha doze passos; o `.p` §14 tem quinze, porque reabrir sob o mesmo id, o relato que
abre o turno, o revisor e o comando ganharam PR próprio — cada um, junto de outro, pediria "and" no
título. O porquê de cada diferença está na `.i`.
