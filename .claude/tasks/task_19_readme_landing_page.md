# Task 19 — README Landing Page (v2 — Improvements Pass)

**Purpose:** Improve the existing README with targeted additions: richer inline visualizations that help audiences understand the system without reading prose, plus three alternative layout variants so the author can compare and choose a direction.

**Baseline:** The current `README.md` (committed `d9808e6`) is the starting point — do not redesign it. Improve it. The overall structure, SVG banner, Mercedes-style aesthetic, and badge row are locked. Work within them.

**Primary outputs:**
- `README.md` — improved in-place (visualizations added, prose refined)
- `docs/assets/banner.svg` — keep as-is unless Agent 3 finds a clear improvement
- `README-option-a.md` — layout variant A (current design, polished)
- `README-option-b.md` — layout variant B (technical-first, developer audience)
- `README-option-c.md` — layout variant C (visual-first, product/admin audience)

**Design direction:** Restrained luxury — Mercedes-Benz, not SaaS startup. Black and white. Geometry over decoration. Visualizations must earn their place: each one should convey something that prose or a table cannot. The three pillars, the middleware stack, the dispatch lifecycle, and the hash-chain mechanism are natural candidates. Do not add visualizations just to add them.

**Visualization philosophy (agents must internalize this before designing):**
- A good README visualization follows three design rules from Edward Tufte: maximize data-ink ratio, remove chartjunk, show the data. If a diagram does not reveal structure or relationships that text cannot, cut it.
- GitHub natively renders Mermaid diagrams (`\`\`\`mermaid` code fences) — use them for flow, state machines, and sequence diagrams. No external image hosting needed.
- SVG files committed to the repo render inline with `<img src="...svg">` — use for static structural diagrams that Mermaid cannot express elegantly (e.g., a hash chain visualization, a role permission matrix).
- Tables in GitHub markdown render as visual grids — a well-designed table IS a visualization. Use them for the role-permission matrix and feature comparison.
- Do NOT add pie charts, bar charts, or generic "stats graphics" — they add visual noise without information density.

---

## Execution Flow

```
[Step 0]  Pre-flight → /tmp/t19-preflight.md
               ↓
[Parallel] Agent 1 — Visualization Designer  → /tmp/t19-agent-1.md + SVG files in docs/assets/
           Agent 2 — Content Refiner         → /tmp/t19-agent-2.md (prose improvements + diagram upgrades)
           Agent 3 — Layout Variant Designer → README-option-b.md + README-option-c.md
               ↓ all complete
[Serial]   Agent 4 — Synthesis               → README.md (improved) + README-option-a.md
               ↓
[Serial]   Agent 5 — QA Reviewer             → quality pass + commit + push
```

Agents 1, 2, and 3 run fully in parallel. Each writes only to its assigned files. Agent 4 reads all three and assembles the improvements into the main README, and also produces option-a (the polished version of the current design).

---

## Step 0 — Pre-flight (run in orchestrating session, not a subagent)

```bash
REPO=/Users/hoangharry/mh_code/internships/Bravo/aethel-workspace
cd "$REPO"

echo "## Current README (baseline to improve)" > /tmp/t19-preflight.md
cat README.md >> /tmp/t19-preflight.md

echo -e "\n## CLAUDE.md key sections" >> /tmp/t19-preflight.md
head -150 CLAUDE.md >> /tmp/t19-preflight.md

echo -e "\n## Dispatch status enum values" >> /tmp/t19-preflight.md
grep -n "dispatch_status\|PENDING\|UNDER_REVIEW\|IN_TRANSIT\|DELIVERED\|ESCALATED\|DISPATCHED\|REJECTED" \
  aethel-core/internal/domain/dispatch.go 2>/dev/null | head -20 >> /tmp/t19-preflight.md

echo -e "\n## RBAC roles and permissions" >> /tmp/t19-preflight.md
grep -n "role\|permission\|ADMIN\|RECEPTION\|USER\|SYS_ADMIN" \
  aethel-core/internal/rbac/middleware.go 2>/dev/null | head -40 >> /tmp/t19-preflight.md

echo -e "\n## Middleware stack order in server.go" >> /tmp/t19-preflight.md
grep -n "r\.Use\|Use(" aethel-core/internal/api/server.go | head -20 >> /tmp/t19-preflight.md

echo -e "\n## Green note hash chain fields" >> /tmp/t19-preflight.md
grep -n "hash\|chain\|previous\|sequence\|SHA" \
  aethel-core/internal/domain/governance.go 2>/dev/null | head -20 >> /tmp/t19-preflight.md

echo -e "\n## Existing docs/assets files" >> /tmp/t19-preflight.md
find docs/assets -type f 2>/dev/null | sort >> /tmp/t19-preflight.md

echo -e "\n## Git HEAD" >> /tmp/t19-preflight.md
git rev-parse --short HEAD >> /tmp/t19-preflight.md

echo "Pre-flight complete. $(wc -l < /tmp/t19-preflight.md) lines written."
```

