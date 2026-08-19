# Convenção de testes

🇬🇧 [English version](TESTING.md)

Vale para todo código deste repositório. Um PR que viole qualquer regra aqui é bloqueado.

---

## 1. TDD

Ciclo obrigatório para código novo: **vermelho → verde → refatorar.**

1. Escreva o teste primeiro. Ele **deve** falhar.
2. Verifique que ele falha **pelo motivo certo** — asserção quebrada, não erro de compilação nem typo no nome do arquivo. Teste que falha pelo motivo errado não é vermelho, é ruído.
3. Escreva o mínimo de código para passar.
4. Refatore com o teste verde como rede.

Estilo **London School (mock-first)** para código novo, conforme convenção da organização: as dependências da unidade sob teste são substituídas por dublês, e o que se verifica é a interação com os colaboradores. Isso mantém o teste rápido e a fronteira do módulo explícita.

Exceção deliberada: código de fronteira com o sistema operacional — sandbox, socket, PTY — é testado com o recurso real em `t.TempDir()`, não com mock. Mockar `syscall` testa o mock, não o comportamento.

---

## 2. Bug exige teste de reprodução — antes da correção

**Regra explícita, sem exceção:**

1. Reproduza o bug em um teste. O teste **falha**.
2. Confirme que ele falha exatamente pelo sintoma relatado.
3. Só então corrija.
4. O mesmo teste passa, sem ser alterado.

O teste de reprodução entra **no mesmo commit ou PR** da correção. Um PR de `fix:` sem teste novo é bloqueado.

### Por que a ordem importa

Um teste escrito depois da correção não prova que reproduzia o bug — só que o código atual passa nele. Se ele nunca foi visto vermelho, não há evidência de que ele pegaria a regressão. Escrever antes é o que transforma o teste em rede de segurança em vez de decoração.

### Regressão é permanente

Teste de reprodução nunca é removido, nem "simplificado" em refatoração. Nomeie de forma rastreável — `TestEventLog_NoGapUnderConcurrentAppend_Issue42` — para que fique óbvio, anos depois, que aquele caso existe porque quebrou de verdade uma vez.

---

## 3. Cobertura: piso de 90%, e cenário crítico testado

A CI falha abaixo de **90%** de cobertura de linha sobre o denominador inteiro,
**e** abaixo de **90%** em qualquer pacote por conta própria.

Os dois números são iguais de propósito. O agregado responde "existe pacote sem
teste nenhum?"; o piso por pacote responde "existe pacote fraco escondido atrás
de um forte?". Enquanto o agregado esteve em 95 ele fazia o segundo trabalho por
tabela, e o piso podia ficar sendo impresso e ignorado. Com o agregado em 90 o
piso é o único que morde, então ele passa a reprovar em vez de apenas relatar.

### Por que 90 e não 95

O agregado esteve em 95, no valor exato que a árvore media. Um gate colado no
medido reprova por arredondamento e por geografia de plataforma, não por código
sem teste — e foi o que aconteceu: três PRs numa noite, nenhum por falta de
teste. 90 é o número que os seis `.i.spec.md` já pedem no próprio pacote, e é o
número que a árvore cumpre com folga.

O agregado sobe quando já está folgado, nunca para um número que a árvore ainda
não cumpre — gate vermelho na chegada é gate que as pessoas aprendem a ignorar.
Foi essa frase que a subida para 95 violou, na mesma página que a escreve.

### Cenário crítico é testado, independentemente de percentual

Percentual mede quanto do código rodou, nunca o que foi asserido. Estes têm
teste porque são o que o produto promete, e nenhum deles é dispensado por um
número bom:

- **toda travessia de fronteira de segurança** — decisão de política, contenção
  de sandbox, leitura e escrita de credencial;
- **todo caminho onde dado do usuário pode sumir** — gravação de sessão, log de
  eventos, compactação de contexto;
- **todo bug já visto uma vez**, com o teste de reprodução da seção 2;
- **todo invariante declarado** em `## N. Invariantes verificáveis` de um
  `.i.spec.md`.

