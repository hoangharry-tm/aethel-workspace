# Task 19 — README Landing Page

**Purpose:** Design and write the project README from scratch as a premium, landing-page-quality document. The README is the project's front page — it must simultaneously impress a developer stumbling onto the repo AND convince an IT administrator that this software is worth deploying. No code changes. Only README.md and supporting SVG assets.

**Primary output:** `README.md` at the repo root
**Secondary output:** `docs/assets/banner.svg` — the project wordmark

**Design direction:** Restrained luxury — think Mercedes-Benz annual report, not a SaaS startup landing page. Black and white. Clean geometry. No marketing superlatives ("best-in-class", "enterprise-grade", "battle-tested"). No fake social proof. The actual technical choices — Argon2id, SHA-256 hash chains, monthly-partitioned audit ledger — ARE the marketing. Let them speak.

---

## Execution Flow

```
[Step 0]  Pre-flight → collect all project context into /tmp/t19-preflight.md
               ↓
[Parallel] Agent 1 — Brand Strategist        → /tmp/t19-agent-1.md  (voice, hierarchy, IA)
           Agent 2 — Technical Content Writer → /tmp/t19-agent-2.md  (prose, diagram, features)
           Agent 3 — Visual Asset Designer    → /tmp/t19-agent-3.md  + docs/assets/banner.svg
               ↓ all complete
[Serial]   Agent 4 — Synthesis               → README.md (final assembly)
               ↓
[Serial]   Agent 5 — QA Reviewer             → quality pass + commit
```

Agents 1, 2, and 3 run fully in parallel. Each writes only to its assigned output. Agent 4 reads all three and assembles the final file. Agent 5 reviews and commits.

---

## Step 0 — Pre-flight (run in orchestrating session, not a subagent)

```bash
REPO=/Users/hoangharry/mh_code/internships/Bravo/aethel-workspace
cd "$REPO"

echo "## Current README content" > /tmp/t19-preflight.md
cat README.md >> /tmp/t19-preflight.md

echo -e "\n## CLAUDE.md — project overview (first 120 lines)" >> /tmp/t19-preflight.md
head -120 CLAUDE.md >> /tmp/t19-preflight.md

echo -e "\n## API route count" >> /tmp/t19-preflight.md
grep -c "operationId" docs/architecture/architecture-api-routes.md >> /tmp/t19-preflight.md || echo "N/A"

echo -e "\n## Migration count" >> /tmp/t19-preflight.md
find aethel-core/internal/database/migrations -name "*.up.sql" | wc -l >> /tmp/t19-preflight.md

echo -e "\n## Go module" >> /tmp/t19-preflight.md
head -5 aethel-core/go.mod >> /tmp/t19-preflight.md

echo -e "\n## Frontend pages" >> /tmp/t19-preflight.md
find aethel-view/app/pages -name "*.vue" | sort >> /tmp/t19-preflight.md

echo -e "\n## Tech stack (go.mod direct deps)" >> /tmp/t19-preflight.md
grep -A 40 "^require (" aethel-core/go.mod | head -40 >> /tmp/t19-preflight.md

echo -e "\n## Security architecture summary" >> /tmp/t19-preflight.md
head -60 docs/architecture/architecture-security.md >> /tmp/t19-preflight.md

echo -e "\n## Git HEAD" >> /tmp/t19-preflight.md
git rev-parse --short HEAD >> /tmp/t19-preflight.md

echo "Pre-flight complete."
```

---

## Agent 1 — Brand Strategist

**You are a senior brand strategist** who has written positioning copy for developer infrastructure products (HashiCorp Vault, Tailscale, PlanetScale). You know the difference between marketing copy that embarrasses engineers and copy that earns their trust.

**Your output is NOT the README itself.** Your output is the strategic blueprint that Agents 2, 3, and 4 will use as their north star.

### Step 1: Read and absorb the project

```bash
cat /tmp/t19-preflight.md
cat /Users/hoangharry/mh_code/internships/Bravo/aethel-workspace/CLAUDE.md | head -80
cat /Users/hoangharry/mh_code/internships/Bravo/aethel-workspace/docs/architecture/architecture-security.md | head -80
```

### Step 2: Define the two audiences and their questions

Think through: a developer who finds this repo on GitHub has three questions in the first 10 seconds:
1. What does this do?
2. Is this serious software or a hobby project?
3. Is it relevant to me?

