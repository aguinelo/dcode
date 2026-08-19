# Config: Sandbox e Política de Permissão

> Nenhuma variável de ambiente nova no código sem estar aqui.
> Precedência: **config travada por administrador** > flag > variável de ambiente > arquivo de config > default.

> `DCODE_SANDBOX_MODE` e `DCODE_APPROVAL_POLICY` são declaradas em `202608072240-client-server-protocol.config.spec.md`, seção 3, porque o protocolo as expõe em `CreateSessionRequest`. Esta spec **não** as redeclara — ela define o comportamento que elas controlam.
>
> `DCODE_APPROVAL_TIMEOUT` **não existe** como variável: o teto de aprovação é a flag `-approval-timeout` do daemon, e a chave foi removida em `202608110900` por nunca ter sido lida.

## 1. Mecanismo de sandbox

| Variável | Tipo | Default | Uso |
|---|---|---|---|
| `DCODE_SANDBOX_BACKEND` | enum | `auto` | `auto`, `seatbelt`, `bubblewrap`, `none`. `auto` escolhe pelo sistema operacional. `none` **só é aceito** junto de `DCODE_SANDBOX_MODE=full-access` — em qualquer outro modo é erro de inicialização, porque prometeria fronteira que não existe (RN-3). |
| `DCODE_SANDBOX_UNREADABLE` | lista de caminhos | **os cofres de credencial** | Caminhos que a sessão **não lê**, separados como `PATH`. `~` é expandido; o próprio home é recusado, porque cobrir tudo abaixo dele não é conjunto nomeado, é outro modo com nome emprestado. **Ausente usa o default abaixo; definir substitui**; o literal `none` não esconde nada. Ignorado em `full-access`, que não promete fronteira e não deve manter uma escondida. |

**Por que existe.** Leitura é concedida em todo lugar de propósito — recusá-la
impede o interpretador de carregar antes de o comando rodar — e o preço é que a
sandbox não protegia segredo nenhum: medido nesta base, um comando sob
`workspace-write` leu uma chave SSH privada sem um pio.

**O default esconde, sem ninguém pedir:**

```
~/.aws  ~/.gnupg  ~/.kube  ~/.config/gcloud  ~/.azure
~/.netrc  ~/.git-credentials  ~/.npmrc  ~/.pypirc  ~/.docker/config.json
<config-root>/credentials          ← a própria chave do dcode
```

O critério para entrar é estreito: **guarda segredo, e nenhuma ferramenta comum
precisa lê-lo como subprocesso de um comando comum.** É isso que torna esconder
gratuito. `aws` e `kubectl` são as exceções que aparecem primeiro, e são a razão
de a variável **substituir** a lista em vez de somar a ela.

**`~/.ssh` ainda não está no default, e a ausência é a linha mais alta desta
seção.** Chave privada é o segredo canônico — e escondê-la hoje pararia `git
push` e todo `ssh` de dentro do sandbox, porque o `ssh` lê a chave ele mesmo. A
saída é o agente: com `SSH_AUTH_SOCK` alcançável, o `ssh` pede ao agente para
assinar e nunca lê a chave, e aí esconder sai de graça. Esse socket é recusado
hoje pela regra que mantém runtime de contêiner de fora; conceder por nome é o
passo seguinte, e `~/.ssh` entra junto — não antes.

## 2. Rede

| Variável | Tipo | Default | Uso |
|---|---|---|---|
| `DCODE_ALLOW_NETWORK` | booleano | **`true`** | Concede rede sem escalonar, dentro do sandbox. Equivale a tratar rede como dentro da fronteira. **Desligar** devolve a pergunta, ao custo de tornar o shell impraticável onde ninguém pode responder. |

O default era `false`, e a consequência não estava escrita em lugar nenhum:
`bash` declara rede sempre — um comando é opaco, então declara-se o pior caso —
então **toda** chamada de shell escalonava. Com `approval.policy = never` isso
virava negação: build, teste e commit todos recusados, e um agente que edita sem
nunca verificar produz mudança que ninguém conferiu, que é pior que mudança
nenhuma.

