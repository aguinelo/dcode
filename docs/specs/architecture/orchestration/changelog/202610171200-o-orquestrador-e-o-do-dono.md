# O orquestrador é o do dono

O `.r` desta família foi aprovado em 2026-10-13 (#461) e nada dele foi
construído. O dono o leu de novo e a visão dele é outra: não uma conversa que
reparte um pedido em filhos e junta os relatos, mas **o dcode como o
orquestrador do projeto** — o que responde a pessoa na hora, reparte por tema,
despacha agentes simples, mede, faz merge, faz deploy e mantém a lista de quem
trabalha, numa execução longa com o dono longe. O `.r` foi reescrito para dizer
isso, e volta a esperar aprovação.

## Por que agora

Em 2026-10-16 o fluxo inteiro foi feito à mão, com agentes do Claude Code em
worktrees. Seis coisas deram errado e cada uma virou regra (§1 e RN-12 do `.r`):
o topo do `CHANGELOG.md` e o número contado conflitaram em todo PR; o hook
sumiu três vezes com `core.hooksPath` reescrito para caminho absoluto; agentes
disseram "o hook passou" sem prova; um force-push foi negado depois de um
rebase; integrar na worktree da pessoa recarregou o app que ela usava; e
decisões chegaram no meio do trabalho, parando a fila.

## O que caiu da #461, e por quê

- **"Fechar o pai encerra os filhos" e o pai como sessão comum.** A orquestradora
  agora é uma por projeto, viva no daemon e persistida em disco (RN-1, RN-2).
  Uma sessão que se fecha não conduz uma execução longa.
- **Q4-A: filhos na mesma árvore, sem `bash`.** Teste vermelho primeiro e gate
  antes do merge pedem um filho que roda testes num lugar só dele; os filhos
  ganham worktree e branch próprias (RN-7). O `owns` disjunto continua, agora
  para evitar conflito de merge, e o diff é conferido contra ele.
- **Q3-A e a RN-7 antiga: um turno de integração por lote, com a definição de
  pronto na árvore inteira.** No lugar, cada filho passa por revisor, gate e
  merge numa fila serial, medido sobre a main de agora (RN-5, RN-6): a
  coerência entre filhos é conferida a cada merge, não uma vez no fim.
- **A grade como lugar da família** (filhos logo depois do pai, faixa de pontos,
  "↳ de <pai>") e a aba Filhos do Crew. O dono quer a grade como está e uma
  lista viva num painel lateral da orquestradora, com a forma da Mudanças
  (RN-9).
- **O orçamento só de tokens do lote.** Agora toda execução longa tem teto de
  custo **e** de tempo, e atingir um para o despacho (RN-3).
- **A observação de que o harness de eval não roda filho.** Ele roda desde
  `delegated-writing/changelog/202608200100`; a seção "O que existe hoje" foi
  corrigida.

## O que continua da #461

O filho como sessão do daemon com `parent_id`; a orquestradora que não espera
filho (era a RN-2); `owns` recusados inteiros quando se cruzam; sem neto; o
pedido de aprovação do filho que espera a pessoa, sem prazo, e que a
orquestradora nunca responde (Q5-B, ADR-02); todo fim de filho com nome; a
orquestradora com `explore` (Q2-A); o título curto dado no lote (Q8-A); o
relato que entra no começo do próximo turno, sem reescrever o prefixo (ADR-03);
protocolo aditivo, MINOR.

## O que é novo

Autonomia em três níveis trocáveis a qualquer momento (RN-3); a lista do que
sempre pergunta, imposta pelo harness (RN-4); revisor só leitura e gate rodado
pela orquestradora no commit do merge, com a alegação do filho registrada e
nunca aceita como prova (RN-5); merge num lugar próprio, fila serial, merge da
main em vez de rebase, main vermelha parando a fila (RN-6); modelos por tarefa
de uma lista permitida, com escalada (RN-8); notificações (RN-10); log de
decisões em JSONL e cenários roteirizados de graça (RN-11); deploy declarado
pelo projeto (RN-13); e a entrega em duas fatias, a primeira só com passo a
passo.

## O que o dono decidiu na revisão do rascunho

Quatro perguntas da §12 viraram requisito, cada uma com o porquê:

- **Q-C — o daemon é um LaunchAgent de usuário do `launchd`**: sobe no login,
  reinicia se cair, independe do app; app e TUI são só clientes. Emenda D19 —
  o app deixa de parar o daemon que subiu —, porque uma execução longa não pode
  morrer com a janela. A decisão do desktop é escrita quando for construída.
- **Q-G — cada filho vira um PR no forge, e a orquestradora faz squash-merge
  pelo `gh` com `--match-head-commit`**: o servidor recusa o merge se a branch
  andou depois da medição. Sem forge, merge local, dizendo o que se perde (o PR
  para ler, o CI do forge, a trava no servidor).
- **Q-I — em autônomo a aprovação do filho espera a pessoa, com notificação**:
  a orquestradora nunca a responde (ADR-02), segue com os outros filhos e pode
  replanejar aquela tarefa por um caminho sem aprovação.
- **Q-D — em supervisionado a decisão de desenho é perguntada**, com opções e
  recomendação, e só aquela tarefa espera.

## O que fica em aberto

Q-A (fragmentos) é PR próprio e o primeiro pré-requisito da primeira fatia;
Q-B (o app de celular) espera o app. Q-E (o que é sensível, quanto um fechado
fica na lista), Q-F (a memória de uma conversa que nunca termina) e Q-H
(conflito em arquivo só de acréscimo) têm recomendação no `.r`, a confirmar
pelo dono.

## Para depois

A entrada de `docs/ROADMAP.md` (perto da linha 345) ainda diz que o harness de
eval não responde chamada delegada; ele responde desde
`delegated-writing/changelog/202608200100`. Fica para um PR próprio, não este.
