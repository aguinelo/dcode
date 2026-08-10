# Política de idioma

🇬🇧 [English version](LANGUAGE.md)

Este projeto é bilíngue: inglês e português do Brasil. Este documento é a regra.

**Isto governa quem contribui, não quem usa.** Artefato de repositório — código,
comentário, commit, spec — segue as regras abaixo. O idioma que o *usuário* vê no
terminal é decisão separada, tomada na RN-19 de `202608081250-client-tui`: a
interface segue o idioma dele, enquanto o texto que o modelo lê — descrição de
ferramenta, erro de ferramenta, doutrina — permanece em inglês por ser superfície
de comportamento, e porque os limiares comportamentais foram medidos em inglês.

Dois públicos distintos. Nenhuma das regras restringe a outra.

---

## 1. Nomenclatura de arquivo

Inglês é o padrão e fica com o nome sem sufixo. Português é a tradução e leva o sufixo
`.pt-BR`.

```
README.md                       inglês (canônico)
README.pt-BR.md                 português

docs/conventions/TESTING.md         inglês (canônico)
docs/conventions/TESTING.pt-BR.md   português
```

Todo par traduzido tem link cruzado para o correspondente na primeira linha após o
título, para que quem chegar em qualquer um dos dois consiga trocar.

## 2. O que existe nos dois idiomas

| Artefato | Idiomas | Canônico |
|---|---|---|
| `README` | ambos | inglês |
| `docs/conventions/**` | ambos | inglês |
| `docs/brand/**` | ambos | inglês |
| Templates de issue e PR | ambos | inglês |
| Comentários de código | só inglês | — |
| Mensagens de commit e títulos de PR | só inglês | — |
| `docs/specs/**` (RPI) | **só português** | português |

## 3. Por que as specs ficam em um idioma só

Esta é a exceção deliberada, e não é esquecimento.

O protocolo RPI define o `.r.spec.md` como **verdade absoluta** — se o código contradiz,
o código está errado. Essa regra só funciona com exatamente uma fonte da verdade. Duas
cópias de uma spec vão divergir, e no momento em que discordarem não há como saber qual
delas o código deveria satisfazer. Spec divergente é pior que spec ausente, porque
parece autoritativa.

Spec também é documento interno de trabalho, não a face pública do projeto. O público de
um `.p.spec.md` é quem vai implementá-lo, e esse público já lê português.

As regras canônicas do RPI exigem português, e este projeto não altera o RPI.

**Se algum dia um contribuidor externo precisar de uma spec em inglês, traduza no pull
request que precisa dela e marque a tradução explicitamente como não-canônica** — nunca
como fonte da verdade paralela.

## 4. Por que os commits ficam em um idioma só

Mensagem de commit alimenta geração de changelog e é lida por ferramenta que assume um
idioma único. Corpo de commit bilíngue dobra o ruído no `git log` sem ajudar ninguém:
quem lê o código já lê inglês, porque o código e seus comentários são em inglês.

Os prefixos de tipo do Conventional Commits (`feat`, `fix`, `docs`) são identificadores
em inglês de qualquer forma.

## 5. Manter as traduções em sincronia

Tradução atrasada é tradução que mente.

- Pull request que altera um documento **deve** atualizar as duas versões no mesmo pull
  request. Alterar só uma é bloqueado na revisão.
- Quando divergirem, a versão canônica vence e a tradução é o bug.
- Traduza sentido, não palavra. Termo técnico consagrado permanece em inglês dentro do
  texto em português — *append-only*, *sandbox*, *cache*, *golden file* — porque
  traduzir dificulta a leitura justamente de quem usa esses termos.