An IT administrator evaluating Aethel for deployment has different questions:
1. Can I trust this to handle sensitive institutional documents?
2. How much operational overhead will this create?
3. What are my obligations if something goes wrong?

Your job is to define how the README answers both sets of questions, in what order, and with what tone.

### Step 3: Define the messaging hierarchy

Write the following in your output:

**A. The single-sentence positioning statement**
One sentence that captures what Aethel is, for whom, and why it's different from generic workflow tools. This is not a tagline. It is the statement every other piece of copy must be consistent with.

**B. The three proof points**
The three most technically credible things about this project — things that demonstrate engineering discipline, not just feature completeness. These must be true, specific, and non-trivial. "Built with Go" is not a proof point. "Refresh tokens rotate atomically using `BeginTx/Commit` to prevent session fixation" is.

**C. The differentiation frame**
One short paragraph that articulates what Aethel is NOT. What category of tool does it superficially resemble? Why is that comparison wrong? This helps the reader recalibrate their mental model quickly.

**D. The tone rules** (what Agent 2 must follow)
- 3 things to ALWAYS do in the copy
- 3 things to NEVER do in the copy
- One sentence describing the voice: "This README sounds like ___"

**E. Information architecture** — ordered list of README sections
Write the complete list of sections in the order they should appear, with a one-sentence rationale for each section's position. Be opinionated. The ordering is a design decision.

### Output format

Write your full strategic brief to `/tmp/t19-agent-1.md`. No section headers other than those defined above. Prose where possible. Be concise — this is a brief, not an essay.

---

## Agent 2 — Technical Content Writer

**You are a senior technical writer** who has written documentation for Go projects, PostgreSQL-backed systems, and developer infrastructure tools. You write like an engineer, not a marketer. Your prose is precise, specific, and respects the reader's intelligence.

**Your job:** Write every prose section of the README — the descriptions, the feature explanations, the architecture narrative, and the Mermaid diagram code. You do NOT write HTML layout, badges, or visual elements. Agent 3 handles those. You write the words and the diagram.

### Step 1: Read context

```bash
cat /tmp/t19-preflight.md
cat /tmp/t19-agent-1.md   # Read Agent 1's strategic brief — follow their tone rules
cat /Users/hoangharry/mh_code/internships/Bravo/aethel-workspace/CLAUDE.md
cat /Users/hoangharry/mh_code/internships/Bravo/aethel-workspace/docs/architecture/architecture-api-routes.md | head -60
cat /Users/hoangharry/mh_code/internships/Bravo/aethel-workspace/docs/architecture/architecture-security.md | head -100
```

### Step 2: Write content blocks

Write each of the following content blocks exactly as they should appear in the final README. Label each block clearly so Agent 4 (Synthesis) knows where to place it.

---

**BLOCK: TAGLINE**
One sentence, centered beneath the banner. Not a slogan. A truthful, precise description of what Aethel is and for whom. Refer to Agent 1's positioning statement. Maximum 20 words.

---

**BLOCK: OPENING**
2–3 paragraphs. No bullet points. Flowing prose. Answer: what is the actual problem this solves, what happens without it, and what is the approach Aethel takes. Write as if the reader is a competent engineer who has seen dozens of SaaS workflow tools and is skeptical. Do not use the words: "seamless", "powerful", "robust", "scalable", "enterprise", "world-class", "cutting-edge", "next-generation", "revolutionize", "game-changing".

The third paragraph must mention the self-hosted nature directly. Use a factual tone: "Aethel runs on your server. Your database. Your network perimeter." Not: "Aethel gives you full control over your data."

---

**BLOCK: THREE PILLARS TABLE CONTENT**
Three cells of text, one per pillar: DAK Diarization, Green Noting Canvas, RBAC Audit Ledger. Each cell: 3–5 sentences. Technical depth. Mention specific implementation details where they increase trust (e.g., name the hash function, mention the partition strategy, name the SQL extension). Do not write generic feature descriptions.

---

**BLOCK: MERMAID ARCHITECTURE DIAGRAM**
Write a `flowchart LR` Mermaid diagram showing the full system topology. Must include:
- Browser (Nuxt 4 SSR)
- Go backend with the 9-layer middleware stack named
- Config cache with TTL
- SSE Broker
- Escalation Worker
- PostgreSQL 16 with key characteristics
- YAML blueprints (seed only path)
- Data flow arrows with labels (JWT Bearer, text/event-stream, first-boot seed, etc.)

