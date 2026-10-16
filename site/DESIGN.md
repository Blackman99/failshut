# failshut design language

**Reference object:** soap-bubble leak test on a pressurized copper refrigerant manifold.
Bubbles at a joint = FAIL OPEN. Dry joint = FAIL SHUT.

## Palette
| Token | Hex | Use |
|---|---|---|
| concrete | `#d9dde3` | page ground (workshop floor) |
| ink | `#14171c` | body text, pipe outlines |
| copper | `#b87333` | pipe body |
| copper-dark | `#8a5524` | pipe shade / fittings |
| soap-cyan | `#3db8b0` | soap film |
| soap-magenta | `#c45a8c` | iridescent sheen on film only |
| seal-ok | `#1f7a6b` | sealed joint badge |
| fail-open | `#e85d04` | leaking joint alert (joints only) |

## Type
- **Sora** 600/800 — brass stamped tags / display
- **Work Sans** 400/600 — body
- **Source Code Pro** 400/600 — commands only

## Layout & motion
Full-bleed copper manifold hero. H1 is a brass tag on a pipe elbow: `FAILSHUT — LEAK CHECK`.
Findings are soap bubbles that nucleate only on fail-open joints. Sections hang from pipe hangers on a horizontal run.
Animation: pressure rise → soap wipe → bubbles on fail-open only. Respect `prefers-reduced-motion`.

## Logo
Copper pipe elbow + one iridescent bubble on the joint. Transparent background. Not a rounded app icon.

Registered in `/workspace/oss-factory/design/registry.md` on 2026-10-16 before site implementation.
