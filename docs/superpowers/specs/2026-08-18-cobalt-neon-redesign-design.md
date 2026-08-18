# Cobalt Neon — Design System Recolor + Interface Polish

## Context

The Dog monitoring platform frontend (`apps/web`) is mid-redesign. Uncommitted
work converted the theme from "Clean professional / Neon Cyber (cyan)" to
"Emerald Slate" (green primary `oklch(0.8871 0.2122 128.5041)`, brand hue 128).
Meanwhile `globals.css` declares `theme: Cobalt`, and the TypeScript theme
(`default.ts`) still carries the legacy indigo `#4F46E5` — the two view models
of the theme are out of sync.

**Decision:** rebase the design on a **Cobalt Blue** brand (matching the Cobalt
intent), keep the current structure (workbench shell, cards, dialogs), and run
a systematic interface-polish pass using the `make-interfaces-feel-better`
skill. Dark mode adopts a **neon** character (deep navy + electric cobalt glow).

## Design Principles

1. One brand hue drives primary, ring, glow, and chart-1.
2. Light theme: crisp cool near-white canvas, medium-saturated cobalt primary,
   dark navy foreground.
3. Dark theme: deep navy surfaces, luminous cobalt accents that read as a live
   signal; glow restrained so surfaces stay readable.
4. Polish pass expresses changes in the existing token system (Tailwind v4
   `@theme inline` + CSS variables). No second styling system.
5. TS theme view (`default.ts` / `brand.ts`) stays in sync with the CSS theme.

## 1. Brand Palette

| Token | Light | Dark |
|---|---|---|
| `--brand-accent` | `oklch(0.5856 0.2070 262.0)` | `oklch(0.7600 0.1300 250.0)` |
| `--brand-accent-foreground` | `oklch(1 0 0)` | `oklch(0.15 0.04 265)` (deep navy) |
| `--brand-hue` (brands) | `258` | — |

- `--ring`, `--primary`, `--chart-1`, `--shadow-glow`, and the focus halo all
  derive from the brand accent.
- Dark glow (`--shadow-glow`): `0 0 0 1px brand/0.28, 0 0 24px brand/0.16,
  0 10px 44px brand/0.12`.

## 2. Light Theme — "Cobalt Slate"

Recolor the emerald-slate surfaces to a cool blue tint (currently green hue 117
→ blue hue ~255):

- `--background: oklch(0.9856 0.004 255.0)` (was green-tinted near-white)
- `--surface`, `--muted`, `--border`, `--input`: cool slate-blue tint
- `--foreground`, `--secondary`, `--card`: unchanged navy/white (already correct)
- `--accent`: very light cobalt tint; `--accent-foreground`: cobalt
- `--success` / `--warning` / `--destructive` / `--info`: keep current values
- Chart ramp: cobalt, navy, green, amber, slate (chart-1 = brand accent)
- Shadow scale: keep current soft scale; `--shadow-panel: var(--shadow-sm)`

## 3. Dark Theme — "Neon Cobalt"

Deep navy canvas (unchanged) with luminous cobalt signal:

- `--brand-accent: oklch(0.7600 0.1300 250.0)`, foreground deep navy
- `--background` / `--card` / `--surface` / `--secondary` / `--muted`:
  unchanged deep navy values
- `--accent`: blue-tinted; `--accent-foreground`: luminous cobalt
- `--ring`, `--sidebar-ring`: luminous cobalt
- `--shadow-glow`: cobalt (see above)
- `base.css` dark body gradients: replace emerald `#155E4A` underglow and cyan
  `#0EA5E9` with cobalt/navy undertones
- Chart ramp: luminous cobalt, blue, green, violet, amber

## 4. TS Theme Sync

- `default.ts` `colors.primary`: `#4F46E5` → cobalt (≈ `#2B5CFF`); background →
  cool white `#F7F8FB`; surface → `#EEF1F7`; border → `#E1E5EF`; keep semantic
  status colors.
- `brand.ts` structure unchanged; `brandThemes` registry stays empty (default
  brand).

## 5. Interface Polish Pass (`make-interfaces-feel-better`)

Audit and fix across console + marketing. Categories and expected findings:

### Typography
- `text-wrap: balance` on headings (page headers, card titles, marketing
  headings); `text-wrap: pretty` where body copy needs it.
- `font-variant-numeric: tabular-nums` on every dynamically updating number:
  KPI cards, uptime %, latency values, probe stats, monitor tables.
- Font smoothing already applied in `base.css` — verify.

### Surfaces
- Concentric border radius: nested surfaces compute `outer = inner + padding`.
- Elevation via layered transparent shadows; borders reserved for structure and
  state.

### Animations
- Replace any `transition: all` with exact property lists.
- Subtle, fixed-`translateY` exit animations; `ease-out` both ways.
- `initial={false}` on `AnimatePresence` where present, unless an intentional
  entrance depends on it.
- Scale-on-press `0.96` already on buttons — extend to any missed interactive
  control, add a `static` escape where motion distracts.
- No custom animation on high-frequency interactions; static cue always present.

### Icons
- One stroke weight per surface; `2px` beside semibold text, `1.5px` beside
  regular. One icon set per surface; states via `currentColor` + opacity.

### Performance & Hit Areas
- `will-change` only on `transform`/`opacity`/`filter`, only where first-frame
  stutter is observed.
- Interactive controls ≥ 40px hit area in dense desktop UI; extend via
  pseudo-element when the visible element is smaller.

## 6. Affected Files

- `apps/web/design-system/brands/default.css`
- `apps/web/design-system/themes/light.css`
- `apps/web/design-system/themes/dark.css`
- `apps/web/design-system/themes/default.ts`
- `apps/web/design-system/base.css`
- `apps/web/app/globals.css` (theme comment sync)
- Plus any component/page found by the polish audit (shared/ui, design-system
  components, entities, widgets, marketing pages).

## 7. Validation

- `apps/web`: typecheck + production build (`npm run build`).
- Manual pass over light + dark in the console and marketing surfaces.
- Confirm focus rings, glow, and contrast hold in both themes.

## Out of Scope

- Layout / information-architecture changes (workbench shell composition stays).
- New animation systems or libraries.
- Brand logo redesign.