O último é o único que uma máquina cobra, e é por isso que ele existe: o
`specguard` reprova invariante que não nomeia um teste existente, e reprova
família que declara invariante sem guarda. Cenário crítico que não vira
invariante fica valendo o que vale uma frase — e este repositório já sabe quanto
é. Ao escrever um, escreva o invariante.

```bash
go test -race -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

### O denominador

Um gate sem denominador definido é ou inalcançável ou vazio. Aqui ele é explícito.

**Entra na conta:** todo código determinístico em `internal/**` e `pkg/**`.

**Fica fora, com justificativa:**

| Exclusão | Motivo |
|---|---|
| Código gerado | não é autorado; testar gerador, não saída |
| `cmd/**` — wiring de `main` | montagem de dependência, sem lógica; coberto por teste de fumaça |
| Caminhos mediados por modelo, atrás de build tag | não é verificável por asserção — ver seção 4 |
| Código específico de SO, fora da plataforma do runner | não é executável ali; coberto na matriz de CI da plataforma correspondente |

### O gate mede a matriz, não um runner

A linha acima ficou meses declarada sem nada que a cumprisse: o gate rodava
dentro de cada job da matriz, então um ramo alcançável só no macOS contava como
descoberto no Ubuntu. Reprovou três PRs numa noite, sempre pelo mesmo motivo, e
sempre em código que estava testado — na outra plataforma.

Na CI o gate roda uma vez, sobre a união dos perfis:

```bash
./scripts/merge-coverage.sh perfis/*/coverage.out > coverage.out
./scripts/coverage.sh coverage.out
```

`make check` continua medindo só a plataforma de quem roda. É uma aproximação
mais severa que a da CI, não mais frouxa — o que ela reprova a CI também
reprovaria por outro caminho, e o contrário é o que este arranjo conserta.

Exclusão nova exige justificativa no PR. "Difícil de testar" não é justificativa — costuma ser sintoma de acoplamento, e a correção é o desenho, não a exceção.

### O que o gate não prova

O gate é **piso, não meta**. Ele pega arquivo sem teste nenhum; não prova correção.

A forma clássica de burlar é teste sem asserção — exercita a linha, não verifica nada, sobe o número. Na revisão, teste que chama função e não afirma nada sobre o resultado é achado, mesmo com a cobertura verde.

Cobertura de linha também não é cobertura de caso: 100% de linha com um único caminho feliz ignora todo erro. Table-driven com casos de borda vale mais que percentual.

---

## 4. Comportamento mediado por modelo

Não entra no gate, e a razão está na fronteira de determinismo declarada em cada `.r.spec.md` (ver `SDD-HARNESS.pt-BR.md`).

Comportamento que emerge da interação com o LLM não é verificável por asserção. É medido por **limiar sobre fixtures**, declarado na seção de contratos comportamentais do `.p.spec.md` correspondente.

- Fica atrás de build tag ou `testing.Short()` — depende de modelo real e custa dinheiro.
- Regressão abaixo do limiar é blocker, igual a teste vermelho.
- Rebaixar limiar no mesmo PR que o quebra exige entrada em `changelog/`, porque é mudança de regra.

**O incentivo correto:** como este código fica fora do gate, existe pressão para empurrar comportamento para o lado determinístico — onde ele conta para a cobertura e é verificável com exatidão. Isso é intencional, e é o mesmo objetivo de arquitetura descrito em `SDD-HARNESS.pt-BR.md`.

---

## 5. Checklist de PR

- [ ] Código novo veio de teste que falhou primeiro.
- [ ] `fix:` acompanha teste de reprodução que falhava antes da correção.
- [ ] `go test -race ./...` limpo.
- [ ] Cobertura ≥ 90% no denominador definido, e nenhum pacote abaixo de 90%.
- [ ] Cenário crítico tocado pelo PR está asserido, e escrito como invariante.
- [ ] Nenhum teste novo sem asserção.
- [ ] Exclusão de cobertura nova, se houver, justificada na descrição do PR.
- [ ] Spec sincronizada, se o comportamento técnico mudou.