---

## Agent 1 — Visualization Designer

**You are a senior information designer** with expertise in developer documentation, data visualization, and GitHub README aesthetics. You've studied Edward Tufte's principles (maximize data-ink ratio, eliminate chartjunk, show the data), and you understand that a visualization in a README earns its place only if it reveals structure that prose and tables cannot.

**Your job:** Design and create SVG visualizations that augment the existing README. Do NOT change the existing content — add to it. You will produce 3 targeted SVG diagrams that address the three most visually underserved concepts in the current README.

**Working directory:** `docs/assets/`

### Step 1: Read the baseline and understand what already exists

```bash
cat /tmp/t19-preflight.md
cat /Users/hoangharry/mh_code/internships/Bravo/aethel-workspace/README.md
ls /Users/hoangharry/mh_code/internships/Bravo/aethel-workspace/docs/assets/
```

The existing README already has: a Mermaid architecture diagram, a three-pillar HTML table, a tech stack badge table, collapsible feature deep-dives. Do not duplicate any of these.

### Step 2: Identify the three visualization opportunities

Apply this decision framework to each candidate concept:
> "If a reader sees only this SVG — with no surrounding text — do they learn something meaningful about the system? Does the diagram reveal a relationship, sequence, or structure that a table or paragraph would require 5x the words to convey?"

The three concepts that pass this test for Aethel:

**Visualization 1 — Dispatch Lifecycle State Machine** (`docs/assets/dispatch-lifecycle.svg`)
The dispatch status transitions form a directed graph with clear fork points (routing, escalation, rejection). This is better as a state machine diagram than as a prose list. Create a horizontal SVG state machine:
- Nodes: PENDING_ASSIGNMENT → UNDER_REVIEW → IN_TRANSIT → DELIVERED (happy path, left to right)
- Branch: UNDER_REVIEW → ESCALATED (triggered by worker, upward branch)
- Branch: UNDER_REVIEW → REJECTED (downward branch)
- Branch: IN_TRANSIT → ATTEMPTED_DELIVERY (loop back node)
- Visual style: rounded rectangle nodes with status label inside; arrow labels for transitions (e.g., "routing rule fires", "threshold exceeded", "admin action"); color-code nodes to match CLAUDE.md status colors: PENDING=slate, UNDER_REVIEW=indigo, IN_TRANSIT=sky, DELIVERED=emerald, ESCALATED=rose, REJECTED=red, ATTEMPTED=amber
- Dimensions: `width="900" height="300"` — fits inline in GitHub on any screen
- Background: white (`#ffffff`) — this diagram sits inline in the README body, not in the dark hero section
- Font: same system stack as banner, but weight 400, size 13px
- Arrows: use `<marker>` + `<path>` SVG primitives for proper arrowheads

**Visualization 2 — Green Note Hash Chain** (`docs/assets/hash-chain.svg`)
The cryptographic chain concept (each note hashes the previous) is the most novel technical feature of the project and the hardest to grasp from prose alone. Create a horizontal chain diagram showing 4 notes:
- Each note is a card: top shows `Note #N` label, middle shows `content` snippet, bottom shows a shortened hash value (e.g., `a3f9...b72c`)
- Connecting arrows: an arrow from each note's hash cell to the next note's "prev_hash" input — visually showing the chain dependency
- Color scheme: monochrome — dark slate cards (`#1e293b`), white text, `#4f46e5` (indigo) for the hash values and connection arrows to draw the eye to the chain mechanism
- Add a subtle "TAMPER DETECTED" annotation on note #3 showing a broken chain (red arrow, red hash text) to immediately communicate what chain violation looks like
- Dimensions: `width="900" height="200"`

