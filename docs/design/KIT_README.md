# EduPilot Impeccable UI Kit

A design handoff for rebuilding EduPilot v2 using the **Red Thread / Academic Instrument** direction.

## Files

- `DESIGN.md` — authoritative UI/UX system and route-by-route contracts.
- `AGENT_PROMPT.md` — ready-to-paste prompt for a coding agent.
- `DESIGN_TOKENS.css` — semantic starter tokens.
- `edupilot-ui-v3.html` — interactive concept prototype. Open locally in a browser; switch roles from the top-right selector.
- `logo-edupilot.svg` — primary wordmark lockup.
- `logo-edupilot-mark.svg` — standalone mark.
- `favicon.svg` — mark-sized asset.

## Recommended placement in the real repo

```text
repo/
├── PRD.md
├── ARCHITECTURE.md
├── DESIGN.md                  <- copy from this kit
├── AGENT_PROMPT.md            <- copy from this kit
├── docs/design/
│   └── edupilot-ui-v3.html    <- optional visual reference
└── frontend/
    └── ...
```

## Agent kickoff

Tell the coding agent:

> Read `PRD.md`, `ARCHITECTURE.md`, `DESIGN.md`, and `AGENT_PROMPT.md` before touching frontend code. Preserve product behavior and data contracts. Rebuild the visual system from tokens/primitives first, then migrate routes in the order specified by `AGENT_PROMPT.md`. Do not invent features.

## Design principle

The interface is minimal by **reducing obstacles**, not by removing real product capabilities. Technical depth remains accessible through progressive disclosure, while each first viewport centers one role-appropriate task.
