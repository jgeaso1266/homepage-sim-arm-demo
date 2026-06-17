import { base } from '$app/paths'
import { SnapshotProto } from '@viamrobotics/motion-tools/lib'

import type { ArmId, CameraPose, Trajectory, TrackStep, TrajectoryProvider } from './types'

interface AssetDoc {
	scene: unknown
	track: TrackStep[]
}

/**
 * StaticProvider loads pre-baked brew assets from /trajectories/<arm>.brew.json
 * (produced by the Go `cmd/bake`). The scene is parsed with the motion-tools
 * SnapshotProto so it can render through the <Snapshot> component.
 */
export class StaticProvider implements TrajectoryProvider {
	async load(arm: ArmId): Promise<Trajectory> {
		const url = `${base}/trajectories/${arm}.brew.json`
		const res = await fetch(url)
		if (!res.ok) {
			throw new Error(`failed to load ${url}: ${res.status}`)
		}
		const doc = (await res.json()) as AssetDoc
		const scene = SnapshotProto.fromJson(doc.scene as never)

		// Extract the baked camera (mm) as a one-time framing, then strip it from
		// the scene: <Snapshot> re-applies sceneCamera on every snapshot change, so
		// leaving it would reset the user's view on each arm switch (and frame).
		let cameraPose: CameraPose | undefined
		const sc = scene.sceneMetadata?.sceneCamera
		if (sc?.position && sc.lookAt) {
			cameraPose = {
				position: [sc.position.x * 0.001, sc.position.y * 0.001, sc.position.z * 0.001],
				lookAt: [sc.lookAt.x * 0.001, sc.lookAt.y * 0.001, sc.lookAt.z * 0.001],
			}
			scene.sceneMetadata!.sceneCamera = undefined
		}

		return { scene, track: doc.track, cameraPose }
	}
}