**Visualization 3 — Role Permission Matrix** (`docs/assets/role-matrix.svg`)
A compact matrix showing which actions each role can perform. This is far more scannable than a prose list. Rows = actions (View documents, Create dispatch, Approve green notes, Manage routing rules, View audit log, Verify audit chain, Manage users, Change branding). Columns = roles (USER, RECEPTION, ADMIN, SYS_ADMIN). Cells: ✓ (emerald filled circle), – (empty circle), or a lock icon for SYS_ADMIN-only rows.
- Visual style: clean grid, alternating row backgrounds (`#f8fafc` / `#ffffff`), header row with dark background (`#1e293b`) and white text
- Dimensions: `width="700" height="320"`

### Step 3: Create the three SVG files

Create each SVG file at its path in `docs/assets/`. Requirements for all three:
- All SVG elements are self-contained: no `<use>` referencing external files, no `<image>` src, no external fonts
- All text uses the system font stack: `font-family="-apple-system, 'Helvetica Neue', Helvetica, Arial, sans-serif"`
- All SVGs are valid: open tag with `xmlns="http://www.w3.org/2000/svg"`, closed `</svg>`
- No `style=` attributes that get stripped — use SVG presentation attributes (`fill`, `stroke`, `font-size`, etc.) directly on elements

### Step 4: Write placement instructions

For each SVG, write the exact markdown snippet (centered `<img>` tag) AND the exact location in the README where it should be inserted (by referencing the surrounding text from the current README). This makes Agent 4's job unambiguous.