Use subgraph blocks to group related nodes. Keep node labels concise but informative. The diagram should convey the architecture to someone who has never read the code.

---

**BLOCK: TECHNICAL HIGHLIGHTS**
Three callout box contents (these will be rendered as `[!NOTE]`, `[!TIP]`, `[!IMPORTANT]` by Agent 4):

1. *Runtime configuration* — explain how the config API works, the 5-minute cache, and why this matters operationally (no restarts, no rebuilds)
2. *Cryptographic document chains* — explain the SHA-256 hash chain construction precisely: `hash(content ‖ sequence ‖ author ‖ previous_hash)`. Mention what this enables (tamper detection at the row level) and draw the analogy to Git's object model.
3. *Argon2id* — explain why Argon2id over bcrypt (RFC 9106, winner of the Password Hashing Competition, memory-hard), what the defaults mean (`65536 KiB memory · 3 iterations · 4 threads`), and that the parameters are blueprint-configurable.

Each callout: 3–5 sentences. Dense but readable.

---

**BLOCK: QUICK START COMMENTARY**
Two sentences maximum, placed before the code block. Must answer: "what do I need before I start" and "how long does this take."

---

**BLOCK: FEATURE DEEP DIVES** (collapsible section content)
Three collapsible sections, one per pillar. Each section: 200–350 words. Include:
- How the feature works end-to-end
- Any non-obvious design decisions (with the reasoning)
- A code snippet or JSON example showing a real API response or database structure (use realistic fake data)
- A brief mention of what happens when things go wrong (the error handling path)

---

**BLOCK: SECURITY SUMMARY**
2 sentences max. Links to `SECURITY.md` and `architecture-security.md`. Names 4–6 specific controls (no generic ones like "secure by design").

---

**BLOCK: CONTRIBUTING NOTE**
3 sentences. Mention the no-mock-DB integration test policy specifically — it is an unusual and credible constraint worth calling out.

---

Write everything to `/tmp/t19-agent-2.md`. Label each block with its name in `## BLOCK: NAME` format so Agent 4 can extract them programmatically.

---

## Agent 3 — Visual Asset Designer

**You are a senior front-end designer** who specializes in high-quality GitHub READMEs and developer documentation aesthetics. You understand what GitHub's markdown renderer actually supports — no guessing, no CSS that gets stripped.

**Your job:** Create the SVG banner, design the badge row, and write all HTML layout blocks. You do not write prose content — Agent 2 handles that.

### Step 1: Read context

```bash
cat /tmp/t19-preflight.md
cat /tmp/t19-agent-1.md   # Read the tone rules and IA — your visual choices must reinforce them
```

### Step 2: Create `docs/assets/banner.svg`

```bash
mkdir -p /Users/hoangharry/mh_code/internships/Bravo/aethel-workspace/docs/assets
```

Design a banner SVG at `docs/assets/banner.svg`. Requirements:

- **Viewport:** `width="1280" height="220"` — renders full-width on GitHub
- **Background:** `#0a0a0a` — near-black, not pure black (less harsh on dark mode)
- **Wordmark:** "AETHEL" — system sans-serif (`-apple-system, 'Helvetica Neue', Helvetica, Arial, sans-serif`), weight 200 (ultralight), font-size 72–80px, fill `#f0f0f0`, letter-spacing 25–30, centered at x=640
- **Subtitle:** "WORKSPACE" — same font, weight 400, font-size 11–13px, fill `#484848`, letter-spacing 8–10, centered below the wordmark
- **Accent rule:** A single 1px horizontal line (`#282828`) between the wordmark and subtitle
- **Corner marks:** Four L-shaped corner accents (40px legs, 1.5px stroke, `#383838`). One at each corner of an inset rectangle. This is the luxury framing detail — do not skip it.
- **No gradients, no drop shadows, no external fonts, no images.** The elegance comes from geometry and negative space, not decoration.

After writing the file, verify it is valid SVG: check the file contains `<svg` and `</svg>`.

### Step 3: Design the badge row

Write the badge row as a markdown snippet that will be placed inside a `<div align="center">` by Agent 4. Use shields.io badges with `style=flat-square`.

