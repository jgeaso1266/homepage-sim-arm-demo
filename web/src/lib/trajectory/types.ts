import type { Snapshot as SnapshotProto } from '@viamrobotics/motion-tools/lib'

/** The arms the demo toggles between. Matches the baked asset filenames. */
export type ArmId = 'xarm6' | 'ur5e'

export const ARMS: { id: ArmId; label: string }[] = [
	{ id: 'xarm6', label: 'xArm6' },
	{ id: 'ur5e', label: 'UR5e' },
]

/** A world-frame pose in motion-tools' common.v1.Pose shape (mm + ov-degrees). */
export interface PoseJson {
	x: number
	y: number
	z: number
	o_x: number
	o_y: number
	o_z: number
	theta: number
}

/** One playback frame: world pose of each moving entity, keyed by entity name. */
export interface TrackStep {
	tMs: number
	poses: Record<string, PoseJson>
}

/** A baked, replayable brew trajectory for one arm. */
export interface Trajectory {
	scene: SnapshotProto
	track: TrackStep[]
}

/**
 * Source of brew trajectories. StaticProvider loads pre-baked assets; a future
 * WasmProvider could plan live in the browser behind this same interface.
 */
export interface TrajectoryProvider {
	load(arm: ArmId): Promise<Trajectory>
}