Example format:
```
PLACEMENT — dispatch-lifecycle.svg:
Insert AFTER the line "```" that closes the Mermaid architecture diagram and BEFORE the "---" separator.
Markdown: <div align="center"><img src="docs/assets/dispatch-lifecycle.svg" width="100%" alt="Dispatch lifecycle state machine"/></div>
```

### Output

Write all placement instructions to `/tmp/t19-agent-1.md`. Create the three SVG files directly in `docs/assets/`. Do not modify README.md — that is Agent 4's job.

---

## Agent 2 — Content Refiner

**You are a senior technical editor** who improves existing documentation without rewriting it. Your instinct is surgical: one precise change beats three good changes.

**Your job:** Read the current README and identify exactly 5–8 targeted improvements to existing content. Additions and refinements only — do not delete any section or restructure the layout.

### Step 1: Read the current README and context

```bash
cat /tmp/t19-preflight.md
cat /Users/hoangharry/mh_code/internships/Bravo/aethel-workspace/README.md
cat /Users/hoangharry/mh_code/internships/Bravo/aethel-workspace/docs/architecture/architecture-security.md | head -80
```

### Step 2: Apply this evaluation rubric to the existing README

For each section of the README, answer:
1. **Information gap:** Is there a fact about this project that a reader would want to know here, that isn't here?
2. **Credibility gap:** Is there a claim that would be more credible with a specific number, filename, or technical detail added?
3. **Clarity gap:** Is there a sentence where a reader would have to re-read to understand what it means?

Prioritize findings in this order: credibility gaps first (they directly affect whether an IT admin trusts the product), information gaps second, clarity gaps third. Stop at 8 findings.

### Step 3: Write improvement patches

For each finding, write a patch in this exact format:

```
PATCH #N — [section name]
TYPE: addition / refinement / clarification
FIND THIS TEXT: "[exact string from current README — enough context to locate it uniquely]"
CHANGE TO: "[new text — include surrounding unchanged words for context]"
RATIONALE: [one sentence: what does this improve and why does it matter to the reader]
```

**Boundaries:** Do not propose changes that:
- Restructure section order
- Replace the existing Mermaid diagram (it already exists and works)
- Change the banner or badge row
- Add new top-level sections (that would change the layout)
- Touch the collapsible sections' structure (only their text content)

**Ideas to evaluate** (not prescriptive — find what actually applies after reading):
- Does the Quick Start section mention what happens on first login? (bootstrap-admin command)
- Does the architecture diagram prose explain the config cache's role in SSR?
- Do the feature deep-dives show a real JSON example of an API response?
- Does the audit log verification section show what the broken-chain response looks like?
- Does the tech stack section note which Go version and why (Go 1.26 is not yet released — verify)?
- Are there any numbers in the prose that could be more specific (e.g., "many migrations" vs "42 migrations")?
- Does the security section mention rate limiting and account lockout specifically?

### Output

Write all patches (exactly 5–8, no more) to `/tmp/t19-agent-2.md` in the patch format above.

---

## Agent 3 — Layout Variant Designer

**You are a creative front-end designer** with experience writing GitHub READMEs for open source projects across different audience types. You know that layout is a design decision — what you put first, how dense the information is, and how much prose vs. visual content shapes who the document speaks to.

**Your job:** Create two alternative README layouts as separate files. These are NOT replacements for the current README — they are options for the author to explore and choose from. The author will compare all three (current + your two) and pick a direction.

**Working directory:** repo root

### Step 1: Read the current README and understand its defaults

```bash
cat /tmp/t19-preflight.md
cat /Users/hoangharry/mh_code/internships/Bravo/aethel-workspace/README.md
```

The current README (`README-option-a.md` — Agent 4 will save this copy) is a balanced design: equal weight on marketing narrative and technical depth, collapsible sections hiding the detail. Your two variants should offer meaningfully different trade-offs.

### Step 2: Create `README-option-b.md` — Technical-First (Developer Audience)

**Design philosophy:** "Show me the architecture, then convince me." This variant is for the developer who opens a README by immediately scrolling to the technical section. It front-loads the architecture diagram and code, and moves marketing prose to the bottom (or removes it). Dense, confident, respects the reader's time.

Key design differences from the current README:
- Lead with the architecture Mermaid diagram immediately after the hero + badges — before any prose
- Replace the "opening" narrative paragraphs with a single-paragraph technical summary (`> [!NOTE]` callout)
- Move the Three Pillars section AFTER the Architecture diagram, not before it
- Expand the Quick Start section to be the second major section (after Architecture)
- Add a "API at a Glance" section showing 5–6 key endpoints with their HTTP method, path, and one-line description in a code block or table — gives developers an instant sense of the API surface
- Make Feature Reference the first detailed section (not hidden in collapsibles — show one pillar expanded, two collapsed)
- Keep the tech stack table but remove the prose tech-stack section that just lists the same info
- Footer: trim to just license + contributing links

Use the same banner SVG (`docs/assets/banner.svg`) and same badges. Do NOT change those.

Write this to `README-option-b.md` at the repo root. It must be complete and renderable.

### Step 3: Create `README-option-c.md` — Visual-First (Product/Admin Audience)

**Design philosophy:** "Show me what I'm getting, then tell me how it works." This variant is for the IT administrator or non-developer evaluating Aethel. More visual, more use-case driven, lighter on implementation details in the main flow.

Key design differences from the current README:
- After the banner + badges, add a visual "Use Case Summary" using an HTML table with 3 rows: one per role (ADMIN, RECEPTION, USER), showing what each role does in the system — like a product feature matrix but narrative
- The "Three Pillars" section leads with a short use-case sentence ("A reception clerk receives an inbound letter from the Ministry of Finance...") before the technical description — grounding the abstract feature in a concrete workflow
- Add a "Deployment Footprint" callout box listing: 2 Docker containers, 2 YAML config files, 1 PostgreSQL database, 0 external services — makes the operational story tangible
- Move Argon2id / cryptographic highlights to the collapsible Feature Reference (they're credibility signals for developers, not the primary IT admin concern)
- The Quick Start section includes a "What You'll See" subsection: 3 bullet points describing what the UI looks like after `make dev` succeeds
- Add a "Designed for regulated environments" `[!IMPORTANT]` callout near the top: 3 bullet points referencing tamper-evident audit log, chain-of-custody tracking, RBAC with `sys_admin` gating — the compliance story
- Footer: add a "Deployment checklist" link pointing to the IT customization guide

Write this to `README-option-c.md` at the repo root. It must be complete and renderable.

### Step 4: Write comparison guide

At the top of your `/tmp/t19-agent-3.md` output, write a short comparison table:

```markdown
## Layout Comparison Guide

