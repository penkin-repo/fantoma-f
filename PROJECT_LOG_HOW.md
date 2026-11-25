# PROJECT_LOG_HOW — How to Fill the Project Log

## Where Is Everything?

| File | Location | Purpose |
|---|---|---|
| `MASTER_PROMPT.md` | `litefont/MASTER_PROMPT.md` | Original design spec + all architectural decisions |
| `PROJECT_LOG.md` | `litefont/PROJECT_LOG.md` | Living changelog — bugs, stages, changes |
| `PROJECT_LOG_HOW.md` | `litefont/PROJECT_LOG_HOW.md` | This file — how to write the log |

---

Не забывай проставлять версию внутри программы в собранном файле чтобы отображалось
и можешь менять масетр промпт или project how если на то есть причины  

## Version Registry Rule

**При каждом релизе обновляй версию в одном месте:**

| Файл | Где | Что менять |
|---|---|---|
| `version.go` | `const AppVersion = "X.Y.Z"` | Единственный источник правды |
| Заголовок окна | `main.go` → `Title: "Fantoma F " + AppVersion` | Автоматически из `AppVersion` |
| Трей | `tray_windows.go` → tooltip | Автоматически из `AppVersion` |
| Настройки (About) | `SettingsView.jsx` → `GetAppVersion()` | Автоматически из `AppVersion` |

**Правило:** меняй только `version.go` — всё остальное подтягивается автоматически.

Версия отображается:
- В заголовке окна: `Fantoma F 1.3.1`
- В трее: `Fantoma F v1.3.1 — Font Manager`
- В Settings → About → Version: `v1.3.1`

## How to Fill PROJECT_LOG.md

### New Version / Stage Entry

Add a new `## [vX.Y.Z]` section at the **top** of the changelog (newest first):

```markdown
## [v0.2.0] — YYYY-MM-DD — Short Description

### Stage: What was done in this stage

**What was implemented:**
- Short bullet list of features / fixes

**Files changed:**
- `app.go` — added X method
- `FontCard.jsx` — added Y feature

**Dependencies added/removed:**
- `some/package v1.2.3` — reason
```

### Bug Entry

Add to the **Bug Log** section in `PROJECT_LOG.md`:

```markdown
## Bug Log

### [BUG-001] — YYYY-MM-DD — Short description
- **File:** `font_windows.go:42`
- **Symptom:** What the user saw
- **Root cause:** What actually went wrong
- **Fix:** What was changed to fix it
- **Status:** ✅ Fixed / ⏳ Open / 🚫 Won't Fix
```

### Architectural Decision

When a significant design decision is made, add it to `MASTER_PROMPT.md` under the **Key Technical Decisions** table AND add a note in `PROJECT_LOG.md`.

---

## Rules

1. **Newest entries at the top** of the changelog section
2. **Never delete old entries** — mark as superseded if needed
3. **Always link to files** when mentioning code changes
4. **Bug numbers are sequential** (BUG-001, BUG-002, …)
5. **Keep MASTER_PROMPT.md in sync** when architecture changes

---

## Version Numbering

`vMAJOR.MINOR.PATCH`

- PATCH → bug fixes, typos, style tweaks
- MINOR → new features, refactors
- MAJOR → breaking changes, full rewrites of a component
