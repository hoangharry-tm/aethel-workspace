# Task 19 — README Landing Page Design

**Purpose:** Transform the project README into a premium, landing-page-quality document that serves two audiences simultaneously: potential contributors and evaluators who discover the repo, and IT administrators making a procurement or deployment decision.

**Design direction:** Restrained luxury — Mercedes-Benz black/white aesthetic. No marketing hyperbole. Confident technical depth. Let the actual implementation choices (Argon2id, cryptographic hash chains, monthly-partitioned audit ledger) do the selling.

**Status:** ✅ COMPLETE — executed directly in this session.

**Outputs:**
- `README.md` — full landing-page README (executed 2026-06-06)
- `docs/assets/banner.svg` — black/white luxury wordmark (executed 2026-06-06)

---

## What Was Built

### Banner SVG (`docs/assets/banner.svg`)
- Pure black (`#0a0a0a`) background
- "AETHEL" in weight-200 system sans-serif, widely letter-spaced
- "WORKSPACE" subtitle in muted `#484848`, 10-tracking
- Corner accent marks (thin `#383838` lines) — the luxury framing detail
- No external font dependencies — renders identically in all browsers and GitHub dark/light mode

### README Structure
1. **Hero** — full-width SVG banner + centered tagline + badge row
2. **Stats bar** — 6-cell HTML table: pages / roles / endpoints / migrations / pillars / p99 latency
3. **"Self-hosted by design"** — `[!IMPORTANT]` callout — the key product differentiator
4. **Three Pillars** — HTML table with technical substance, not marketing copy
5. **Architecture** — Mermaid flowchart showing full system topology
6. **Technical Highlights** — three GitHub callout boxes: runtime config, cryptographic chains, Argon2id
7. **Quick Start** — four commands, service URL table, bootstrap-admin tip
8. **Tech Stack** — shields.io badge grid organized by layer
9. **Feature Reference** — three `<details>` sections (one per pillar), deep technical detail
10. **Configuration** — two `<details>` sections: env vars + blueprint files
11. **Command Reference** — collapsible full make target list
12. **Project Structure** — collapsible annotated directory tree
13. **Documentation** — table linking all architecture/guide docs
14. **Security, Contributing, License** — brief with links to respective files
15. **Footer** — centered links row

---

## What Requires Manual Action from the User

### 1. Demo GIF / Video (high impact)
A short (15–30s) screen recording showing: login → create dispatch → routing rule fires → append green note → view audit log. This is the single highest-ROI asset to add.

**Free tools:**
- **Kap** (Mac) — `getkap.co` — records screen, exports GIF or MP4, free
- **ScreenToGif** (Windows) — `screentogif.com` — free
- **Gifski** — `gif.ski` — CLI tool that converts video to high-quality GIF, free

To embed: add `![Demo](docs/assets/demo.gif)` inside the hero `<div align="center">` block.

### 2. Screenshots
Static screenshots of the dashboard, document detail page, and audit log give the README visual weight that text cannot.

**Free tools:**
- Any browser's built-in screenshot (cmd+shift+4 on Mac)
- **Carbon** — `carbon.now.sh` — beautiful code/terminal screenshots with dark themes

To embed: add `<img src="docs/assets/screenshot-dashboard.png" width="49%"/>` etc. inside a centered div.

### 3. Logo Design (optional upgrade)
The current SVG banner is text-only (intentionally clean). If you want a geometric mark (like a stylized "Æ" or document icon), use:

**Free tools:**
- **Figma** (already installed) — design a mark, export as SVG, replace `docs/assets/banner.svg`
- **Canva** — free tier, good for logo-type lockups

### 4. GitHub Repository Badges (upgrade once public)
Replace the static badges with live ones once the repo is on GitHub:
- Build status: `![CI](https://github.com/your-org/aethel-workspace/actions/workflows/ci.yml/badge.svg)`
- Stars: `![Stars](https://img.shields.io/github/stars/your-org/aethel-workspace?style=flat-square)`

### 5. Live Demo URL (optional)
If you deploy a staging instance, add a `[![Live Demo](https://img.shields.io/badge/demo-live-10b981?style=flat-square)](https://demo.aethel.example.com)` badge to the hero section.

---

## What Was Done Automatically

- Full README.md rewrite with all sections
- SVG banner (black/white luxury design, no external dependencies)
- All shields.io badges (static, render immediately)
- Mermaid architecture diagram (GitHub renders natively, no image needed)
- GitHub native callout boxes (`[!NOTE]`, `[!TIP]`, `[!IMPORTANT]`)
- All technical copy written from actual project knowledge
- Collapsible sections for secondary content (commands, structure, config)
- Footer with links to SECURITY, CODE_OF_CONDUCT, CHANGELOG

---

## If You Want to Extend Further

- **Benchmark table**: Add actual benchmark results from Task 18 once they're run (`go test -bench=. ./...`)
- **Comparison table**: "Aethel vs generic workflow tools" — document what you get that Jira/Trello/Notion don't have (cryptographic audit chains, self-hosted, diarization model)
- **i18n note**: Add a badge or note once the Vietnamese locale (Task 17) is confirmed working
- **Star history chart**: `star-history.com` generates a free chart once the repo has stars
