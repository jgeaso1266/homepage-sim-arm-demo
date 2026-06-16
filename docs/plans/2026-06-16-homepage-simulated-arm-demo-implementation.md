# Homepage Simulated-Arm Demo — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** A standalone, client-side, embeddable 3D demo where a viewer watches Beanjamin's espresso brew sequence and toggles the arm (xArm6 ↔ UR5e) to see the *same* Viam-planned motion code run on a different arm.

**Architecture:** A native-Go baker (`cmd/bake`) plans the brew sequence per arm with rdk `armplanning` and emits static assets: a scene `Snapshot` (obstacles + arm-at-home) plus a per-step pose `track` (world-frame link poses). A SvelteKit app embeds `@viamrobotics/motion-tools`' `Visualizer` + `Snapshot`, and a `<TrajectoryPlayer>` animates the arm by feeding per-step poses into the ECS. A `TrajectoryProvider` interface (`StaticProvider` now, `WasmProvider` later) is the seam.

**Tech Stack:** Go 1.25 + `go.viam.com/rdk@v0.122.0` + `github.com/viam-labs/motion-tools/draw`; SvelteKit + Svelte 5 runes + `@viamrobotics/motion-tools` (Threlte/Koota); Vitest + Playwright.

**Validated by spikes (see design doc):** rdk planning runs standalone and both arms plan all probe poses **with the gripper+filter tool chain modeled (goals keyed to the `filter` frame)**. motion-tools exports everything needed; no upstream changes.

**Key references:**
- Design: `docs/plans/2026-06-16-homepage-simulated-arm-demo-design.md`
- Config (secrets redacted): `data/beanjamin-config.merged.json`
- Spike: `cmd/bake/main.go` (will be replaced by the real baker)
- Beanjamin planning source to mirror: `~/viam/beanjamin/motion.go`, `~/viam/beanjamin/espresso.go`
- motion-tools offline builders: `~/viam/motion-tools/draw/snapshot.go` (`NewSnapshot`, `DrawFrameSystemGeometries`, `DrawFrame`, `DrawGeometry`), `draw.NewDrawnFrameSystem(...).ToTransforms()`
- motion-tools exports: `Visualizer` (main), `Snapshot`+`SnapshotProto`+transform helpers (`/lib`), ECS `useWorld`/`useQuery`/`useTrait`/`traits` (main)
- Snapshot render example: `~/viam/motion-tools/src/routes/snapshot/+page.svelte`

**Conventions:** Go per motion-tools `.claude/rules/go.md` & `testing-go.md` (`go.viam.com/test`). Frontend per `svelte.md`, `three.md`, `frontend-aesthetics.md`, `testing-frontend.md`. One focused unit per file. DRY, YAGNI, TDD, frequent commits.

---

## Phase 1 — Baker: frame system from config

### Task 1: Parse the Beanjamin config into obstacle frames

**Files:**
- Create: `internal/scene/config.go`
- Test: `internal/scene/config_test.go`
- Read input: `data/beanjamin-config.merged.json`

**Step 1: Write the failing test.** Parse the committed config and assert known obstacles load with correct world placement.

```go
package scene

import (
	"testing"
	"go.viam.com/test"
)

func TestLoadObstacles(t *testing.T) {
	obs, err := LoadObstacles("../../data/beanjamin-config.merged.json")
	test.That(t, err, test.ShouldBeNil)
	// coffee-machine-base is a world-parented box at x=740 (from config).
	cm, ok := obs["coffee-machine-base"]
	test.That(t, ok, test.ShouldBeTrue)
	test.That(t, cm.Parent, test.ShouldEqual, "world")
	test.That(t, cm.Translation.X, test.ShouldAlmostEqual, 740.0)
	// nested obstacle keeps its parent (e.g. coffee-machine-mid -> coffee-machine-base).
	mid, ok := obs["coffee-machine-mid"]
	test.That(t, ok, test.ShouldBeTrue)
	test.That(t, mid.Parent, test.ShouldEqual, "coffee-machine-base")
}
```

**Step 2: Run, verify it fails.** `go test ./internal/scene/ -run TestLoadObstacles -v` → FAIL (undefined `LoadObstacles`).

