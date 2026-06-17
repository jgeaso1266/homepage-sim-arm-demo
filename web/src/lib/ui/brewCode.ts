import type { ArmId } from '$lib/trajectory/types'

/**
 * Per-arm Viam component model — the ONLY thing that differs between arms. The
 * drawer highlights it to make "same code, different arm" undeniable.
 */
const ARM_MODEL: Record<ArmId, string> = {
	xarm6: 'ufactory:xArm6',
	ur5e: 'universal-robots:ur5e',
}

/**
 * The code shown in the drawer. The story is Viam's hardware abstraction: the arm
 * is a component you pick in config (the `MODEL` token, the only thing that
 * changes), and the brew routine calls the Motion service the same way for any
 * arm. `MODEL` marks the swap point for highlighting.
 *
 * Representative, not literal: for a static, serverless web demo the trajectories
 * are pre-planned with Viam's motion planner (see the note under the drawer);
 * a live machine would issue these same motion.move() calls at runtime.
 */
const TEMPLATE = `# 1 · machine config — the arm is a component you swap
arm = { "name": "arm", "model": "MODEL" }

# 2 · the brew routine — the same code drives any arm
for pose in espresso_recipe:          # grinder · tamp · brew
    await motion.move("arm", pose)`

export interface CodeSegment {
	text: string
	highlight: boolean
}

/**
 * Returns the drawer code split into segments, with the arm model marked for
 * highlighting. The model is the only thing that changes between arms.
 */
export function brewCodeSegments(arm: ArmId): CodeSegment[] {
	const parts = TEMPLATE.split('MODEL')
	const model = ARM_MODEL[arm]
	const segments: CodeSegment[] = []
	parts.forEach((part, i) => {
		segments.push({ text: part, highlight: false })
		if (i < parts.length - 1) {
			segments.push({ text: model, highlight: true })
		}
	})
	return segments
}
