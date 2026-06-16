import { describe, expect, it, vi, afterEach } from 'vitest'

// Mock the motion-tools proto so this unit test exercises StaticProvider's own
// logic (fetch → parse → shape) without depending on protobuf-es class
// resolution in the node test environment. fromJsonString just round-trips here.
vi.mock('@viamrobotics/motion-tools/lib', () => ({
	SnapshotProto: { fromJson: (v: unknown) => v },
}))

import { StaticProvider } from './StaticProvider'

// A minimal but valid snapshot protojson + a 2-step track.
const sampleDoc = {
	scene: {
		uuid: 'AAAAAAAAAAAAAAAAAAAAAA==',
		transforms: [
			{
				referenceFrame: 'arm:arm:base',
				physicalObject: {
					center: { x: 0, y: 0, z: 0, oX: 0, oY: 0, oZ: 1, theta: 0 },
					box: { dimsMm: { x: 100, y: 100, z: 100 } },
					label: 'arm:base',
				},
			},
		],
	},
	track: [
		{ tMs: 0, poses: { 'arm:arm:base': { x: 0, y: 0, z: 0, o_x: 0, o_y: 0, o_z: 1, theta: 0 } } },
		{ tMs: 40, poses: { 'arm:arm:base': { x: 1, y: 2, z: 3, o_x: 0, o_y: 0, o_z: 1, theta: 0 } } },
	],
}

afterEach(() => vi.restoreAllMocks())

describe('StaticProvider', () => {
	it('loads and parses a baked trajectory', async () => {
		vi.stubGlobal(
			'fetch',
			vi.fn(async () => new Response(JSON.stringify(sampleDoc), { status: 200 }))
		)

		const traj = await new StaticProvider().load('xarm6')

		// scene round-tripped through the mocked fromJsonString
		expect((traj.scene as { transforms: unknown[] }).transforms.length).toBe(1)
		expect(traj.track.length).toBe(2)
		expect(traj.track[1].poses['arm:arm:base'].x).toBe(1)
	})

	it('throws on a failed fetch', async () => {
		vi.stubGlobal(
			'fetch',
			vi.fn(async () => new Response('nope', { status: 404 }))
		)
		await expect(new StaticProvider().load('ur5e')).rejects.toThrow(/404/)
	})
})
