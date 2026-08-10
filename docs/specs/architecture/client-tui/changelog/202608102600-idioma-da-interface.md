# Idioma da interface

**Data:** 2026-08-10
**Specs afetadas:** `202608081250-client-tui` (`.r`, `.p`, `.config`, `.i`), `202608081203-configuration` (`.config`), `docs/conventions/LANGUAGE.md`

> **Regra:** o texto que a **pessoa** lê tem idioma; o texto que o **modelo** lê continua em inglês.

Resolve as cinco perguntas de `docs/backlog/202608100900-idioma-da-interface.md`.

## Dois eixos, e só um estava aberto

**Conversa — já resolvida, sem código.** `Doctrine.Style` carrega *"Answer in the language the user wrote in."*, e é por isso que o modelo respondeu em português sem ninguém configurar nada. Adaptativo é melhor que default fixo: um repositório com issues em inglês e um usuário que escreve em português continuam sendo atendidos cada um na sua língua.

Com `202608101800` isso passa a ser editável por quem quiser forçar — `style.md` na raiz do usuário. Nada a fazer aqui.

**Interface — não existe.** Toda cadeia é literal em inglês no código.

## A divisão que decide o trabalho

Nem toda cadeia é interface, e confundir as duas é o erro caro:

| Categoria | Quem lê | Idioma |
|---|---|---|
| Texto do cliente — TUI, `/help`, saída de CLI | a pessoa | **do usuário** |
| Erro de configuração | a pessoa | **do usuário** |
| Erro de ferramenta | **o modelo**, para se corrigir | **inglês, sempre** |
| Descrição de ferramenta e doutrina | o modelo | **inglês, sempre** |

Erro de ferramenta é o caso difícil, porque o modelo lê para se recuperar **e** a pessoa lê na tela. Quem manda é o modelo, por dois motivos:

- A RN-3 de `behavior-definition` diz que mensagem de erro de ferramenta **é superfície de comportamento** — a camada mais eficiente da pilha, onde a recuperação é ensinada. Traduzi-la é mudar o prompt.
- Os limiares comportamentais — `tool-over-shell`, `reminder-acted-upon` e companhia — foram **medidos em inglês**. Traduzir invalida a medição sem que nada quebre visivelmente.

A ADR-05 já estabelece que a formulação pertence à família; idioma é decisão da mesma natureza. **O cliente traduz apenas o que ele mesmo compõe em volta.**

Isso vai escrito na spec com o motivo, para que a próxima pessoa não "conserte".

## As cinco decisões

### 1. Default

`DCODE_LANG` quando definida. Vazia, lê `LC_ALL` e depois `LANG`. Idioma não suportado ou ausente cai em **pt-BR**.

O pedido original foi "default pt-BR". Interpretei como **identidade do produto**, não como imposição: quem tem a máquina em inglês recebe inglês, e o que muda é qual idioma é o fallback quando não há informação. Um produto cujo fallback é pt-BR é um produto brasileiro; um cujo fallback é inglês é um produto em inglês com tradução pendurada.

É o único ponto onde interpretei em vez de seguir ao pé da letra, e está registrado aqui por isso.

### 2. Onde as cadeias moram

**Mapa em Go, embutido.** Sem dependência, sem arquivo externo que possa faltar, e coerente com um binário estático sem cgo.

Arquivo por idioma facilitaria contribuição de tradução e criaria carregamento em runtime — o que também cria um modo de falha novo, num produto cuja interface precisa funcionar antes de qualquer configuração ter sido lida.

### 3. Chave ausente

**A pergunta some.** Um teste exige que todo idioma declarado cubra **todas** as chaves, e falha se faltar uma. Chave ausente deixa de ser condição de runtime e vira erro de build.

Cair para inglês esconderia tradução faltando para sempre; mostrar a chave crua é feio e chega tarde. Nenhum dos dois é necessário quando a ausência não compila.

### 4. Colisão com `LANGUAGE.md`

Não há, mas parece haver — e é por isso que a distinção vai escrita **na própria convenção**:

> `LANGUAGE.md` governa **quem contribui**: artefato canônico em inglês, comentário e commit em inglês, spec em português. Idioma de interface governa **quem usa**. Públicos diferentes, decisões independentes.

### 5. Largura

O renderizador já mede em células de exibição, então acento e CJK não quebram layout. O risco é outro: **texto traduzido é mais longo** — alemão passa de 30% —, e há linha com coluna fixa, como a de ferramenta, com 6 células para o nome e 26 para o alvo.

Teste de largura **por idioma declarado**, sobre as linhas de coluna fixa. É o que evita descobrir o estouro em captura de tela.

## Escopo do primeiro corte

`internal/tui` e `cmd/dcode` — a interface de verdade. `internal/tools`, `internal/policy` e a doutrina ficam em inglês por decisão, não por atraso.

## Fronteira de determinismo

| Parte | Regime | Verificação |
|---|---|---|
| resolução do idioma a partir de config e ambiente | determinístico | asserção |
| cobertura total de chaves por idioma | determinístico | asserção de build |
| largura das linhas de coluna fixa por idioma | determinístico | asserção |
| ausência de tradução no texto voltado ao modelo | determinístico | varredura |

Regime **inteiramente determinístico**: nada aqui depende do modelo. A conversa, que dependeria, já está resolvida pela doutrina e fora deste escopo.

## Invariantes

- `DCODE_LANG` vence `LC_ALL`, que vence `LANG`; nenhum vale, resolve `pt-BR`.
- Idioma declarado e desconhecido resolve `pt-BR`, sem erro.
- Todo idioma declarado cobre 100% das chaves — falha de teste se faltar.
- Nenhuma chave de tradução é referenciada por `internal/tools`, `internal/policy` ou `internal/behavior` — varredura de importação.
- Descrição de ferramenta e texto de erro de ferramenta são idênticos em qualquer idioma — asserção sobre a saída de `Build` e sobre `Result`.
- Linha de coluna fixa cabe na largura declarada em **todo** idioma.

## Impacto

- Novo pacote de cadeias em `internal/tui`; nenhuma mudança de comportamento.
- Uma chave nova, `DCODE_LANG`.
- Nota na `LANGUAGE.md` e na `LANGUAGE.pt-BR.md` separando contribuinte de usuário.
- `docs/backlog/202608100900-idioma-da-interface.md` deixa de ser pauta: as cinco perguntas estão decididas aqui.
