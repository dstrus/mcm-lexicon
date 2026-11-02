<script lang="ts">
	import Fill from '#lib/copy/Fill.svelte';
	import { m } from '#lib/copy/messages.ts';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	let source = $derived(data.status.source);
</script>

<h1 class="caps">{m.menus.title}</h1>
<p class="muted where">
	{#if source?.kind === 'mo2'}
		<Fill text={m.menus.inInstance.text}>{#snippet path()}<span class="path">{source.path}</span>{/snippet}{#snippet profile()}<span class="path">{source.profile}</span>{/snippet}</Fill>
	{:else if source}
		<Fill text={m.menus.inData.text}>{#snippet path()}<span class="path">{source.path}</span>{/snippet}</Fill>
	{:else}
		<Fill text={m.menus.inCatalog.text}>{#snippet path()}<span class="path">{data.status.catalogFile}</span>{/snippet}</Fill>
	{/if}
</p>
<p class="empty">{m.menus.notYet}</p>

<style>
	h1 {
		font-size: var(--step-3);
		font-weight: 600;
		line-height: 1;
	}
	.where {
		margin: var(--space-2) 0 var(--space-5);
		max-width: 70ch;
	}
	.path {
		color: var(--ink);
		word-break: break-all;
	}
	.empty {
		max-width: 60ch;
	}
</style>