Badges to include (in this order):
1. Go version — `https://img.shields.io/badge/Go-1.26-00ADD8?style=flat-square&logo=go&logoColor=white`
2. Nuxt version — `https://img.shields.io/badge/Nuxt-4-00DC82?style=flat-square&logo=nuxt.js&logoColor=white`
3. PostgreSQL version — `https://img.shields.io/badge/PostgreSQL-16-336791?style=flat-square&logo=postgresql&logoColor=white`
4. License — `https://img.shields.io/badge/License-Apache_2.0-4f46e5?style=flat-square`
5. OpenAPI — `https://img.shields.io/badge/OpenAPI-3.1_·_58_endpoints-6BA539?style=flat-square&logo=openapiinitiative&logoColor=white`
6. Docker — `https://img.shields.io/badge/Docker-ready-2496ED?style=flat-square&logo=docker&logoColor=white`
7. Self-hosted — `https://img.shields.io/badge/self--hosted-by_design-0a0a0a?style=flat-square`

### Step 4: Design the stats bar

Write a centered HTML table with 6 cells: pages, roles, API endpoints, migrations, domain pillars, p99 latency target. Each cell: bold number on top, `<sub>` label below. Use real numbers from the preflight context.

### Step 5: Design the tech stack section

Write a `<table>` with rows for: Frontend, Backend, Database, Auth, Infrastructure, Testing. Each row: left cell = bold layer name, right cell = shields.io badges for that layer's technologies.

Badge colors should use the technology's official brand colors where available. Use the same `style=flat-square` throughout for visual consistency.

### Step 6: Write the "placeholder" comment blocks

At two specific locations in your output, write HTML comments marking where the user must insert their own assets:

```html
<!-- 📸 INSERT DEMO GIF HERE
     Suggested: 15-30s recording of login → dispatch creation → green note → audit log verify
     Free tools: Kap (Mac, getkap.co) · ScreenToGif (Windows) · Gifski (CLI, gif.ski)
     Embed with: <img src="docs/assets/demo.gif" width="100%" alt="Aethel Workspace Demo"/>
-->
```

```html
<!-- 📸 INSERT SCREENSHOTS HERE (optional)
     Suggested: dashboard.png, document-detail.png, audit-log.png
     Use Carbon (carbon.now.sh) for code screenshots with dark theme
     Embed with: <img src="docs/assets/screenshot-dashboard.png" width="49%"/>
-->
```

Place the first placeholder after the badges in the hero section. Place the second inside the Quick Start section after the service URL table.

### Output

Write all HTML/markdown blocks (badge row, stats bar, tech stack table, placeholder comments) to `/tmp/t19-agent-3.md`. Also create `docs/assets/banner.svg` directly. Label each block in `## BLOCK: NAME` format.

---

## Agent 4 — Synthesis

**Run after Agents 1, 2, and 3 all complete.** You are a senior technical editor. Your job is to assemble the final README.md from the three agent outputs. You will do light copy editing to ensure the document reads as a single coherent voice — but do not rewrite any block substantially. Trust the agents.

### Step 1: Verify all inputs exist

```bash
for i in 1 2 3; do
  [ -f "/tmp/t19-agent-$i.md" ] && echo "Agent $i: ✅ ($(wc -l < /tmp/t19-agent-$i.md) lines)" || echo "Agent $i: ❌ MISSING — stop and re-run the missing agent"
done

[ -f "/Users/hoangharry/mh_code/internships/Bravo/aethel-workspace/docs/assets/banner.svg" ] && echo "banner.svg: ✅" || echo "banner.svg: ❌ MISSING"
```

If any input is missing, stop. Do not proceed with partial inputs.

### Step 2: Read all agent outputs in full

```bash
cat /tmp/t19-agent-1.md
cat /tmp/t19-agent-2.md
cat /tmp/t19-agent-3.md
```

### Step 3: Assemble the README

Write `README.md` at the repo root. The document must follow this exact structure, assembled from the labeled blocks in agent outputs:

