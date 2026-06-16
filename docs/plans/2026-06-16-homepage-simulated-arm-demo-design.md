# Homepage Simulated-Arm Demo — Design

**Date:** 2026-06-16
**Status:** Design validated, ready for implementation planning
**Repo:** `~/viam/homepage-simulated-arm-demo` (standalone; consumes motion-tools as a dependency)

## Goal

An interactive, embeddable 3D demo — eventually for the Viam homepage — that shows a
barista robot (Beanjamin) making espresso, and lets a viewer **swap the arm** and watch
the **same motion code** drive a different arm. The message: *"Beanjamin runs on an
xArm. Here's the exact same code on a UR5e."*

## Narrative / value prop

Beanjamin (Viam's real barista) runs on a uFactory **xArm**. The demo proves Viam's
portability claim by running the **identical** motion-planning code against a second arm
(**UR5e**) and showing the result is the same task, executed by a structurally different
robot. The "same code" is made undeniable via a collapsible code drawer where only the
arm-model argument changes when you toggle.

## Decisions log (what we chose and why)

| Decision | Choice | Rationale |
| --- | --- | --- |
| Runtime | **Client-side**, no per-viewer server | Must scale to thousands of concurrent homepage viewers. |
| Motion source | **Real Viam motion planning** (`armplanning.PlanMotion`) | Authentic to the value prop — not a scripted animation. |
| When planning runs | **Offline (build-time) bake → static replay** for v1; WASM live-planning as a designed-for-later v2 | Fixed task ⇒ only 2 possible outputs, so baked looks identical to live and carries zero build/runtime risk. WASM only earns its keep if the *problem* can change at runtime (drag goal/obstacle). |
| Arm visual fidelity | **Hybrid**: GLTF mesh where available, else schematic capsule/box geometry from the kinematics model | Ship immediately on kinematics geometry; drop in meshes later with no rework. |
| Arm lineup | **xArm6 ↔ UR5e** (two-arm toggle) | Punchy A/B; xArm6 is Beanjamin's real arm, UR5e is the contrast. |
| Code visibility | **Collapsible code drawer** | Clean homepage look by default; expandable proof for the curious. |
| Task & scene | Beanjamin's real **brew sequence** + **real obstacle scene** from its config | Authentic geometry and poses; collision-aware planning. |
| Pose/scene data source | **Beanjamin machine config export** (provided) | Real poses + frame system without live machine access. |
| Code location | **Standalone repo** `~/viam/homepage-simulated-arm-demo` | Per user requirement; motion-tools consumed as a dependency. |

## Architecture

Two layers split by a single interface, so v1 (baked) and v2 (WASM) differ in exactly
one place.

### Build time (native Go — runs once, in CI or by hand)

`cmd/bake/` lifts Beanjamin's planning loop (`beanjamin/motion.go` + `espresso.go`):

```
for arm in [xarm6, ur5e]:
  fs := frameSystem(armKinematics[arm], sceneObstaclesFromConfig)   # arm at world origin
  for step in brewSequence:
    plan = armplanning.PlanMotion(fs, goal[step], start, constraints)   # REAL Viam planning
    for inputs in plan.Trajectory():
      poses := fs.Transform(inputs)        # forward kinematics in Go → per-link world poses
      track.append({tMs, poses})
  write static/<arm>.brew.json   # scene snapshot + pose track, via draw.Snapshot.MarshalJSON
```

- Imports: `go.viam.com/rdk/motionplan/armplanning`, `go.viam.com/rdk/referenceframe`,
  `go.viam.com/rdk/spatialmath`, and `github.com/viam-labs/motion-tools/draw` (snapshot
  builders, used **in-memory** — no RPC/viz server).
- Reads the committed, **secret-stripped** Beanjamin config (`data/beanjamin-config.merged.json`)
  plus each arm's rdk kinematics file (`components/arm/fake/kinematics/{xarm6,ur5e}.json`).

### Runtime (browser — pure client-side)

```
TrajectoryProvider (interface)
 ├─ StaticProvider  → fetch('<arm>.brew.json')          ← v1
 └─ WasmProvider    → planner.wasm in a Web Worker       ← v2 drop-in (GOOS=js GOARCH=wasm -tags no_cgo)
        │ returns { sceneSnapshot, track: {tMs, poses}[] }
        ▼
<TrajectoryPlayer>  — mounts scene once, interpolates poses (lerp position / slerp
                      orientation) between waypoints, writes link poses into the ECS,
                      calls invalidate() (renderer is on-demand, not a continuous loop)
        ▼
motion-tools renderer (embedded Visualizer) — hybrid mesh/geometry, static scene
UI overlay — [ xArm6 ‖ UR5e ] toggle · "Make coffee" · collapsible code drawer
```

**FK is never done in the browser.** Forward kinematics runs in Go (native at build, or
WASM later) via the frame system's `Transform`, exactly as Beanjamin's
`viz.DrawFrameSystem(fs, fsInputs)` does. The asset stores **posed link transforms**, not
joint angles, so both providers are symmetric and the renderer never sees a joint angle.

## Asset format

`static/<arm>.brew.json`:

```
{
  "scene":  <draw.Snapshot>,         // static once: obstacles + arm link geometry/hierarchy
                                     //   (capsule/box, or mesh ref where available)
  "track":  [ { "tMs": <number>, "poses": { "<frameName>": <Pose> } }, ... ]  // moving frames only
}
```

- Scene is a real `draw.Snapshot` (reuses `MarshalJSON`) → renders through the existing
  `<Snapshot>` path unchanged.
- Track is a thin per-waypoint pose map; the player tweens between adjacent steps.
- Size: static geometry once + tens of pose maps ⇒ a few hundred KB/arm, gzip-friendly.
  Trivial as a static/CDN asset.

## Scene & task (from Beanjamin config)

- **Coordinate frame:** everything in `world` millimeters, arm at the origin.
- **Obstacles (real, from fragment `e6103e56`):** `coffee-machine-base/mid/top` +
  `buffer-left/front/right` + `actuation-area`; `grinder` and `decaf-grinder` stacks;
  `tamper-base/mid/top/left/right`; `cleaner-*`; `wall`; `arm-mount`; plus shelves /
  table / ice-maker from the machine config.
- **Gripper-attached frames:** `filter` (sphere), `portafilter-handle` (box),
  `coffee-claws-middle` (box).
- **Brew sequence (from `espresso.go`, trimmed for legibility — drop the 10–25s real-world
  pauses and speech/order plumbing):**
  `home → grinder_approach → grinder_activate → tamper_approach → tamper_activate →
  coffee_approach (linear constraint) → coffee_in → coffee_locked (pivot)`.
- **Pose values (from config):** `grinder_activate` (x280,y-540,z95),
  `tamper_activate` (x615,y-436,z112), `coffee_in` (x690,y-12,z155),
  `coffee_locked_final` (x561,y9,z155), `staging` (x-400,y450,z95), plus
  `claws-position-switch` approach poses. `AllowedCollisions` from `espresso.go`
  (e.g. `filter`↔`coffee-machine-actuation-area`) are reused.

## Arms & reachability

Beanjamin's real arm is a uFactory **xArm** (`viam_ufactory` / `viam-xarm-pr5944`),
validating xArm6 in the lineup. The deepest goal (`coffee_in` ≈ 690 mm radius) sits near
xArm6's reach limit, so the planner needs the **real arm-base frame** to plan truthfully.
UR5e (~850 mm reach) should reach positionally, but **the baker must verify every goal
plans successfully for both arms** and tune base offset / drop a pose if not. This is a
gating check in the build order.

## Repo layout

```
~/viam/homepage-simulated-arm-demo
├─ cmd/bake/            # Go: plan brew sequence per arm → static assets
├─ data/
│  └─ beanjamin-config.merged.json   # secret-stripped build input (committed)
├─ static/             # baked: <arm>.brew.json, scene assets, coffeemat.glb, arm GLBs (later)
├─ web/                # SvelteKit app; depends on @viamrobotics/motion-tools
│  ├─ TrajectoryProvider / StaticProvider (WasmProvider stub)
│  ├─ TrajectoryPlayer.svelte
│  └─ routes/barista/+page.svelte  (Visualizer + overlay UI)
└─ docs/plans/         # this design
```

## UI / UX

- **Arm toggle** `[ xArm6 ‖ UR5e ]` — switching loads that arm's asset and re-frames;
  the scene stays put so the *arm* is visibly the only change.
- **"Make coffee"** — primary CTA; plays the trajectory with a slim progress bar +
  reset/replay.
- **Code drawer** — default closed; expands to show the real Go `PlanMotion` loop with the
  model argument (`xarm6.json` / `ur5e.json`) highlighted and swapping on toggle. One line
  changes; the rest is static. Caption: *"Same motion code. Different arm."*
- **Camera** — fixed flattering framing via the existing `cameraControls`; optional idle orbit.
- **Aesthetics** — follow motion-tools' `frontend-aesthetics` / prime-core conventions.

## Testing

- **Go:** assert both arms plan **all** brew poses successfully (reachability) and produce
  non-empty trajectories; assert the baked asset round-trips.
- **Vitest:** `<TrajectoryPlayer>` pose interpolation (lerp/slerp) and pose application.
- **Playwright e2e:** load `/barista`, toggle arm, click "Make coffee", assert the scene
  animates (entities update).

## Risks & open spikes

1. **motion-tools render-path export (task-1 spike).** Does `@viamrobotics/motion-tools`
   export the snapshot render path, or only `Visualizer`? If not, add a small upstream
   export or render via `Visualizer` + local config.
2. **UR5e reachability** of the deepest poses — verified by the baker; may need base-offset
   tuning or dropping a pose.
3. **WASM (v2 only).** `armplanning` → `js/wasm` with `-tags no_cgo` (pure-Go IK path
   confirmed to exist: `motionplan/ik/solver_nocgo.go`). Real build spike required before
   committing to v2; not on the v1 critical path.
4. **Secrets.** The provided config contained a live Slack bot token and machine API key.
   These are redacted in `data/beanjamin-config.merged.json` and must never be committed.
   **Rotate them** — they were pasted into a chat transcript.

## Required inputs / dependencies

- ✅ Beanjamin machine config (provided; secret-stripped copy saved).
- ⏳ Final fully-resolved config export if `beanjamin-config.merged.json` is missing any
  machine-specific obstacles (baker can be pointed at the most complete export).
- Arm kinematics: `xarm6.json`, `ur5e.json` (ship with rdk).
- `@viamrobotics/motion-tools` (npm) + `github.com/viam-labs/motion-tools` (Go).

## Sequenced build order

1. Repo scaffold (Go module + SvelteKit + motion-tools deps) **and** the motion-tools
   render-path spike.
2. `cmd/bake/` + reachability check for **both** arms.
3. Asset format + `StaticProvider`.
4. `<TrajectoryPlayer>`.
5. `/barista` route + overlay UI (toggle, Make coffee, code drawer).
6. Polish + Playwright e2e.
7. (Post-v1) WASM `WasmProvider` + interactive draggable goal/obstacles.
