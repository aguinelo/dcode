# Valor que não converte é dito, não trocado pelo padrão

**Data:** 2026-10-18
**Specs afetadas:** `202608081203-configuration` (`.p`: §5 ganha `Err` e diz o
vocabulário de `Bool` e de `Int`; §7 ganha uma linha)

## O defeito

`Resolved.Bool` e `Resolved.Int` devolviam o padrão em silêncio quando o valor
não convertia, e `Int` lia com `fmt.Sscanf("%d")`, que para no primeiro
caractere que não é dígito. Na `main`:

```
allow_network = "talvez"      → sessão aberta com a rede ligada (o padrão)
DCODE_MAX_ITERATIONS=12abc    → 12
```

Nos dois casos a configuração respondia sem erro e sem aviso, e a linha do
arquivo valia como se não existisse — ou, pior, como um prefixo dela.

## O que mudou

- `Int` lê o valor inteiro, com `strconv.Atoi`: `12abc` não é doze.
- `Bool` mantém o vocabulário que sempre aceitou (`1/true/yes/on`,
  `0/false/no/off`, sem distinguir maiúsculas). Não passa a ser
  `strconv.ParseBool`, porque `yes`/`on` são aceitos hoje e tirá-los quebraria
  arquivos que já funcionam; o defeito não era o vocabulário, era o silêncio.
- O valor que não converte continua devolvendo o padrão ao chamador — quem lê
  monta uma configuração inteira numa expressão —, mas fica registrado no
  `Resolved`. `Err()` devolve todos, cada um com **chave, valor e camada**
  (arquivo, ou o nome da variável de ambiente).
- `FromEnv` pergunta `Err()` depois de ler e **recusa**: a CLI sai com a
  mensagem, e o daemon recusa a sessão com `workspace_invalid`, como já recusava
  uma configuração de projeto que não se lê. `dcode serve` pergunta de novo
  depois de ler as chaves de `record.*`, que são lidas fora de `FromEnv`.

## Por quê

Aviso em vez de erro foi considerado e descartado. Aviso é o que se faz com uma
chave travada, onde o valor efetivo é o do administrador e está certo. Aqui o
valor efetivo é um padrão que ninguém escolheu, e em `allow_network` o padrão é
abrir a rede: a sessão rodaria sob uma regra que o arquivo tentou mudar. É a
mesma razão pela qual chave desconhecida é erro e não aviso.

## O que NÃO mudou

Chave ausente e valor vazio continuam devolvendo o padrão sem erro. `String` não
converte nada e não muda. `evals.FromReader` lê `eval.runs`/`eval.enabled` por
uma interface própria e fica fora desta mudança.