```
1.  <div align="center"> [banner SVG img] </div>
2.  <div align="center"> [BLOCK: TAGLINE from Agent 2] </div>
3.  <div align="center"> [BLOCK: badge row from Agent 3] </div>
4.  <div align="center"> [BLOCK: stats bar from Agent 3] </div>
5.  [BLOCK: placeholder — demo GIF from Agent 3]
6.  ---
7.  [BLOCK: OPENING prose from Agent 2]
8.  > [!IMPORTANT] — self-hosted callout (write this yourself: 2 sentences, factual tone)
9.  ---
10. ## Three Pillars  [HTML table using BLOCK: THREE PILLARS TABLE CONTENT from Agent 2]
11. ---
12. ## Architecture  [BLOCK: MERMAID ARCHITECTURE DIAGRAM from Agent 2]
13. ---
14. ## Technical Highlights
    [> [!NOTE]] [BLOCK: runtime config callout from Agent 2]
    [> [!TIP]] [BLOCK: cryptographic chains callout from Agent 2]
    [> [!IMPORTANT]] [BLOCK: Argon2id callout from Agent 2]
15. ---
16. ## Quick Start
    [BLOCK: QUICK START COMMENTARY from Agent 2]
    [bash code block with 4 commands]
    [service URL table]
    [BLOCK: placeholder — screenshots from Agent 3]
17. ---
18. ## Tech Stack  [BLOCK: tech stack table from Agent 3]
19. ---
20. ## Feature Reference
    [BLOCK: FEATURE DEEP DIVES from Agent 2 — three <details> sections]
21. ---
22. ## Configuration
    <details> env vars table </details>
    <details> blueprint files table </details>
23. ---
24. ## Command Reference
    <details> full make target list </details>
25. ---
26. ## Project Structure
    <details> annotated directory tree </details>
27. ---
28. ## Documentation  [table linking all docs in docs/]
29. ---
30. ## Security  [BLOCK: SECURITY SUMMARY from Agent 2]
31. ## Contributing  [BLOCK: CONTRIBUTING NOTE from Agent 2]
32. ## License  [one sentence + Apache 2.0 link]
33. ---
34. <div align="center"> <sub> footer links </sub> </div>
```

For sections 22–26, write the content yourself using the preflight data — these are factual sections (tables, code, trees) that do not require creative judgment. Use the existing README.md as your source of facts for env vars, commands, and project structure.

**Critical rules for assembly:**
- Do not add any section that is not in the structure above
- Do not remove or reorder sections
- Do not alter the prose blocks from Agent 2 except to fix obvious grammatical errors
- Do not alter the visual elements from Agent 3 except to fix obviously broken HTML
- The document must render correctly on GitHub — test every HTML block for proper tag closure

### Step 4: Self-review before writing

Before writing the file, answer these questions to yourself:
- Does the tagline work without the banner for context? (It must — some readers see plain text)
- Is there any claim in the opening that is not verifiable from the codebase?
- Does the Mermaid diagram close all subgraph blocks?
- Do all `<details>` tags have matching `</details>`?
- Is every badge URL pointing to a valid shields.io endpoint?

If you find issues, fix them before writing.

### Step 5: Write the file

Write the complete, final README.md to `/Users/hoangharry/mh_code/internships/Bravo/aethel-workspace/README.md`.

### Output

Write to `/tmp/t19-agent-4.md`:

```markdown
## Agent 4 — Synthesis
_Assembled at: [timestamp]_

### Inputs used
- Agent 1 (Brand): ✅/❌
- Agent 2 (Content): ✅/❌
- Agent 3 (Visual): ✅/❌

### Sections assembled: N/34
### Lines in final README.md: N
### Known gaps (user must fill): [list placeholder locations]
```

---

## Agent 5 — QA Reviewer + Commit

**Run after Agent 4 completes.** You are a meticulous technical editor. Read the assembled README.md and apply a structured quality pass before committing.

### Step 1: Read the assembled README

```bash
cat /Users/hoangharry/mh_code/internships/Bravo/aethel-workspace/README.md
wc -l /Users/hoangharry/mh_code/internships/Bravo/aethel-workspace/README.md
```

### Step 2: Apply the QA checklist

Run each check. Note failures. Fix them before committing.

**Structural checks:**
```bash
cd /Users/hoangharry/mh_code/internships/Bravo/aethel-workspace

# Every <details> must have a matching </details>
echo "=== details open ===" && grep -c "<details" README.md
echo "=== details close ===" && grep -c "</details>" README.md

# Every div must close
echo "=== div open ===" && grep -c "<div" README.md
echo "=== div close ===" && grep -c "</div>" README.md

# SVG banner is referenced correctly
grep "docs/assets/banner.svg" README.md && echo "Banner: ✅" || echo "Banner: ❌ MISSING"

# Mermaid block is present
grep -c '```mermaid' README.md && echo "Mermaid: ✅" || echo "Mermaid: ❌"

