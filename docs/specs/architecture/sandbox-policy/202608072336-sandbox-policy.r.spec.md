# Research: Sandbox e Política de Permissão

> Fonte da verdade de negócio para a fronteira de execução e para a decisão de autorização.
> Decisão de arquitetura de origem: **ADR-02 — Sandbox e aprovação são preocupações separadas**.

## 1. Contexto

Um agente com acesso a shell é superfície de ataque. A ADR-02 decidiu copiar do Codex o modelo mais bem desenhado da categoria, com duas camadas ortogonais:

- **Sandbox** — fronteira técnica, aplicada pelo sistema operacional.
- **Política de aprovação** — decisão de autorização, independente da fronteira.

Manter as duas separadas é o que reduz fadiga de aprovação. Harness que mistura pergunta demais, o usuário desliga o prompt inteiro, e a segurança vira decoração. Esse é o modo de falha real — não o ataque sofisticado, mas o usuário exausto.

Este componente também é metade da tese do produto. jcode tem a performance e nenhum sandbox; sem esta camada, o projeto vira jcode em Go.

## 2. Fronteira de determinismo

**Regime: determinístico.**

Avaliação de política e aplicação de fronteira são regra explícita. Não há mediação por modelo: dado um pedido de execução e uma configuração, a decisão é sempre a mesma.

**Consequência para a revisão:** tudo aqui é verificável por asserção. O `.p.spec.md` não tem seção de contratos comportamentais.

Isto é deliberado e inegociável: **segurança mediada por modelo não é segurança.** Se a decisão de permitir dependesse do julgamento do modelo, um prompt adversarial a contornaria.

## 3. User stories

| # | Como | Quero | Para |
|---|---|---|---|
| US-1 | usuário | que o agente edite meu projeto sem pedir permissão a cada arquivo | trabalhar sem fadiga de aprovação |
| US-2 | usuário | ser perguntado antes de algo sair do meu projeto | rede e escrita fora do workspace são decisões minhas |
| US-3 | usuário | que uma instrução maliciosa num arquivo lido não escape da fronteira | a fronteira não pode depender do julgamento do modelo |
| US-4 | usuário | rodar em modo só-leitura quando estou explorando código alheio | inspecionar sem risco |
| US-5 | administrador | travar o modo de sandbox para toda a equipe | política organizacional não ser contornável por env var |

## 4. Regras de negócio

### RN-1 — Sandbox e aprovação são ortogonais
São dois eixos independentes. O sandbox define **o que é fisicamente possível**; a política define **o que exige consentimento**.

| | `read-only` | `workspace-write` | `full-access` |
|---|---|---|---|
| Ler dentro do workspace | livre | livre | livre |
| Escrever no workspace | **impossível** | livre | livre |
| Escrever fora | **impossível** | escalona | livre |
| Rede | **impossível** | escalona | livre |

"Livre" e "escalona" são política; "impossível" é sandbox. Um cliente **nunca** amplia o sandbox — só responde dentro da política vigente.

### RN-2 — A fronteira é aplicada pelo sistema operacional
Validação de caminho dentro do processo **não é sandbox**. O agente executa comandos arbitrários; qualquer verificação em Go é contornável pelo próprio comando que ela permitiu.

Validação em processo continua existindo como primeira linha — ela dá erro melhor e mais rápido — mas nunca é a garantia.

### RN-3 — Falha fechada, sempre
Sandbox que não pôde ser estabelecido não degrada para execução sem sandbox. A sessão falha na criação, com erro explicando o que faltou.

Um harness que silenciosamente roda sem fronteira quando o mecanismo não está disponível é pior que um sem sandbox nenhum: promete o que não entrega.

### RN-4 — O workspace é a unidade de confiança
A raiz do sandbox é o workspace declarado na criação da sessão. Caminho é resolvido — links simbólicos inclusive — antes de qualquer comparação. `../` e symlink apontando para fora são cruzamento de fronteira, não atalho.

### RN-5 — Escalonamento é por operação, não por sessão
Aprovar uma escrita fora do workspace não aprova as próximas. Aprovação de escopo de sessão existe como escolha explícita do usuário (`allow_session`), nunca como default.

### RN-6 — Nenhuma execução contorna o avaliador
Todo comando e toda ferramenta com efeito passam pelo avaliador de política antes de executar. Não existe caminho alternativo, nem em teste, nem em depuração.

### RN-8 — O modo é um nome para um par, e é derivado dele
Quem usa a ferramenta escolhe autonomia, não coordenadas: `plan`, `assist` e
`auto` são os nomes dos três pares de (sandbox, política) que a §2.1 do `.p`
fixa. Os eixos continuam ortogonais — o que os modos acrescentam é um vocabulário
por cima deles, não uma terceira dimensão.

O nome é sempre **calculado a partir do par em vigor**, nunca guardado ao lado.
Nome guardado à parte é nome que diverge, e um crachá que diz `assist` sobre uma
sessão sem fronteira é pior que crachá nenhum: ele convida a confiar.

Par que não corresponde a nenhum dos três não é aproximado para o vizinho — fica
sem nome. E trocar de modo não interrompe o turno em andamento: ajustar
autonomia não é cancelar trabalho.

### RN-7 — Política do administrador vence a do usuário
Configuração travada por administrador não é sobrescrevível por variável de ambiente nem por flag. É o que torna o dcode adotável em organização.

## 5. Fora de escopo

- Isolamento por container ou máquina virtual — ADR-02 adiou; custo de startup incompatível com a ADR-01.
- Autenticação e transporte remoto.
- Varredura de conteúdo em busca de instrução adversarial. A defesa é a fronteira, não a detecção.
- Windows no MVP — a spec define o comportamento, a implementação vem depois. **O release não publica binário Windows**: `internal/sandbox` não tem backend ali, então `New` falha fechado e o binário não consegue criar sessão. Publicar artefato que instala, verifica e recusa a primeira coisa que se pede é pior que não publicar — e `TestThePublishedMatrixIsExactlyWhatCanRun` impede a matriz de divergir de novo.

## 6. Changelog

_Sem alterações desde a criação._