Medido: numa execução autônoma real, 120 chamadas de ferramenta sem um único
comando executado. O modelo compensou conferindo por leitura — 73 `grep` para
responder o que um `go test` responderia.

Quem precisa da postura antiga põe `sandbox.allow_network = false` no arquivo de
configuração. A contenção não mudou: `read-only` continua sem rede, e conceder
não abre o que o modo fechou.

## 3. Governança

| Variável | Tipo | Default | Uso |
|---|---|---|---|
| `DCODE_REQUIREMENTS_FILE` | caminho | vazio | Configuração **travada** pelo administrador. Vazio procura `requirements.toml` na raiz de configuração e, não achando, não há camada travada — o caso normal de uma pessoa na própria máquina. Apontado explicitamente e ausente é **erro**: "não há política" e "a política não carregou" são fatos diferentes, e começar assim mesmo entregaria em silêncio toda permissão que o administrador quis reter. |
| `DCODE_CONFIRM_WRITE` | lista | `.git/**`, `.env*`, `.dcode/**`, arquivos de lock | Caminhos cuja **escrita** pede confirmação mesmo dentro do workspace. Não é contenção — é atenção: são coisas que parecem workspace e não são de mexer. |
| `DCODE_CONFIRM_READ` | lista | `.env*`, `**/*_rsa`, `**/credentials*` | Caminhos cuja **leitura** pede confirmação. |
| `DCODE_CONFIRM_COMMAND` | lista | ver abaixo | Comandos que pedem confirmação. Regra que casa texto de comando é regra que se contorna sem querer — por isso ela só **escalona** o que já era permitido, e nunca resgata o que foi negado. |

### O que pede confirmação por default

Esta linha declarava defaults desde sempre e `DefaultRules()` mandava lista
**vazia**: a promessa de confirmar antes de destruir existia na documentação e em
nenhum outro lugar.

Dois tipos, e estão aqui por motivos diferentes:

- **Destroem trabalho que o repositório não recupera** — `rm -rf`, `sudo`,
  `git push --force`, `git reset --hard`, `git clean -f`, `git branch -D`,
  `mkfs`, `dd … of=/dev/…`, `chmod -R 777`, `shutdown`.
- **Alcançam o mundo de forma irreversível** — `npm publish`, `cargo publish`,
  `gem push`, `twine upload`, `gh release create`, `docker push`. Publicar não é
  destruir, e é igualmente impossível de desfazer.

Também entram `curl … | sh` e `wget … | sh`: buscar e executar no mesmo fôlego é
confiança sem leitura.

**Isto não é fronteira e não pode ser.** Comando é texto, e a mesma destruição
sempre pode ser escrita de outro jeito — por script, por alias, por variável. O
que a lista compra é atrito contra o acidente, que é o que de fato acontece.
Contenção é o sandbox, e só ele.

> O arquivo de requisitos é o análogo do `requirements.toml` do Codex, elogiado na ADR-02 como o que torna o dcode adotável em organização. Separar config de usuário de config travada é o ponto inteiro.


## 4. Diagnóstico

| Variável | Tipo | Default | Uso |
|---|---|---|---|

## 5. Constantes não configuráveis

| Constante | Valor | Motivo |
|---|---|---|
| Falha fechada quando o mecanismo não está disponível | sempre | RN-3; degradar silenciosamente promete o que não entrega. |
| Resolução de symlink antes da comparação | sempre | RN-4; sem isso a fronteira é contornável com um `ln -s`. |
| Comparação de contenção por componente | sempre | Prefixo de string deixa `/proj2` passar por `/proj`. |
| Passagem obrigatória pelo avaliador | sempre | RN-6; caminho alternativo é blocker. |
| `never` nega o que cruzaria | sempre | A alternativa seria conceder em silêncio. |
| `never` nega o que uma regra escalonaria | sempre | Antes as regras eram puladas sob `never`, e a única configuração sem ninguém olhando era a única que não parava diante de `rm -rf /`. A autorização expressa não chega, então a resposta é não. |

## 6. Changelog

- [202608190030 — Trabalho comum não pergunta, destruição pergunta sempre](changelog/202608190030-trabalho-comum-nao-pergunta.md)


_Sem alterações desde a criação._
