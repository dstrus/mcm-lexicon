<!-- A recording's step files as written, in replay order, for checking a
     recording character for character. -->
<script lang="ts">
	import { goto } from '$app/navigation';
	import { m } from '#lib/copy/messages.ts';
	import { useShortcuts } from '#lib/ui/shortcuts.svelte.ts';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const t = m.recording;
	let rec = $derived(data.detail.recording!);

	useShortcuts(() => [{ key: 'Escape', shown: 'Esc', label: t.up, run: () => goto(`/recordings?item=${encodeURIComponent(rec.name)}`) }]);
</script>

{#if !rec.files.length}
	<p class="empty">{t.noFiles}</p>
{:else}
	{#each rec.files as f, i (f.path)}
		<section>
			<h2 class="caps">{f.name}</h2>
			<p class="muted where"><span class="raw">{f.path}</span>{' · '}{f.mod}{' · '}{t.stepCount({ count: f.steps })}</p>
			<pre>{data.detail.texts[i]}</pre>
		</section>
	{/each}
{/if}

<style>
	.empty {
		margin-top: var(--space-5);
	}
	section {
		margin-top: var(--space-5);
	}
	h2 {
		font-size: var(--step--1);
		color: var(--accent);
		font-weight: 500;
		padding-bottom: var(--space-1);
		border-bottom: 1px solid var(--rule-strong);
	}
	.where {
		margin: var(--space-2) 0;
		font-size: var(--step--1);
	}
	.raw {
		font-family: var(--font-mono);
	}
	pre {
		margin: 0;
		padding: var(--space-3);
		background: var(--glass-strong);
		border: 1px solid var(--rule);
		border-radius: var(--radius);
		font: 400 var(--step--1) / 1.55 var(--font-mono);
		white-space: pre-wrap;
		overflow-wrap: anywhere;
		max-width: 100ch;
	}
</style>
