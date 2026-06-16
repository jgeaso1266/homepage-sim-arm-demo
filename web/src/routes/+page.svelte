<script lang="ts">
	// Smoke import to verify the motion-tools integration resolves and builds.
	import { Visualizer } from '@viamrobotics/motion-tools'
	import { Snapshot, SnapshotProto } from '@viamrobotics/motion-tools/lib'

	const ready = Boolean(Visualizer && Snapshot && SnapshotProto)
</script>

<main style="padding: 2rem">
	<h1>Homepage Simulated-Arm Demo</h1>
	<p>motion-tools integration resolves: {ready}</p>
	<p><a href="/barista">→ /barista</a> (coming soon)</p>
</main>