| Dimension | Option A (current) | Option B (technical) | Option C (product/admin) |
|---|---|---|---|
| Primary audience | Mixed | Developer | IT administrator |
| First impression | Elegant, balanced | Dense, technical | Approachable, visual |
| Time to architecture | ~3 scrolls | Immediate | ~5 scrolls |
| Marketing prose | Moderate | Minimal | Moderate |
| Best for | GitHub repo homepage | Hacker News, dev communities | Internal evaluation, procurement |
| Visual weight | Medium | Low | High |
```

Then add 2–3 sentences of honest opinion: which one would you choose for this project, and why?

### Output

Write comparison guide + opinion to `/tmp/t19-agent-3.md`. Create `README-option-b.md` and `README-option-c.md` directly at the repo root.

---

## Agent 4 — Synthesis

**Run after Agents 1, 2, and 3 all complete.** You are a senior technical editor. Apply the improvements from Agents 1 and 2 to the main README, and save the current version as option-a.

### Step 1: Verify all inputs

```bash
REPO=/Users/hoangharry/mh_code/internships/Bravo/aethel-workspace
cd "$REPO"

for i in 1 2 3; do
  [ -f "/tmp/t19-agent-$i.md" ] && echo "Agent $i: ✅" || echo "Agent $i: ❌ MISSING"
done

for f in docs/assets/dispatch-lifecycle.svg docs/assets/hash-chain.svg docs/assets/role-matrix.svg; do
  [ -f "$f" ] && echo "$f: ✅" || echo "$f: ❌ MISSING"
done

for f in README-option-b.md README-option-c.md; do
  [ -f "$f" ] && echo "$f: ✅" || echo "$f: ❌ MISSING"
done
```

Stop if any file is missing.

### Step 2: Save option-a (current README, unchanged)

```bash
cd /Users/hoangharry/mh_code/internships/Bravo/aethel-workspace
cp README.md README-option-a.md
```

### Step 3: Apply Agent 1's SVG placements to README.md

Read `/tmp/t19-agent-1.md` for the exact placement instructions. For each SVG, insert the `<img>` snippet at the specified location in `README.md`. Use `Edit` tool with the surrounding text as context to place precisely.

After inserting all three SVGs, verify the placements did not break surrounding structure:
```bash
grep -c "<div align=" README.md
grep -c "</div>" README.md
```
Counts must be equal.

### Step 4: Apply Agent 2's content patches to README.md

Read `/tmp/t19-agent-2.md`. For each PATCH, apply it to `README.md` using exact string replacement on the FIND THIS TEXT. Do not apply any patch that would conflict with a prior SVG insertion. If a conflict exists, note it but skip the conflicting patch.

### Step 5: Final self-review

```bash
cd /Users/hoangharry/mh_code/internships/Bravo/aethel-workspace

# Structural integrity
echo "details open/close:" && grep -c "<details" README.md && grep -c "</details>" README.md
echo "div open/close:" && grep -c "<div" README.md && grep -c "</div>" README.md

# No forbidden words
grep -in "seamless\|powerful\|robust\|enterprise\|world-class\|cutting-edge\|revolutionize" README.md | head -5 || echo "✅ No forbidden words"

# SVG references valid
grep "dispatch-lifecycle\|hash-chain\|role-matrix\|banner" README.md | head -10
```

Fix any structural issues found. Do not change content.

### Output

Write to `/tmp/t19-agent-4.md`:
```markdown
## Agent 4 — Synthesis
_Completed at: [timestamp]_

### SVG placements applied: N/3
### Content patches applied: N/N (list any skipped + reason)
### README.md integrity: ✅/❌
### README-option-a.md saved: ✅/❌
```

---

## Agent 5 — QA Reviewer + Commit + Push

**Run after Agent 4 completes.**

### Step 1: Full read of all output files

```bash
REPO=/Users/hoangharry/mh_code/internships/Bravo/aethel-workspace
wc -l "$REPO"/README.md "$REPO"/README-option-a.md "$REPO"/README-option-b.md "$REPO"/README-option-c.md
for f in dispatch-lifecycle.svg hash-chain.svg role-matrix.svg banner.svg; do
  head -1 "$REPO/docs/assets/$f" | grep -q "<svg" && echo "$f: valid SVG ✅" || echo "$f: ❌ invalid"
done
```

### Step 2: Verify all three SVG files render correctly

Each SVG must:
- Open with `<svg` and close with `</svg>`
- Contain no `<image src=` external references
- Contain no `<style>` blocks (use presentation attributes)
- Be ≤ 30KB (GitHub has display limits for large inline SVGs)

```bash
for f in dispatch-lifecycle.svg hash-chain.svg role-matrix.svg; do
  SIZE=$(wc -c < /Users/hoangharry/mh_code/internships/Bravo/aethel-workspace/docs/assets/$f)
  echo "$f: ${SIZE} bytes"
