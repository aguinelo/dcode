# Language policy

🇧🇷 [Versão em português](LANGUAGE.pt-BR.md)

This project is bilingual: English and Brazilian Portuguese. This document is the rule.

---

## 1. File naming

English is the default and takes the bare filename. Portuguese is the translation and
carries the `.pt-BR` suffix.

```
README.md                       English (canonical)
README.pt-BR.md                 Portuguese

docs/conventions/TESTING.md         English (canonical)
docs/conventions/TESTING.pt-BR.md   Portuguese
```

Every translated pair cross-links to its counterpart in the first line after the title,
so a reader who lands on either one can switch.

## 2. What gets both languages

| Artifact | Languages | Canonical |
|---|---|---|
| `README` | both | English |
| `docs/conventions/**` | both | English |
| `docs/brand/**` | both | English |
| Issue and PR templates | both | English |
| Code comments | English only | — |
| Commit messages and PR titles | English only | — |
| `docs/specs/**` (RPI) | **Portuguese only** | Portuguese |

## 3. Why specs stay in one language

This is the deliberate exception, and it is not an oversight.

The RPI protocol makes `.r.spec.md` the **absolute truth** — if the code contradicts it,
the code is wrong. That rule only works with exactly one source of truth. Two copies of
a spec will drift, and the moment they disagree there is no way to tell which one the
code is supposed to satisfy. A drifted spec is worse than a missing one, because it
looks authoritative.

Specs are also internal working documents, not the public face of the project. The
audience for a `.p.spec.md` is whoever is implementing it, and that audience already
reads Portuguese.

The canonical RPI rules require Portuguese, and this project does not modify RPI.

**If an external contributor ever needs a spec in English, translate it in the pull
request that needs it and mark the translation explicitly as non-canonical** — never as
a parallel source of truth.

## 4. Why commits stay in one language

Commit messages feed changelog generation and are parsed by tooling that assumes a
single language. Bilingual commit bodies double the noise in `git log` without helping
anyone: contributors who read the code already read English, since the code and its
comments are English.

Conventional Commits type prefixes (`feat`, `fix`, `docs`) are English identifiers
regardless.

## 5. Keeping translations in sync

A translation that lags is a translation that lies.

- A pull request that changes a document **must** update both language versions in the
  same pull request. Changing only one is blocked in review.
- When they disagree, the canonical version wins and the translation is the bug.
- Translate meaning, not words. Technical terms of art stay in English inside the
  Portuguese text — *append-only*, *sandbox*, *cache*, *golden file* — because
  translating them makes the text harder to read for the people who actually use them.
