# Homepage Simulated-Arm Demo

An interactive, client-side 3D demo: a barista robot (Viam's **Beanjamin**) makes
espresso, and you can **swap the arm** and watch the **same motion code** drive a
different robot.

> **Same motion code. Different arm.**

Beanjamin runs on a uFactory **xArm6**. This demo plans Beanjamin's real espresso
brew sequence with genuine [Viam motion planning](https://docs.viam.com/services/motion/)
and lets you toggle between the **xArm6** and a **UR5e** — the planning code is
identical; only the arm's kinematics model changes.

## How it works

```
BUILD TIME (Go)                          RUNTIME (browser, static)
┌──────────────────────────┐            ┌────────────────────────────────┐
│ cmd/bake                 │  static    │ SvelteKit + @viamrobotics/      │
│  rdk armplanning plans   │  JSON      │ motion-tools                    │
│  the brew sequence for   │ ─────────► │  • StaticProvider loads asset   │
│  each arm → scene + a     │  assets   │  • TrajectoryPlayer replays it  │
│  per-step pose track     │            │  • Visualizer renders the scene │
└──────────────────────────┘            └────────────────────────────────┘
```

- **Go baker** (`cmd/bake`, `internal/{scene,brew,bake}`): builds each arm's frame
  system from the Beanjamin machine config, plans the brew sequence with
  `rdk/motionplan/armplanning`, and writes `web/static/trajectories/<arm>.brew.json`
  (a motion-tools scene snapshot + a per-step world-pose track). Planning happens
  once, offline; the result is a static asset, so the runtime scales to any number
  of viewers with no server.
- **Web app** (`web/`): a SvelteKit app that embeds the `Visualizer` from
  `@viamrobotics/motion-tools`, loads the baked asset, and replays the trajectory
  by feeding per-step poses into the scene. The arm toggle swaps which baked asset
  plays; the collapsible code drawer shows the real planning loop with only the
  arm-model name changing.

## Quick start

```bash
make bake     # plan both arms → static assets (uses Go + rdk)
make dev      # run the web app (http://localhost:5173)
```

For a production-equivalent run:

```bash
make build              # build web/build (static)
pnpm -C web preview     # serve the build
```

## Tests

```bash
make test       # Go: planning gate (both arms), scene, baker round-trip
make test-web   # web unit tests (StaticProvider)
make e2e        # Playwright: render → brew → toggle → code drawer
```

## Notes

- **Arms:** xArm6 (Beanjamin's real arm) and UR5e. Adding another arm is mostly a
  matter of adding its kinematics model + a validated ready start config
  (`internal/brew/ready.go`) and re-baking.
- **Collision model:** the brew motion is planned collision-aware against the
  interaction stations (coffee machine, grinder, tamper) and structure (table,
  ceiling, mount). A few peripheral, non-interaction obstacles (camera mast,
  speaker, stream-deck, stray cup) are excluded from collision because the generic
  rdk kinematics sweep into them where Beanjamin's actual arm clears them.
- **motion-tools:** consumed as the published `@viamrobotics/motion-tools` npm
  release (the demo doesn't depend on a local checkout).
- **Design + plan:** see `docs/plans/2026-06-16-*`.