**Step 3: Implement `LoadObstacles`.** Define an `Obstacle` struct `{Name, Parent string; Translation r3.Vector; Orientation spatialmath.Orientation; Geometry spatialmath.Geometry}`. Parse the JSON `components[]`, selecting entries where `model == "erh:vmodutils:obstacle"` (and the gripper-attached `filter`/`portafilter-handle`/`coffee-claws-middle` are handled in Task 2, not here). Build `spatialmath.Geometry` from `frame.geometry` (`box`→`NewBox`, `sphere`→`NewSphere`). Skip components with no `frame`. Return `map[string]Obstacle`.

**Step 4: Run, verify pass.**

**Step 5: Commit.** `feat(baker): parse obstacle frames from beanjamin config`

### Task 2: Build the planning frame system (arm + tool chain + obstacles)

**Files:**
- Create: `internal/scene/framesystem.go`
- Test: `internal/scene/framesystem_test.go`

**Step 1: Failing test.** Build the FS for each arm and assert structure + reachability sanity (mirrors the spike).

```go
func TestBuildFrameSystem_xarm6(t *testing.T) {
	fs, err := BuildFrameSystem("arm", "../../data/kinematics/xarm6.json", "../../data/beanjamin-config.merged.json")
	test.That(t, err, test.ShouldBeNil)
	names := fs.FrameNames()
	test.That(t, names, test.ShouldContain, "arm")
	test.That(t, names, test.ShouldContain, "filter")                // tool frame
	test.That(t, names, test.ShouldContain, "coffee-machine-base")   // obstacle
}
```

**Step 2: Run, verify fails.**

**Step 3: Implement `BuildFrameSystem(armName, modelPath, configPath)`.**
- `referenceframe.ParseModelJSONFile(modelPath, armName)`; `fs := referenceframe.NewEmptyFrameSystem("fs")`; `fs.AddFrame(model, fs.World())`.
- **Tool chain (REQUIRED — spike finding):** add `gripper` static frame (z≈105) on the arm model, `filter` static frame (z≈220) on gripper, both via `referenceframe.NewStaticFrameWithGeometry` using the geometries from config (`filter`=sphere r35, `coffee-claws-middle`=box on gripper, `portafilter-handle`=box on filter). Reuse the exact translations from `data/beanjamin-config.merged.json` (filter z=220; portafilter-handle z=-107.5 on filter; coffee-claws-middle z=90 on gripper). Gripper z=105 is from the config fragment mod — hardcode with a `// from beanjamin fragment mod` comment.
- Add each obstacle from `LoadObstacles` via `NewStaticFrameWithGeometry`, parenting per `Obstacle.Parent` (add in topological order so parents exist first — sort so `world`-parented come first, then resolve remaining iteratively).

**Step 4: Run, verify pass.**

**Step 5: Commit.** `feat(baker): build planning frame system with tool chain + obstacles`

---

## Phase 2 — Baker: plan the brew sequence

### Task 3: Define the brew sequence

**Files:**
- Create: `internal/brew/sequence.go`
- Test: `internal/brew/sequence_test.go`

**Step 1: Failing test.** Assert the sequence has the expected named steps and absolute poses match config.

```go
func TestBrewSequence(t *testing.T) {
	seq := Sequence()
	names := make([]string, len(seq))
	for i, s := range seq { names[i] = s.Name }
	test.That(t, names, test.ShouldResemble, []string{
		"home", "grinder_approach", "grinder_activate",
		"tamper_approach", "tamper_activate",
		"coffee_approach", "coffee_in",
	})
	// coffee_in absolute pose from config.
	ci := seq[6]
	test.That(t, ci.Pose.Point().X, test.ShouldAlmostEqual, 689.6)
}
```

**Step 2: Run, verify fails.**

