import type { SnapshotProto } from '@viamrobotics/motion-tools/lib'

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
	/** Sign-post text shown while this frame plays (e.g. "Grinding"); may be absent. */
	label?: string
}

/** A camera framing in meters, for the Visualizer's cameraPose prop. */
export interface CameraPose {
	position: [number, number, number]
	lookAt: [number, number, number]
}

/** A baked, replayable brew trajectory for one arm. */
export interface Trajectory {
	scene: SnapshotProto
	track: TrackStep[]
	/** Initial framing extracted from the baked scene (camera stripped from the
	 * scene so <Snapshot> never re-applies it and reset the user's view). */
	cameraPose?: CameraPose
}

/**
 * Source of brew trajectories. StaticProvider loads pre-baked assets; a future
 * WasmProvider could plan live in the browser behind this same interface.
 */
export interface TrajectoryProvider {
	load(arm: ArmId): Promise<Trajectory>
}