done
```

### Step 3: Verify option-b and option-c are complete

Each variant README must contain:
- The banner `<img>` reference
- At least one Mermaid code block
- At least one `> [!` callout block
- A Quick Start section with bash code block

```bash
for opt in b c; do
  f="/Users/hoangharry/mh_code/internships/Bravo/aethel-workspace/README-option-$opt.md"
  echo "=== option-$opt ==="
  grep -c "banner.svg" "$f" && echo "banner: ✅" || echo "banner: ❌"
  grep -c '```mermaid' "$f" && echo "mermaid: ✅" || echo "mermaid: ❌"
  grep -c '> \[!' "$f" && echo "callouts: ✅" || echo "callouts: ❌"
  grep -c '```bash' "$f" && echo "bash blocks: ✅" || echo "bash blocks: ❌"
done
```

### Step 4: Commit and push

```bash
cd /Users/hoangharry/mh_code/internships/Bravo/aethel-workspace
git add README.md README-option-a.md README-option-b.md README-option-c.md docs/assets/
git commit -m "$(cat <<'EOF'
docs(readme): add visualizations + content refinements + 3 layout variants

Visualizations (new SVG assets):
- docs/assets/dispatch-lifecycle.svg: state machine diagram with color-coded status nodes
- docs/assets/hash-chain.svg: cryptographic chain diagram with tamper-detection annotation
- docs/assets/role-matrix.svg: RBAC permission matrix (4 roles × 8 actions)

Content improvements:
- [N targeted patches from Agent 2 — update this list from /tmp/t19-agent-2.md]

Layout variants for author review:
- README-option-a.md: current design (baseline, unchanged)
- README-option-b.md: technical-first (developer audience, architecture leads)
- README-option-c.md: visual-first (IT admin/product audience, use-case driven)

Co-Authored-By: Claude Code Task 19 <noreply@anthropic.com>
EOF
)"
git push origin dev
echo "Pushed: $(git rev-parse --short HEAD)"
```

---

## Definition of Done

- [ ] `docs/assets/dispatch-lifecycle.svg` — valid SVG, dispatch state machine with colored nodes
- [ ] `docs/assets/hash-chain.svg` — valid SVG, 4-note chain with tamper annotation
- [ ] `docs/assets/role-matrix.svg` — valid SVG, 4-role × 8-action permission grid
- [ ] All three SVGs are ≤ 30KB
- [ ] All three SVGs inserted into `README.md` at correct locations
- [ ] 5–8 targeted content patches applied to `README.md`
- [ ] `README.md` passes structural integrity check (`<div>` and `<details>` open=close counts)
- [ ] No forbidden marketing words in `README.md`
- [ ] `README-option-a.md` — copy of current README (before improvements)
- [ ] `README-option-b.md` — technical-first variant, complete and renderable
- [ ] `README-option-c.md` — visual-first variant, complete and renderable
- [ ] All option files contain: banner, Mermaid diagram, callout boxes, Quick Start
- [ ] Agent 3's comparison table written to `/tmp/t19-agent-3.md`
- [ ] All changes committed and pushed to `dev` branch

---

## What the User Must Do After Running This Task

**Review the three options:**
```bash
# On GitHub — push and compare the three README files side by side
# Or locally with any Markdown preview tool (VS Code, Typora, Marked 2)
open README-option-a.md  # baseline
open README-option-b.md  # technical-first
open README-option-c.md  # visual-first
```

Read `/tmp/t19-agent-3.md` for the comparison table and Agent 3's honest recommendation.

**Pick one as the final README.md:**
```bash
cp README-option-[a/b/c].md README.md
git add README.md && git commit -m "docs(readme): adopt option-[a/b/c] as canonical"
git push origin dev
```

**Add user-provided assets** (requires running app — cannot be automated):

| Asset | Tool (free) | Where to put it |
|---|---|---|
| Demo GIF (15–30s: login → dispatch → audit verify) | **Kap** (Mac) · **ScreenToGif** (Windows) | `docs/assets/demo.gif` |
| Screenshots | Browser · **Carbon** (carbon.now.sh) | `docs/assets/screenshot-*.png` |
| Custom logo mark (optional) | **Figma** (already installed) | Replace `docs/assets/banner.svg` |
| Live CI badges (once repo is public) | shields.io dynamic URLs | Replace static version badges |