**Step 3: Implement `Sequence() []Step`.** `Step{Name string; Pose spatialmath.Pose; Linear bool}`. Use absolute poses from config: `grinder_activate`(280,-540,95 / 0,-1,0,-180), `tamper_activate`(615,-435.7,112.3 / .81,-.59,0,-180), `coffee_in`(689.6,-12.45,155 / .66,-.75,0,-179). Define `home` as a comfortable pose above the workspace (e.g. 300,0,500 pointing down). Define each `_approach` as its `_activate` pose offset +80mm along world tool-approach (translate along the orientation's -Z, i.e. standoff); compute via `spatialmath` so it's derived, not magic numbers. Mark `coffee_approach` and `coffee_in` `Linear: true` (linear constraint).

**Step 4: Run, verify pass.** **Step 5: Commit.** `feat(baker): define brew sequence from config poses`

### Task 4: Plan the sequence into a joint trajectory

**Files:**
- Create: `internal/brew/plan.go`
- Test: `internal/brew/plan_test.go`

**Step 1: Failing test.** Plan the full sequence for BOTH arms; assert success + non-empty, monotonic trajectory. (This is the real reachability gate.)

```go
func TestPlanSequence_bothArms(t *testing.T) {
	for _, arm := range []string{"xarm6", "ur5e"} {
		fs, err := scene.BuildFrameSystem("arm", "../../data/kinematics/"+arm+".json", "../../data/beanjamin-config.merged.json")
		test.That(t, err, test.ShouldBeNil)
		traj, err := PlanSequence(context.Background(), logging.NewTestLogger(t), fs, "arm", "filter", Sequence())
		test.That(t, err, test.ShouldBeNil)
		test.That(t, len(traj), test.ShouldBeGreaterThan, len(Sequence()))
	}
}
```

**Step 2: Run, verify fails.**

**Step 3: Implement `PlanSequence(ctx, logger, fs, armName, toolFrame, steps) ([]referenceframe.FrameSystemInputs, error)`.** Mirror `~/viam/beanjamin/motion.go:moveToRawPose`: start at `NewZeroInputs(fs)`; for each step, `armplanning.PlanMotion` with `Goals=[NewPlanState(FrameSystemPoses{toolFrame: NewPoseInFrame(World, step.Pose)}, nil)]`, `StartState=NewPlanState(nil, prevInputs)`, and `Constraints` with a `LinearConstraint` when `step.Linear`. Append each plan's `Trajectory()` entries; carry the final inputs of each plan as the next start. Return the concatenated trajectory.

**Step 4: Run, verify pass** (this confirms the full obstacle-aware sequence is reachable for both arms; if a pose fails, tune its approach/standoff here, not downstream). **Step 5: Commit.** `feat(baker): plan full brew sequence for both arms`

---

## Phase 3 — Baker: emit static assets

### Task 5: Build scene snapshot + world-frame pose track

**Files:**
- Create: `internal/bake/asset.go`
- Test: `internal/bake/asset_test.go`

**Step 1: Failing test.** Build the asset for one arm; assert it has a scene with obstacle + arm-link transforms and a track whose every step poses every moving frame in world coordinates.

```go
func TestBuildAsset(t *testing.T) {
	a, err := BuildAsset(ctx, logging.NewTestLogger(t), "xarm6")
	test.That(t, err, test.ShouldBeNil)
	test.That(t, len(a.Scene.Transforms()), test.ShouldBeGreaterThan, 0)
	test.That(t, len(a.Track), test.ShouldBeGreaterThan, 0)
	// every track step poses the same set of moving (arm/tool) frames.
	test.That(t, len(a.Track[0].Poses), test.ShouldEqual, len(a.Track[1].Poses))
}
```

**Step 2: Run, verify fails.**

**Step 3: Implement.** `Asset{Scene *draw.Snapshot; Track []Step}`, `Step{TMs int; Poses map[string]Pose}`.
- Scene: `snap := draw.NewSnapshot()`; `snap.DrawFrameSystemGeometries(fs, traj[0], colors)` (arm-at-home + tool) and add obstacles via `snap.DrawGeometry(...)`. Use two colors (arm vs obstacles).
- Track: for each trajectory step `inputs`, compute each moving frame's **world** pose via `fs.Transform(inputs.ToLinearInputs(), NewPoseInFrame(frameName, ZeroPose), World)` for the arm link + tool frames, and record `{frameName: pose}`. (World-frame per Spike-1 refinement so the player writes straight to `traits.Matrix`.) Space `TMs` evenly (e.g. 40ms/step).

**Step 4: Run, verify pass.** **Step 5: Commit.** `feat(baker): build scene snapshot + world-frame pose track`

### Task 6: Serialize assets to `static/`

**Files:**
- Modify: `cmd/bake/main.go` (replace the spike with the real CLI)
- Create: `internal/bake/write.go`
- Test: `internal/bake/write_test.go`
- Output: `web/static/trajectories/{xarm6,ur5e}.brew.json`

**Step 1: Failing test.** Round-trip: marshal an asset to JSON and reload; assert the scene parses as a `draw` snapshot and the track survives.

**Step 2–4:** Implement `WriteAsset(path, *Asset)` — emit `{ "scene": <snapshot via Snapshot.MarshalJSON>, "track": [...] }`. `Pose` JSON shape MUST match motion-tools' `common.v1.Pose` (`x,y,z,o_x,o_y,o_z,theta`) so the web side reuses `poseToMatrix`. Update `cmd/bake/main.go` to loop both arms → `BuildAsset` → `WriteAsset` into `web/static/trajectories/`. Run `go run ./cmd/bake` and verify both files exist and are valid JSON.

**Step 5: Commit.** `feat(baker): write brew trajectory assets for both arms`

---

## Phase 4 — Web app scaffold + provider

### Task 7: Scaffold the SvelteKit app

**Files:** Create `web/` (SvelteKit skeleton, Svelte 5, TS, Vitest, Playwright, Tailwind), add `@viamrobotics/motion-tools` + threlte peers. Use `pnpm`.

**Steps:** `pnpm create svelte@latest web` (skeleton, TS); add deps; configure `static` adapter; verify `pnpm -C web dev` serves a blank page. Add a smoke Vitest test. **Commit.** `chore(web): scaffold sveltekit app with motion-tools`

> **Verification note:** confirm the installed `@viamrobotics/motion-tools` version exports `Visualizer`, and `/lib` exports `Snapshot`/`SnapshotProto`/`poseToMatrix` (Spike 1). If the published version lags the local repo, `pnpm` link `~/viam/motion-tools` or bump.

### Task 8: Trajectory types + StaticProvider

**Files:**
- Create: `web/src/lib/trajectory/types.ts`, `web/src/lib/trajectory/StaticProvider.ts`
- Test: `web/src/lib/trajectory/StaticProvider.spec.ts`

**Step 1: Failing test.** Mock `fetch` returning a minimal `{scene, track}`; assert `StaticProvider.load('xarm6')` returns a parsed `SnapshotProto` scene + typed track.

**Step 2–4:** Define `ArmId = 'xarm6' | 'ur5e'`, `TrackStep = { tMs: number; poses: Record<string, PoseJson> }`, `Trajectory = { scene: SnapshotProto; track: TrackStep[] }`, and `interface TrajectoryProvider { load(arm: ArmId): Promise<Trajectory> }`. `StaticProvider` fetches `/trajectories/<arm>.brew.json`, parses `scene` via `SnapshotProto.fromJson(json.scene)`, returns track as-is.

**Step 5: Commit.** `feat(web): trajectory types + static provider`

---

## Phase 5 — TrajectoryPlayer

### Task 9: Verify Snapshot reconciliation behavior (spike-let)

**Files:** none (investigation; record finding in this plan/PR description).

Mount `<Visualizer><Snapshot {snapshot}/></Visualizer>`, swap the `snapshot` prop, and determine whether `Snapshot.svelte` **reconciles entities by UUID** (updates in place) or **re-spawns**. Read `~/viam/motion-tools/src/lib/components/Snapshot.svelte`. This decides Task 10's strategy:
- **Reconciles:** player = rebuild a snapshot per step (cheap) and swap the prop.
- **Re-spawns:** player = query entities by `traits.Name`, write `traits.Matrix` from track poses + `entity.changed(traits.Matrix)` + `invalidate()` in a `useTask`.

### Task 10: `<TrajectoryPlayer>` component

**Files:**
- Create: `web/src/lib/trajectory/TrajectoryPlayer.svelte`
- Test: `web/src/lib/trajectory/TrajectoryPlayer.spec.ts`

**Step 1: Failing test.** Given a 2-step track, assert that advancing time interpolates a known frame's pose (lerp position, slerp orientation) between steps, and that `playing=false` holds at step 0.

**Step 2–4:** Implement per Task 9's finding. Props: `{ trajectory: Trajectory; playing: boolean; ondone?: () => void }`. Mount the scene once. In a Threlte `useTask((delta)=>{...})`, advance a clock, find bracketing track steps, interpolate each frame's pose (use `poseToMatrix` + `Matrix4`/`Quaternion` slerp), apply to the matching entity's `traits.Matrix` (world-frame ⇒ direct write), `entity.changed(traits.Matrix)`, `invalidate()`. Stop + `ondone` at the end.

**Step 5: Commit.** `feat(web): trajectory player with pose interpolation`

---

## Phase 6 — Route + overlay UI

### Task 11: `/barista` route with arm toggle + Make coffee

**Files:**
- Create: `web/src/routes/barista/+page.svelte`, `web/src/lib/ui/ArmToggle.svelte`, `web/src/lib/ui/BrewButton.svelte`

**Steps:** Page holds `arm = $state<ArmId>('xarm6')` and `playing = $state(false)`. On arm change, `StaticProvider.load(arm)` (await; show a subtle loading state) and reset the player. Render `<Visualizer cameraPose={...}><TrajectoryPlayer .../></Visualizer>` with a fixed flattering `cameraPose`. Overlay: `<ArmToggle bind:value={arm}>` (`[ xArm6 ‖ UR5e ]`), `<BrewButton onclick={() => playing = true}>` ("Make coffee"), caption "Same motion code. Different arm." Follow `frontend-aesthetics`. **Commit.** `feat(web): barista route with arm toggle and brew button`

### Task 12: Collapsible code drawer

**Files:**
- Create: `web/src/lib/ui/CodeDrawer.svelte`, `web/src/lib/ui/brewCode.ts`

**Steps:** `brewCode.ts` exports the real Go `PlanMotion` loop text (lifted from `internal/brew/plan.go`) as a string with a placeholder for the model arg. `CodeDrawer` is default-collapsed; expands to show the code with the arm-model line (`xarm6.json` / `ur5e.json`) highlighted and swapping with the `arm` prop. **Commit.** `feat(web): collapsible code drawer proving same-code`

---

## Phase 7 — End-to-end + polish

### Task 13: Playwright e2e

**Files:** Create `web/e2e/barista.spec.ts`.

**Steps:** Load `/barista`; assert a canvas renders; click "Make coffee" and assert playback starts (e.g. a progress indicator appears / a known entity transform changes via an exposed `window.__playerState` test hook); toggle to UR5e and assert the asset reloads (network request for `ur5e.brew.json`). **Commit.** `test(web): e2e barista flow`

### Task 14: README + polish

**Files:** Create `README.md` (what it is, `make bake`, `pnpm -C web dev`, embedding note); add a `Makefile` (`bake`, `dev`, `build`, `test`). Camera framing pass, idle orbit (optional), responsive overlay. **Commit.** `docs: readme + makefile; polish`

---

## Deferred (post-v1, explicitly out of scope)

- WASM `WasmProvider` (`GOOS=js GOARCH=wasm -tags no_cgo`) + interactive draggable goal/obstacles — design seam exists (`TrajectoryProvider`); requires a build spike.
- Per-arm GLTF meshes (hybrid fidelity) — scene/track already keyed by frame name; meshes drop in by attaching to link frames.
- Homepage embed packaging.

## Global verification

- `go test ./...` green; `go run ./cmd/bake` produces both assets.
- `pnpm -C web test` (vitest) + `pnpm -C web test:e2e` (playwright) green.
- Manual: `pnpm -C web dev` → `/barista`, both arms play the brew, code drawer shows one changed line.
- @superpowers:verification-before-completion before claiming done.
