<!--
@component
Plays a baked brew Trajectory by swapping the <Snapshot> prop over time. The
scene renders at its start pose; on play, a clock advances through the track and
each step's world poses are applied to a clone of the scene snapshot, which
<Snapshot> reconciles in place (it keys entities by UUID).
-->
<script lang="ts">
	import { onDestroy } from 'svelte'
	import { Snapshot } from '@viamrobotics/motion-tools/lib'

	import type { Trajectory } from './types'
	import { applyStep } from './snapshotStep'

	interface Props {
		trajectory: Trajectory
		playing: boolean
		ondone?: () => void
	}

	let { trajectory, playing = $bindable(), ondone }: Props = $props()

	let current = $state.raw(trajectory.scene)
	let raf = 0
	let startTime = 0

	const track = $derived(trajectory.track)
	const durationMs = $derived(track.length ? track[track.length - 1].tMs : 0)

	// Reset to the scene's start pose whenever the trajectory changes.
	$effect(() => {
		void trajectory
		current = trajectory.scene
	})

	function stop() {
		if (raf) cancelAnimationFrame(raf)
		raf = 0
		startTime = 0
	}

	function frame(now: number) {
		if (!startTime) startTime = now
		const elapsed = now - startTime

		// Find the latest track step at or before elapsed time.
		let i = 0
		while (i + 1 < track.length && track[i + 1].tMs <= elapsed) i++
		current = applyStep(trajectory.scene, track[i])

		if (elapsed >= durationMs) {
			stop()
			playing = false
			ondone?.()
			return
		}
		raf = requestAnimationFrame(frame)
	}

	$effect(() => {
		if (playing && track.length > 0) {
			stop()
			raf = requestAnimationFrame(frame)
		} else if (!playing) {
			stop()
		}
	})

	onDestroy(stop)
</script>

<Snapshot snapshot={current} />
