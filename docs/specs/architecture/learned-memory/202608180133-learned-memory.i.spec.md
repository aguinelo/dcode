# Implementation: Memória aprendida

## Ordem de execução

A ordem existe para que cada passo seja verificável sozinho, e para que a
garantia mais importante — nada aprendido vence nada humano — esteja no lugar
antes de haver o que aprender.

### 0. Promover a lista de invariantes

- [ ] Renomear `## 10. Invariantes a garantir` para
      `## 10. Invariantes verificáveis` no `.p`.
- [ ] Criar o guarda de invariantes do pacote de memória, nomeando
      `learned-memory`, no mesmo formato dos outros dez.

> Os caminhos de arquivo aqui estão escritos sem crase de propósito. O
> `specguard` trata caminho em crase num `.i` como **afirmação de que o arquivo
> existe**, e num spec de desenho nenhum existe ainda. Passam a levar crase
> quando forem escritos, e a partir daí o guarda cobra que continuem lá.

**Primeiro, e não por formalidade.** Enquanto a seção tiver o outro título, o
`specguard` não cobra nada dela — e uma lista de invariantes que ninguém cobra é
exatamente a forma de defeito que esta spec inteira existe para não repetir. Com
o guarda no lugar desde o começo, cada invariante entregue nos passos abaixo é
cobrada no mesmo commit que a implementa.

### 1. A fonte, e sua posição na tabela

- [ ] `behavior.SourceLearned`, com autoridade `0`.
- [ ] Teste que falha se `learned` ordenar acima de qualquer fonte humana, em
      toda combinação.
- [ ] Teste que falha se alguma chave de configuração alcançar a tabela.

**Antes de tudo o mais, e sozinho.** Se esta parte estiver errada, todo o resto é
um caminho para o agente reescrever as próprias restrições devagar. É a mesma
razão pela qual `Safety` foi construída antes de a sobreposição existir.

### 2. Leitura e procedência

- [ ] `internal/memory`: ler `.dcode/memory.md`, devolver as memórias tipadas.
- [ ] Arquivo ausente devolve nada, sem erro. Diretório sem `.dcode` idem.
- [ ] Arquivo malformado devolve o que deu para ler **e diz o que não deu** —
      falhar a sessão por causa de um bloco torto seria a memória tomando o
      produto de refém, o mesmo raciocínio que a gravação já segue.
- [ ] O prefixo nomeia a procedência como aprendida.
- [ ] Teste: workspace sem memória produz prefixo **byte-idêntico** ao anterior.

Neste ponto o componente é útil sozinho: uma pessoa pode escrever `.dcode/memory.md`
à mão e o agente passa a ler. **Vale parar aqui e usar assim por um tempo** — é a
forma mais barata de descobrir se o formato serve antes de haver ferramenta que o
escreva.

### 3. Limite e obsolescência

- [ ] Teto de `memory.max_entries`, mais recentes primeiro, corte declarado.
- [ ] Memória cujo commit não existe mais é marcada e **permanece**.
- [ ] Teste do corte, e teste de que nada é removido do arquivo.

A checagem de commit usa o instantâneo que `internal/vcs` já tira, não uma
leitura nova: duas leituras do git na mesma sessão podem discordar, e discordar
sobre onde a sessão está é pior que não saber.

### 4. A ferramenta

- [ ] `tools.Remember`: tipo, assunto, corpo.
- [ ] Declara escrita no caminho da memória, e em nenhum outro.
- [ ] Recusa tipo fora da lista, nomeando os três.
- [ ] Recusa assunto vazio.
- [ ] Acrescenta; nunca reescreve o que estava lá.
- [ ] Teste de que a memória escrita **não** altera o prefixo da sessão corrente.

### 5. O lembrete de Camada 2

- [ ] Detectar mesmo código de erro de ferramenta, mesmo caminho, duas vezes no
      turno.
- [ ] Emitir uma vez por turno; rearmar quando a situação deixar de existir.
- [ ] Teste de que não carrega contagem no texto — número que varia entre
      execuções idênticas quebra a reprodutibilidade (RN-7 de `context-engine`).

**Por último, e só se necessário.** É o contrapeso para o modelo não chamar
`remember`, e não se constrói contrapeso antes de medir o peso.

### 6. Os contratos

- [ ] `remembers-what-cost-time`, `does-not-remember-activity`,
      `uses-what-it-remembers`.
- [ ] Rodar, ler o número, **então** declarar limiar.

## Ordem de dependência

```
behavior.SourceLearned
        ↓
internal/memory (ler)  ←──  internal/vcs (commit, já existe)
        ↓
behavior.Build (prefixo)
        ↓
tools.Remember (escrever)
        ↓
lembrete de Camada 2
```

A escrita depende da leitura, e não o contrário. Construir a ferramenta primeiro
daria um produto que grava algo que nada lê — a forma de defeito que este
repositório mais encontrou.

## Fora deste componente

**A gravação de sessão.** Ela responde "o que aconteceu"; esta responde "o que
este repositório ensina". Nenhuma das duas lê a outra.

**O `/init`.** Ele destila o que o repositório **já diz**; a memória guarda o que
ninguém escreveu ainda. Um dia pode haver ponte, e será decisão própria.

**A poda.** `session.Prune` limpa gravação por idade e tamanho. Memória não é
podada por nada — RN-8 e RN-10 juntas dizem que a única remoção legítima é
alguém apagando no PR.

## Armadilhas conhecidas

**Memória escrita alterando o prefixo da sessão que a escreveu.** Tentador, e
quebra a pureza que `context-engine` garante. O teste que impede isso não é
opcional.

**Marcar obsolescência como remoção.** A heurística vai errar, e o erro apaga
conhecimento em silêncio. RN-8 existe por isso.

**Deixar `remember` reescrever o arquivo** para ordenar, deduplicar ou limpar.
Cada uma dessas transforma um diff de três linhas num diff do arquivo inteiro, e
um diff ilegível é uma revisão que não acontece — que é o único portão de
qualidade que este desenho tem.

**Construir o consolidador porque o modelo não chamou a ferramenta.** Se a
medição mostrar isso, a resposta é o lembrete de Camada 2. Um segundo modelo
rodando depois que a pessoa parou de olhar é a forma que RN-6 recusa.

## Changelog

- [202608180133 — Memória aprendida](changelog/202608180133-memoria-aprendida.md)
