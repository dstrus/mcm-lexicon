<!-- An option's text for reading, with the raw selector a recording matches
     beneath it in the File font when the two differ, or when the selector
     has a space at an end (The Raw Selector Rule). -->
<script lang="ts">
	import { m } from '#lib/copy/messages.ts';
	import { clipped } from '#lib/ui/clipped.ts';
	import { edgeSpace } from './values.ts';

	let { text, raw, clip = false, certainty = 'exact' }: { text: string; raw: string; clip?: boolean; certainty?: string } = $props();
	let empty = $derived(certainty === 'unknown' ? m.values.runtime : m.values.noLabel);

	let spaced = $derived(edgeSpace(raw));
	let showRaw = $derived(raw !== text || spaced);
	// A space at an end is shown as a middle dot, so it can be seen.
	let visible = $derived(raw.replace(/^ +| +$/g, (s) => '·'.repeat(s.length)));
</script>

{#if clip}
	<span class="text" use:clipped>{text || empty}</span>
{:else}
	<span class="text">{text || empty}</span>
{/if}
{#if showRaw}
	<span class="raw" title={spaced ? m.values.edgeSpace : undefined}>{visible}</span>
{/if}

<style>
	.text,
	.raw {
		display: block;
	}
	.text {
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.raw {
		font-family: var(--font-mono);
		font-size: var(--step--1);
		color: var(--muted);
		overflow-wrap: anywhere;
	}
</style>
