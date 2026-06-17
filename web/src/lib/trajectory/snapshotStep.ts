import type { SnapshotProto } from '@viamrobotics/motion-tools/lib'

import type { TrackStep } from './types'

/**
 * applyStep returns a clone of the base scene snapshot with the moving frames'
 * geometry world poses overwritten from a track step. Transform UUIDs are
 * preserved, so <Snapshot> reconciles entities in place (it keys by UUID) rather
 * than re-spawning.
 *
 * IMPORTANT: motion-tools' `DrawFrameSystemGeometries` emits each geometry with
 * `poseInObserverFrame.pose` = identity and the geometry's actual *world* pose in
 * `physicalObject.center` (verified against the baked asset). The renderer places
 * the mesh at `center` within the identity world frame, so the world pose must be
 * written to `center`, NOT `poseInObserverFrame`. The baker builds the track the
 * same way (it reads each transform's `center`), so the two stay consistent.
 * Transforms with no track entry (the static obstacles) keep their baked pose.
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
