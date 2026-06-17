<!--
@component
A collapsible drawer showing the brew-planning code. Default-collapsed for a
clean view; expands to show the real Go planning loop with the arm-model name
highlighted, making "same code, different arm" undeniable — only the highlighted
token changes when you toggle arms.
-->
<script lang="ts">
	import type { ArmId } from '$lib/trajectory/types'
	import { brewCodeSegments } from './brewCode'

	interface Props {
		arm: ArmId
	}

	let { arm }: Props = $props()
	let open = $state(false)

	const segments = $derived(brewCodeSegments(arm))
</script>

<div class="drawer" class:open>
	<button class="tab" onclick={() => (open = !open)} aria-expanded={open}>
		<span class="chev" class:open>›</span>
		{open ? 'Hide code' : 'Show the code'}
	</button>

	{#if open}
		<pre class="code"><code
				>{#each segments as seg, i (i)}{#if seg.highlight}<span class="hl"
							>{seg.text}</span
						>{:else}{seg.text}{/if}{/each}</code
			></pre>
		<p class="note">
			Only the highlighted arm model changes — the motion code is identical. Viam plans
			the path for whichever arm is configured. (Pre-planned here so the demo runs in your
			browser with no server.)
		</p>
	{/if}
</div>

<style>
	.drawer {
		pointer-events: auto;
		background: rgba(8, 13, 24, 0.82);
		border: 1px solid rgba(148, 163, 184, 0.25);
		border-radius: 0.75rem;
		backdrop-filter: blur(10px);
		overflow: hidden;
		max-width: 560px;
	}
	.tab {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		width: 100%;
		border: 0;
		background: transparent;
		color: #cbd5e1;
		padding: 0.6rem 0.9rem;
		font-size: 0.9rem;
		cursor: pointer;
		text-align: left;
	}
	.chev {
		display: inline-block;
		transition: transform 0.15s ease;
		font-size: 1.1rem;
		line-height: 1;
	}
	.chev.open {
		transform: rotate(90deg);
	}
	.code {
		margin: 0;
		padding: 0.25rem 1rem 0.9rem;
		font-family: ui-monospace, 'SF Mono', Menlo, Consolas, monospace;
		font-size: 0.78rem;
		line-height: 1.5;
		color: #cbd5e1;
		white-space: pre;
		overflow-x: auto;
	}
	.hl {
		background: rgba(45, 212, 191, 0.22);
		color: #5eead4;
		font-weight: 700;
		border-radius: 3px;
		padding: 0 2px;
	}
	.note {
		margin: 0;
		padding: 0 1rem 0.8rem;
		font-size: 0.75rem;
		color: #94a3b8;
	}
</style>
