# O formato é o mesmo

**2026-08-31** — o bloco de skills passa a dizer qual é o formato, além de onde
elas moram.

## A lacuna que sobrou

A correção anterior fez o agente saber onde as skills moram, e funcionou pela
metade: ele foi olhar o diretório e citou o "None are installed" do próprio
prompt. Depois disso, pedido para instalar uma skill do GitHub, concluiu:

> mesmo eu rodando aqui, sou o agente com tools limitadas, NÃO o Claude Code.
> Skills são coisas que carregam no Claude Code, não no meu agente.

Ele acertou a parte difícil — identificou que a URL apontava para um skill
dentro de um **plugin**, que plugin e marketplace são empacotamento do Claude
Code, e que instalar plugin mexe no setup global e por isso pede confirmação.
Tudo correto.

E errou a parte fácil: nunca considerou `curl` do `SKILL.md` e escrita em
`.dcode/skills/`. Duas linhas, dentro do workspace, sem cruzar fronteira nenhuma.

## Por que ele não considerou

O bloco dizia onde as skills moram e que escrever uma é escrita comum. Não dizia
**qual é o formato**. Sem isso, "esse `SKILL.md` que eu achei no GitHub" e "um
arquivo que eu posso pôr em `.dcode/skills/`" continuam sendo duas coisas
diferentes na cabeça de quem lê.

## O que é fato, e por isso pode ser dito

Não é promessa: é medido. A `web-design-engineer` do `ConardLi/garden-skills` —
formato Anthropic puro, `SKILL.md` numa pasta com `name` e `description` — foi
baixada, posta em `.dcode/skills/` e **carregou e foi aplicada** num teste de
campo nesta mesma tarde. O `description` já era aceito como `when_to_use` desde
antes desta família existir.

## O que fica de fora, de propósito

**Plugin, marketplace, comando de instalação.** Isso é empacotamento e
distribuição de outro produto, não formato, e o agente já raciocina bem sobre
isso sozinho — a transcrição acima é a prova.

**A divergência do casamento.** Lá o modelo decide pela descrição; aqui é
determinístico por palavra, que é a razão do teto de 120 caracteres e do campo
`triggers`. Isso é escolha de desenho, não lacuna, e não cabe no prefixo: quem
precisa saber é quem escreve skill, e está na `.r`.

## O tamanho

409 bytes, com teto de 520 no teste. Continua sendo pointer, não manual, e o teto
existe para que continue.

## Invariantes

- `TestTheAgentIsToldWhereSkillsLiveEvenWithNoneInstalled` — agora exige
  `SKILL.md` e `description` na seção, além do caminho.
