# O arquivo que parava o produto

**2026-08-31** — RN-7 ganha a regra do arquivo ruim; `LoadSkills` e `ParseSkill`
passam a devolver aviso em vez de erro fatal.

## Como foi encontrado

Teste de campo, não leitura de código. Baixei uma skill real de
`ConardLi/garden-skills/skills/web-design-engineer`, instalei em
`.dcode/skills/`, e o `dcode` recusou-se a rodar:

```
$ dcode --dump-prompt
dcode: behavior: .../web-design-engineer.md has a `when_to_use` of 455
characters, over the 120 limit.
$ echo $?
1
```

Não era só o `--dump-prompt`: qualquer coisa que montasse o prompt saía com 1.
Um arquivo, e o binário não rodava mais naquele workspace.

## Por que isso é grave, e não cosmético

`description` de 455 caracteres é **normal** no formato de onde estas skills
vêm. Não era um arquivo corrompido; era um arquivo comum do ecossistema.

E `.dcode/skills/` chega por `git clone`. Ou seja: qualquer repositório clonado
podia decidir se o `dcode` rodava — negação de serviço por arquivo de texto, na
mesma superfície que a RN-11 já trata com desconfiança ("veio junto com código
que pode ter sido clonado de qualquer lugar, e **não é o usuário**").

## A regra

Os tetos continuam. O que muda é o que acontece quando são estourados.

| Situação | Antes | Agora |
|---|---|---|
| `when_to_use` acima de 120 | erro fatal | **aparada** em fronteira de palavra, corte reportado |
| sem `when_to_use`, sem corpo, frontmatter aberto | erro fatal | **pulado**, com aviso |
| arquivo acima do teto de bytes | erro fatal | **pulado**, com aviso |
| diretório ilegível | erro fatal | erro fatal |

O último continua fatal porque é a máquina falhando, e não um arquivo estando
errado — a distinção é a mesma que separa "não olhei" de "olhei e não há".

## Aparar ou pular

A linha é aparada; o corpo, nunca. Corpo cortado no teto é orientação que para
no meio da frase, e orientação ausente e **declarada ausente** é melhor que
isso. A linha, não: ela é economia de índice, e o teto se justifica sozinho —
o que o teto não justifica é jogar fora um corpo que funciona porque a frase que
o descreve foi escrita para um produto sem teto.

A elipse entra no orçamento do corte. Aparar até o teto e **depois** anexar "…"
produz linha acima do teto, que é a mesma classe de erro que a função existe
para impedir.

## Onde o aviso aparece

`--dump-prompt`, em bloco próprio, separado dos avisos de doutrina porque
respondem perguntas diferentes e quem procura um não devia ter que ler o outro.

## Invariantes

- `TestASkillWhoseLineIsTooLongIsTrimmedAndReported`
- `TestASkillFileThatCannotBeReadIsSkippedAndReported`
- `TestAnOversizeSkillIsSkippedAndReported`
