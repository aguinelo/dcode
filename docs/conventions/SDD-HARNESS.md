# Spec-driven development applied to an agent harness

🇧🇷 [Versão em português](SDD-HARNESS.pt-BR.md)

> **This document does not modify RPI.** The canonical ArcaSolucoes protocol — four
> `.r`/`.p`/`.config`/`.i` files, the `.r` > `.p`/`.config` > `.i` precedence order,
> timestamp naming, spec↔code synchrony, changelog, zero tolerance for data loss —
> applies here in full and without exception.
>
> What this guide answers is **how to fill those four files** when the product is an
> agent harness, where part of the behavior is mediated by a language model. No new
> artifact, no new rule.

---

## 1. The problem

The golden rule of `.p.spec.md` is *"use exactly the names, fields and types defined"*.
That settles API contracts and data models. It does not settle this:

> when an edit fails on an ambiguous match, the agent re-reads the file instead of
> retrying blind

That is model-mediated behavior. It is not a schema, and it is not verifiable by
assertion.

The temptation is to invent a fifth file for it. **That is the wrong way out** — it would
diverge from the protocol shared across repositories and break the tooling that audits
RPI. The right way out is to notice that a behavioral contract with a threshold **already
is a technical contract**, and technical contracts are exactly what `.p` holds.

---

## 2. Where each concern goes

| Concern | Canonical file | Why |
|---|---|---|
| Which behavior is deterministic and which is model-mediated | `.r.spec.md` | It is domain truth about what the system is. Context, not contract. |
| Behavior scenarios with thresholds | `.p.spec.md` | A threshold is a verifiable technical contract. Same standing as a schema. |
| Model and version the threshold was measured against; thresholds as constants | `.config.spec.md` | It is environment definition — varies per environment, like a feature flag. |
| Building the eval suite and its fixtures | `.i.spec.md` | It is an execution step, with order and dependencies. |
| Stability level of a public contract | `.p.spec.md` | It is a property of the contract. RPI's `changelog/` already supplies the change semantics. |

---

## 3. Determinism boundary in `.r.spec.md`

Every `.r.spec.md` in this project declares which regime its scope operates in:

| Regime | Meaning | How it is verified |
|---|---|---|
| **Deterministic** | behavior defined by explicit rule | assertion in `go test` |
| **Model-mediated** | behavior emerges from interaction with the LLM | statistical threshold over fixtures |
| **Mixed** | the spec covers both; the section says **where** the line falls | both, separately |

Without that declaration, review applies the wrong standard to the wrong artifact — it
demands assertions for statistical behavior, or accepts a threshold where a guarantee was
available.

**Architectural corollary:** pushing behavior to the deterministic side is a design goal,
not an accident. If context assembly is a pure function `(session state) → []Message`, it
is exactly golden-testable — and the append-only decision (ADR-03) already makes that
natural, because the prefix is a pure function of history. The same holds for tool
dispatch, sandbox decisions and tool-call parsing.

---

## 4. Behavioral contracts in `.p.spec.md`

When `.r` classifies the scope as model-mediated or mixed, `.p` gains a **"Behavioral
contracts"** section with a scenario-and-threshold table. It is an ordinary `.p` section,
subject to the same golden rule: scenario identifiers are exact names, used as such in
code and fixtures.

| ID | Scenario | Expected behavior | Threshold | Fixture |
|---|---|---|---|---|
| `edit-ambiguous` | `Edit` with an ambiguous match | re-reads the file, no blind retry | ≥ 95% | `testdata/evals/edit-ambiguous/` |
| `path-missing` | nonexistent path | explicit error, no invented path | 100% | `testdata/evals/path-missing/` |
| `compaction-long` | compaction during a long task | current task survives the cut | ≥ 98% | `testdata/evals/compaction-long/` |

**Rules of use:**

1. Verifying a behavioral contract is measurement against a threshold, never a boolean.
2. A 100% threshold is only legitimate when the behavior is in fact deterministic. In
   that case, question whether the scenario belongs in another `.p` section, verifiable
   by assertion.
3. Measurement depends on a real model and costs money: it sits behind a build tag or
   `testing.Short()`, outside the standard `go test` run.
4. A regression below the threshold is a pull request blocker, same as a red test.
5. **Lowering a threshold in the same pull request that broke it is the anti-pattern this
   section exists to catch.** A threshold change is a rule change and requires a
   `changelog/` entry, per section 6 of `RPI-SPEC-RULES.md`.

The model and version a threshold was measured against live in `.config.spec.md` —
switching models invalidates the threshold, not the scenario.

---

## 5. Stability level in `.p.spec.md`

`sales-api` is internal: breaking a contract is a coordination problem. Here, three things
are contracts with third parties — the **client-server protocol**, the **plugin ABI** and
the **config schema**.

Every `.p.spec.md` that defines a public contract declares its level in the first section:

| Level | Meaning |
|---|---|
| `experimental` | may break in any version, without a changelog entry |
| `stable` | breaking it requires a `changelog/` entry + major version bump |
| `frozen` | does not change; additive extension only |

The level applies to the whole spec, and an individual endpoint or symbol may declare a
more restrictive level of its own. Promotion criteria are written into `.i.spec.md` — it
is not a spur-of-the-moment decision.

This creates no new mechanism: RPI's `changelog/` is already the record of rule changes.
The declaration only says **which** changes require one.

---

## 6. Effect on tooling

None. Specs remain four `.spec.md` files under `docs/specs/**` sharing a timestamp
prefix. The `embarca-pr-review` skill audits this project with no modification — what it
validates is the taxonomy, and the taxonomy is intact.

The Go-specific review checklist lives in `docs/conventions/GO-CODE-REVIEW.md`, in this
repository, as a project convention.
