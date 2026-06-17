<script lang="ts">
	import { onMount } from 'svelte'
	import { ViamAppProvider, ViamProvider } from '@viamrobotics/svelte-sdk'
	import { Visualizer } from '@viamrobotics/motion-tools'

	import { StaticProvider } from '$lib/trajectory/StaticProvider'
	import TrajectoryPlayer from '$lib/trajectory/TrajectoryPlayer.svelte'
	import { ARMS, type ArmId, type CameraPose, type Trajectory } from '$lib/trajectory/types'
	import CodeDrawer from '$lib/ui/CodeDrawer.svelte'

	const provider = new StaticProvider()

	let arm = $state<ArmId>('xarm6')
	let playing = $state(false)
	let phase = $state('')
	let trajectory = $state.raw<Trajectory | undefined>(undefined)
	let loading = $state(false)
	// Applied to the Visualizer once (stable reference), so switching arms never
	// re-frames the camera and the viewer keeps whatever view they orbited to.
	let cameraPose = $state.raw<CameraPose | undefined>(undefined)

	async function load(which: ArmId) {
		loading = true
		playing = false
		try {
			const next = await provider.load(which)
			cameraPose ??= next.cameraPose
			trajectory = next
		} catch (e) {
			console.error('trajectory load failed', e)
		} finally {
			loading = false
		}
	}

	onMount(() => load(arm))

	function selectArm(which: ArmId) {
		if (which === arm) return
		arm = which
		load(which)
	}
</script>

<div class="root">
	<ViamProvider config={{ defaultOptions: { queries: { staleTime: Infinity } } }} dialConfigs={{}}>
		<!--
			ViamAppProvider only satisfies the app-client context the Visualizer's
			internal hooks read; with empty credentials and no partID it never connects
			to app.viam.com. This is a static, client-side replay — no live machine.
		-->
		<ViamAppProvider
			serviceHost="https://app.viam.com"
			credentials={{ type: 'api-key', payload: '', authEntity: '' }}
		>
			<!-- Framing comes from the baked scene camera, applied once (see load). -->
			<Visualizer {cameraPose} inputBindingsEnabled={false}>
				{#if trajectory}
					<TrajectoryPlayer {trajectory} bind:playing bind:label={phase} />
				{/if}
			</Visualizer>
		</ViamAppProvider>
	</ViamProvider>

	<div class="overlay">
		<div class="caption">Same motion code. Different arm.</div>

		{#if phase}
			<div class="signpost">
				<span class="dot"></span>
				{phase}
			</div>
		{/if}

		<div class="bottom">
			<CodeDrawer {arm} />

			<div class="controls">
				<div class="toggle" role="group" aria-label="Select arm">
					{#each ARMS as a (a.id)}
						<button class:active={a.id === arm} disabled={loading} onclick={() => selectArm(a.id)}>
							{a.label}
						</button>
					{/each}
				</div>

				<button
					class="brew"
					disabled={!trajectory || playing || loading}
					onclick={() => (playing = true)}
				>
					{playing ? 'Brewing…' : 'Make coffee'}
				</button>
			</div>
		</div>
	</div>
</div>

<style>
	.root {
		position: fixed;
		inset: 0;
	}

	/* Hide motion-tools' built-in chrome (frame tree, transform controls, logs,
	   settings) for a clean homepage view. Those overlays are the non-canvas
	   children of the Visualizer's root; keep the child that holds the WebGL
	   canvas. */
	:global(.overflow-hidden.dark\:bg-white > div:not(:has(canvas))) {
		display: none !important;
	}

	.overlay {
		position: absolute;
		inset: 0;
		pointer-events: none;
		display: flex;
		flex-direction: column;
		justify-content: space-between;
		padding: 1.5rem;
	}
	.caption {
		font-size: 1.25rem;
		font-weight: 600;
		letter-spacing: 0.01em;
		color: #f1f5f9;
		text-shadow: 0 1px 8px rgba(0, 0, 0, 0.6);
	}
	/* Sign-post badge: current action, centered near the top during the brew. */
	.signpost {
		position: absolute;
		top: 1.4rem;
		left: 50%;
		transform: translateX(-50%);
		display: inline-flex;
		align-items: center;
		gap: 0.55rem;
		background: rgba(15, 23, 42, 0.78);
		border: 1px solid rgba(148, 163, 184, 0.3);
		border-radius: 9999px;
		padding: 0.45rem 1.1rem;
		font-size: 1rem;
		font-weight: 600;
		color: #f1f5f9;
		backdrop-filter: blur(8px);
		animation: signpost-in 0.2s ease;
	}
	.dot {
		width: 0.55rem;
		height: 0.55rem;
		border-radius: 9999px;
		background: #2dd4bf;
		animation: pulse 1.1s ease-in-out infinite;
	}
	@keyframes pulse {
		0%,
		100% {
			opacity: 1;
		}
		50% {
			opacity: 0.3;
		}
	}
	@keyframes signpost-in {
		from {
			opacity: 0;
			transform: translate(-50%, -4px);
		}
		to {
			opacity: 1;
			transform: translate(-50%, 0);
		}
	}
	.bottom {
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		gap: 0.9rem;
	}
	.controls {
		pointer-events: auto;
		display: flex;
		align-items: center;
		gap: 1rem;
	}
	.toggle {
		display: inline-flex;
		background: rgba(15, 23, 42, 0.7);
		border: 1px solid rgba(148, 163, 184, 0.3);
		border-radius: 9999px;
		padding: 0.25rem;
		backdrop-filter: blur(8px);
	}
	.toggle button {
		border: 0;
		background: transparent;
		color: #cbd5e1;
		padding: 0.5rem 1.1rem;
		border-radius: 9999px;
		cursor: pointer;
		font-size: 0.95rem;
	}
	.toggle button.active {
		background: #2dd4bf;
		color: #042f2e;
		font-weight: 600;
	}
	.brew {
		border: 0;
		background: #2dd4bf;
		color: #042f2e;
		font-weight: 700;
		padding: 0.6rem 1.4rem;
		border-radius: 9999px;
		cursor: pointer;
		font-size: 1rem;
	}
	.brew:disabled {
		opacity: 0.55;
		cursor: default;
	}
</style>
