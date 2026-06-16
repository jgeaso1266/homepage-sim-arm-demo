import { base } from '$app/paths'
import { SnapshotProto } from '@viamrobotics/motion-tools/lib'

import type { ArmId, Trajectory, TrackStep, TrajectoryProvider } from './types'

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
		return { scene, track: doc.track }
	}
}