# No external image URLs (only relative paths allowed)
grep -oP 'src="https?://[^"]*"' README.md | grep -v "shields.io" | head -5 && echo "WARNING: non-shield external images" || echo "External images: ✅"
```

**Content checks (manual):**
- Does the README contain any of these forbidden words? (`grep -in "seamless\|powerful\|robust\|scalable\|enterprise\|world-class\|cutting-edge\|next-generation\|revolutionize\|game-changing" README.md`)
- Is the tagline ≤ 20 words?
- Does the architecture diagram close all subgraph blocks?
- Do the Quick Start commands match the actual Makefile targets?

**Visual checks:**
```bash
# Banner SVG is valid
head -3 docs/assets/banner.svg | grep -c "<svg" && echo "SVG valid: ✅" || echo "SVG invalid: ❌"
tail -3 docs/assets/banner.svg | grep -c "</svg>" && echo "SVG closed: ✅" || echo "SVG unclosed: ❌"
```

### Step 3: Fix any failures

Fix directly in `README.md` and `docs/assets/banner.svg`. Do not make content changes — only fix broken structure.

### Step 4: Commit

```bash
cd /Users/hoangharry/mh_code/internships/Bravo/aethel-workspace
git add README.md docs/assets/
git commit -m "$(cat <<'EOF'
docs(readme): redesign as premium landing-page — Mercedes-style black/white

- SVG wordmark banner: black bg, ultralight sans-serif, corner accent marks
- Hero: badge row (Go 1.26, Nuxt 4, PostgreSQL 16, OpenAPI 3.1, Docker, Apache 2.0)
- Stats bar: 17 pages · 58 endpoints · 42 migrations · 3 domain pillars
- Three pillars: technical-depth HTML table (not marketing copy)
- Architecture: Mermaid flowchart showing full system topology
- Technical highlights: 3 GitHub callout boxes (runtime config, hash chain, Argon2id)
- Feature reference: 3 collapsible deep-dive sections (one per pillar)
- Placeholders for demo GIF and screenshots (user-provided)

Co-Authored-By: Claude Code Task 19 <noreply@anthropic.com>
EOF
)"
echo "Committed: $(git rev-parse --short HEAD)"
```

---

## Definition of Done

- [ ] `docs/assets/banner.svg` exists and is valid SVG (contains `<svg` and `</svg>`)
- [ ] `README.md` references `docs/assets/banner.svg` in the hero `<img>`
- [ ] `README.md` contains a Mermaid architecture diagram
- [ ] Badge row contains ≥ 6 shields.io badges
- [ ] Stats bar HTML table has 6 cells with real numbers
- [ ] Three pillars section uses an HTML table
- [ ] Three `<details>` collapsible sections for feature deep-dives exist
- [ ] No forbidden marketing words in any prose section
- [ ] All `<details>` tags matched (open count = close count)
- [ ] All `<div>` tags matched (open count = close count)
- [ ] Banner SVG uses corner accent marks (the luxury framing detail)
- [ ] Two placeholder comment blocks exist (demo GIF + screenshots)
- [ ] Quick Start section has ≤ 4 commands and a service URL table
- [ ] Changes committed to `dev` branch

---

## What the User Must Do After Running This Task

These assets require a running app and cannot be automated:

| Asset | Tool (free) | Where to put it |
|---|---|---|
| Demo GIF (15–30s: login → dispatch → green note → audit verify) | **Kap** (Mac, getkap.co) · **ScreenToGif** (Windows) | `docs/assets/demo.gif` |
| Screenshots (dashboard, doc detail, audit log) | Browser screenshot · **Carbon** (carbon.now.sh) | `docs/assets/screenshot-*.png` |
| Custom logo mark (optional — current text wordmark is intentional) | **Figma** (already installed) | Replace `docs/assets/banner.svg` |
| Live CI badges (once repo is public on GitHub) | shields.io dynamic badge URLs | Replace static badges in hero |
| Live demo URL (once staging is deployed) | Any host | Add badge to hero section |

After adding assets, uncomment or replace the `<!-- INSERT ... -->` placeholder blocks in the README.
