# Convenção de versionamento

🇬🇧 [English version](VERSIONING.md)

## 1. SemVer, e o que ele não enxerga

Versões são `MAJOR.MINOR.PATCH`, com tag `vX.Y.Z`, como o [SemVer 2.0.0][semver]
define.

**Antes de 1.0, uma quebra sobe MINOR e não MAJOR.** `0.x` diz que a forma ainda
se mexe, e gastar o 1 na primeira quebra é como projetos chegam em 7.0 sem nada
estável dentro.

[semver]: https://semver.org/

## 2. A superfície pública, nomeada

Quase todo projeto adota SemVer e pula esta parte — e depois discute em revisão
se aquilo era quebra ou não. Aqui fica escrito.

**Coberto pela versão:**

| Superfície | Por que conta |
|---|---|
| comandos e flags da CLI | é o que uma pessoa digita |
| chaves de configuração, `DCODE_*` e TOML | é o que uma pessoa ajusta, e o que um administrador trava |
| `pkg/client` | a doc do próprio pacote o chama de API pública para consumidores do daemon |
| o protocolo cliente–servidor | cliente e daemon em versões diferentes precisam concordar |

**Não coberto:** tudo em `internal/`, o formato em disco dos registros de sessão,
e as fixtures de eval. Mudam sem versão dizer nada, e ninguém de fora constrói
sobre eles.

## 3. Comportamento faz parte do contrato

É aqui que um produto agêntico deixa o padrão para trás, e a diferença não é
acadêmica.

Mudar **uma frase** da descrição de uma ferramenta levou delegação de zero usos
para cinco numa medição. Nenhuma API mudou. Nenhuma assinatura se moveu. Um
usuário sentiria na hora.

Então:

- **contrato comportamental removido, ou cujo limiar desce, é no mínimo MINOR** —
  e o changelog nomeia o contrato;
- **descrição de ferramenta, lembrete ou doutrina que muda de sentido é no mínimo
  MINOR**, pelo mesmo motivo;
- limiar que sobe, ou contrato novo, é MINOR como funcionalidade.

A regra existe porque o SemVer lê assinaturas, e a superfície deste produto é em
parte feita de frases.

## 4. Derivar, em vez de decidir

Os commits daqui já seguem [Conventional Commits][cc] — `feat:`, `fix:`, `docs:`
— e seguiam antes de alguém combinar. A versão sai deles:

```bash
./scripts/version.sh      # a próxima versão, dos commits desde a última tag
./scripts/changelog.sh    # o esqueleto da seção dela
```

| commit | efeito |
|---|---|
| `feat:` | MINOR |
| `fix:`, `chore:`, `docs:`, `test:`, `refactor:`, `perf:`, `build:`, `ci:` | PATCH |
| qualquer `tipo!:`, ou `BREAKING CHANGE:` no corpo | MAJOR — ou MINOR enquanto abaixo de 1.0 |

[cc]: https://www.conventionalcommits.org/

**Os dois scripts recusam em vez de adivinhar.** Commit fora da convenção é erro,
não "patch por segurança"; tag que não é `vX.Y.Z` é erro, não uma tentativa de
aritmética. Versão escolhida por palpite silencioso é pior que script que para.

## 5. O changelog é gerado como esqueleto, nunca como prosa

O `scripts/changelog.sh` produz o que existe mecanicamente: quais PRs entraram,
agrupados, com número.

Ele **não** produz o **porquê**, e não deve. *"A recusa era honesta mas instruía
o abandono"* não está em assunto de commit nenhum e não sai de gerador nenhum.
Uma ferramenta que tentasse escreveria uma frase plausível sobre uma decisão que
ninguém tomou — pior que uma lista sem frase alguma.

Portanto: gerar o esqueleto, e escrever a razão de cada linha à mão.

## 6. Publicar

1. `make check` verde, e CI verde na `main`.
2. `./scripts/version.sh` para o número.
3. `./scripts/changelog.sh` para o esqueleto; escrever as razões; mover as
   entradas de `Não lançado` para a seção nova e reescrever o estado atual.
4. Mergear isso, e então marcar o commit do merge: `git tag -a vX.Y.Z -m 'vX.Y.Z'`.
5. `git push origin vX.Y.Z` — **isto publica.** O workflow compila todas as
   plataformas, gera checksums, assina com cosign e cria o release no GitHub.

O passo 5 é o que não volta. Tudo antes dele é reversível; tag publicada não é, e
é o único passo em que vale parar.
