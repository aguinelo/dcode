# Rolagem, indicador de processamento e navegação

**Data:** 2026-08-08
**Specs afetadas:** `202608081250-client-tui` (`.r`, `.p`, `.config`), `202608072240-client-server-protocol` (`.p`), `202608072334-provider-adapter` (`.p`)

## O que mudou

### Atalhos — `stable`, então a mudança é declarada

| Antes | Agora | Motivo |
|---|---|---|
| `p` alterna painel | **`^P`** alterna painel | `p` comia a primeira letra de toda mensagem começando com ela |
| `↑` `↓` movem no fluxo | `↑` `↓` **sensíveis ao contexto** | linha vazia percorre o histórico; linha com texto move no fluxo |
| — | `PgUp` `PgDn` `Home` `End` | rolagem, que a seção 7 já prometia e não existia |
| `Enter` sobre item alterna | `Tab` alterna, `Esc` fecha | `Enter` continua sendo enviar, sempre |
| — | `^A ^E ^W ^U ^K` `←` `→` `Del` | edição de linha |
| — | `?` em linha vazia abre ajuda | convenção de todo pager |

### Novas regras

**RN-12 — Rolagem para de seguir, e volta a seguir.** O fluxo acompanha a saída
mais nova por default. Rolar para cima desliga o acompanhamento; voltar ao fim
religa. Enquanto está desligado, saída nova **não move o que está na tela**.

**RN-13 — Uma tela pausada se anuncia.** Fora do fim, o cliente diz quantas
linhas há abaixo e como voltar.

**RN-14 — Trabalho em curso é visível e interrompível.** Enquanto um turno roda,
o cliente mostra indicador animado, o que está rodando, há quanto tempo, o custo
até aqui e a tecla que interrompe.

**RN-15 — Cor é informação, nunca decoração.** Cada cor tem papel semântico.
Desligada, nada de conteúdo se perde — o indicador de `full-access` muda o texto
além da cor. `NO_COLOR` e `TERM=dumb` desligam; `DCODE_COLOR` decide sobre os
dois.

**RN-16 — Letra nunca é atalho.** Numa linha em que se digita, atalho é tecla de
controle ou pontuação. Uma letra que às vezes é comando é uma letra que às vezes
some.

## Por que mudou

**A rolagem já estava na spec e nunca existiu.** `ScrollTop` estava no modelo,
não era lido por ninguém, e o renderizador mostrava só a cauda. Sessão longa era
irrecuperável: o que saiu da tela tinha saído para sempre.

**O `p` foi encontrado por teste.** Digitando "primeiro comando" a tela recebia
"rimeiro comando". Está na seção 7 como `stable` desde a primeira versão — e é
por isso que a troca vem com esta entrada, e não caladamente.

**O indicador de processamento é o que separa "trabalhando" de "travado".** Sem
ele a única ação disponível ao usuário diante de um turno longo é matar o
processo. A contagem é medida pelo cliente, não pelo servidor: precisa avançar
entre eventos, e carimbo de tempo do servidor só está certo no instante em que
chega.

## Impacto no protocolo

Aditivo, cliente antigo ignora campo desconhecido:

- `tool.completed` ganha `lines`, `files`, `added`, `removed`, `exit_code`,
  `has_exit`, `duration_ms`. Sem isso o resumo por ferramenta da seção 3.1 não é
  implementável sem reconstruir número por casamento de texto — que quebra no
  dia em que a redação muda, em todo cliente ao mesmo tempo.
- `turn.completed` ganha `usage`. Ponteiro, porque zero token e token
  desconhecido são fatos diferentes.
- `Session` ganha `context_window`. Sem denominador, `12400 tokens` não responde
  nada.

## Impacto no provider

**O dialeto openai só manda uso se for pedido.** `stream_options.include_usage`
passa a ir em toda requisição; sem ele o campo vem `null` em todo frame, em
qualquer provedor que fale esse dialeto.

**`Decoder` ganha `Close()`.** O MiniMax repete `finish_reason` e anexa o uso ao
**último** frame, e nunca manda `[DONE]`. Encerrar no primeiro finish jogava a
contabilidade fora. O decoder segura o evento terminal até o transporte dizer que
não vem mais nada.

## Alternativa descartada

Manter `p` e desambiguar por contexto. Não há contexto: no momento em que se
digita `p` como primeira letra, a linha está vazia — que é exatamente a condição
do atalho. Qualquer regra aqui erra em um dos dois sentidos, e errar comendo
caractere é o pior dos dois.
