# Changelog — desktop

🇺🇸 [English version](CHANGELOG.md)

O registro próprio do cliente desktop, versionado à parte do núcleo: tags
`desktop-vX.Y.Z`, a partir de 0.1.0. Nada foi publicado ainda. O que mudou e por
quê, uma entrada cada; o detalhe de uma decisão está em `docs/DECISIONS.md`.

---

## Não publicado

- **A área do desktop existe, à parte do núcleo.** Electron Forge com Vite,
  TypeScript e React, em `desktop/`, com README, changelog, versão e `AGENTS.md`
  próprios, e um `go.mod` de fachada para o `go … ./...` do núcleo parar na porta
  em vez de entrar no `node_modules`. À parte porque o desktop depende do núcleo e
  nunca o contrário, e um cliente que sai no próprio ritmo não deve pegar
  emprestada a versão do núcleo.
- **A janela principal do design v2, sobre eventos de protocolo gravados.** A
  lateral de projetos e sessões com as marcas de estado, uma sessão por vez com o
  seu fluxo — a pergunta, linhas de ferramenta compactas, o texto do modelo, o
  card de delegação, a linha de atividade —, o card de aprovação com as três
  respostas e o relógio ao vivo, o composer e a barra inferior, nos dois temas,
  seguindo o sistema. Gravados, e não ao vivo, porque a janela tinha de estar
  certa antes de ser ligada: a próxima versão conecta o processo principal ao
  socket do daemon. A barra inferior diz `gravação` para uma gravação nunca
  passar por daemon, e toda ação que precisaria de um avisa que nada foi enviado.
- **Um redutor puro de eventos do protocolo para o que a janela mostra**, testado
  com vitest. Lê só campos que o protocolo traz, validados na fronteira: payload
  que não bate é dito no fluxo da sessão, evento repetido não muda nada, evento
  que falta é nomeado. Onde o design pede o que o fio não traz — o progresso de
  um filho, tokens ao vivo, os caminhos pedidos na aprovação, o loop —, desenha o
  que existe e o `docs/DECISIONS.md` lista a lacuna, porque campo inventado é
  promessa que o daemon nunca fez.
- **Tipos do protocolo gerados do `internal/protocol`** com o tygo, commitados, e
  conferidos no CI; todo tipo de evento que o núcleo declara precisa de leitor
  aqui, e toda forma de payload precisa concordar com o tipo gerado, ou os testes
  e o typecheck reprovam. Uma definição só do fio, e um desktop que percebe quando
  ele se move.
- **Ordem, rótulos e recolhimento dos projetos são preferências locais**, guardadas
  no armazenamento do próprio app e nunca mandadas ao daemon: recolher e expandir,
  recolher tudo, renomear no lugar, mover para cima e para baixo, e arrastar para
  reordenar. Armazenamento que não se lê ou não se grava é dito, e a lateral volta
  ao arranjo inicial.
- **O renderer não alcança nada.** Isolamento de contexto, sandbox, nenhum Node,
  um preload que expõe a plataforma e o nome de quem usa e mais nada, uma CSP que
  não o deixa conectar a lugar nenhum, fontes empacotadas, nenhuma porta aberta —
  e o lint reprova um import de Node em código do renderer.
- **O design pode ser conferido num loop.** `npm run check:visual` compara a
  janela em modo fixture com as capturas de referência do design (Chromium do
  Playwright, pixelmatch) e reprova além de um limite cujo raciocínio fica ao
  lado dele: o próprio mock do design, desenhado pelo mesmo Chromium, fica a
  1,15% e 1,19% das referências, e a janela mede 1,17% e 1,19%. O
  `desktop/docs/loop/` deixa o `/loop` do próprio dcode iterar contra ele, com as
  referências e a régua protegidas. Só local por enquanto: as referências ainda
  não estão no repositório.
