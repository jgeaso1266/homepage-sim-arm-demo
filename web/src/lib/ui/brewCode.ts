import type { ArmId } from '$lib/trajectory/types'

/**
 * The brew-planning code shown in the drawer. It mirrors the real Go baker
 * (internal/brew/plan.go + cmd/bake): the same sequence is planned for whichever
 * arm. `ARM` marks the only thing that differs between the two arms — the
 * kinematics model — so the drawer can highlight it.
 */
const TEMPLATE = `// Plan Beanjamin's espresso routine for the selected arm.
fs := scene.BuildFrameSystem("arm", "ARM.json", machineConfig)
inputs := brew.ReadyConfig("ARM")

for _, step := range brew.Sequence() {
    plan, _, err := armplanning.PlanMotion(ctx, logger, &armplanning.PlanRequest{
        FrameSystem: fs,
        Goals:       []*PlanState{step.Goal()},
        StartState:  armplanning.NewPlanState(nil, inputs),
        Constraints: step.Constraints(),
    })
    if err != nil {
        return err
    }
    trajectory = append(trajectory, plan.Trajectory()...)
    inputs = trajectory.Last()
}`

export interface CodeSegment {
	text: string
	highlight: boolean
}

/**
 * Returns the brew code split into segments, with the arm model name marked for
 * highlighting. The arm name is the ONLY thing that changes between arms.
 */
export function brewCodeSegments(arm: ArmId): CodeSegment[] {
	const parts = TEMPLATE.split('ARM')
	const segments: CodeSegment[] = []
	parts.forEach((part, i) => {
		segments.push({ text: part, highlight: false })
		if (i < parts.length - 1) {
			segments.push({ text: arm, highlight: true })
		}
	})
	return segments
}
