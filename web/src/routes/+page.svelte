<script lang="ts">
	// Smoke import to verify the motion-tools integration resolves and builds.
	// (MotionTools is the Visualizer component; the local dist predates the
	// Visualizer alias, so we import the name the built package exposes.)
	import { MotionTools } from '@viamrobotics/motion-tools'
	import { Snapshot, SnapshotProto } from '@viamrobotics/motion-tools/lib'

	const ready = Boolean(MotionTools && Snapshot && SnapshotProto)
</script>

<main style="padding: 2rem">
	<h1>Homepage Simulated-Arm Demo</h1>
	<p>motion-tools integration resolves: {ready}</p>
	<p><a href="/barista">→ /barista</a> (coming soon)</p>
</main>
