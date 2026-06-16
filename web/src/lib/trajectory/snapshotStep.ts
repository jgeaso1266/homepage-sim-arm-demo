import type { Snapshot as SnapshotProto } from '@viamrobotics/motion-tools/lib'

import type { TrackStep } from './types'

/**
 * applyStep returns a clone of the base scene snapshot with the moving frames'
 * geometry world poses overwritten from a track step. Transform UUIDs are
 * preserved, so <Snapshot> reconciles entities in place (it keys by UUID) rather
 * than re-spawning. The geometry world pose lives in physicalObject.center (the
 * observer frame is World), matching how the baker emits it.
 */
export function applyStep(base: SnapshotProto, step: TrackStep): SnapshotProto {
	const next = base.clone()
	for (const transform of next.transforms) {
		const pose = step.poses[transform.referenceFrame]
		const center = transform.physicalObject?.center
		if (!pose || !center) continue
		center.x = pose.x
		center.y = pose.y
		center.z = pose.z
		center.oX = pose.o_x
		center.oY = pose.o_y
		center.oZ = pose.o_z
		center.theta = pose.theta
	}
	return next
}
