<!-- Every file Save will write, as it will be written, against what the
     folder holds now: the Draft Rule's proof. A new file is all added
     lines; one Save removes is all removed. -->
<script lang="ts">
	import { goto } from '$app/navigation';
	import type { PlannedFile } from '#lib/api/types/recordings.ts';
	import { m } from '#lib/copy/messages.ts';
	import LineDiff from '#lib/ui/LineDiff.svelte';
	import { useShortcuts } from '#lib/ui/shortcuts.svelte.ts';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const t = m.files;
	let ed = $derived(data.editor);

	const changeWord = (f: PlannedFile) => t.changes[f.change];

	useShortcuts(() => [{ key: 'Escape', shown: 'Esc', label: m.recording.up, run: () => goto(`/recordings?item=${encodeURIComponent(ed.draft.name)}`) }]);
</script>

{#if !ed.files.length}
	<p class="empty">{m.recording.noFiles}</p>
{:else}
	{#if ed.target.dir}<p class="muted where">{t.inFolder({ folder: ed.target.dir })}</p>{/if}
	{#each ed.files as f (f.path)}
		<section>
			<h2 class="caps">{f.name}<span class="word">{changeWord(f)}</span></h2>
			<p class="muted path"><span class="raw">{f.path}</span></p>
			{#if f.change === 'same'}
				<pre>{f.text}</pre>
			{:else}
				<LineDiff before={f.old} after={f.text} label={t.changesTo({ file: f.name })} />
			{/if}
		</section>
	{/each}
{/if}

<style>
	.empty {
		margin-top: var(--space-5);
	}
	.where {
		margin-top: var(--space-4);
		font-size: var(--step--1);
		overflow-wrap: anywhere;
	}
	section {
		margin-top: var(--space-5);
		max-width: 100ch;
	}
	h2 {
		display: flex;
		align-items: baseline;
		gap: var(--space-3);
		font-size: var(--step--1);
		color: var(--accent);
		font-weight: 500;
		padding-bottom: var(--space-1);
		border-bottom: 1px solid var(--rule-strong);
	}
	.word {
		color: var(--muted);
		font-weight: 400;
	}
	.path {
		margin: var(--space-2) 0;
		font-size: var(--step--1);
	}
	.raw {
		font-family: var(--font-mono);
	}
	pre {
		margin: 0;
		font: 400 var(--step--1) / 1.55 var(--font-mono);
		color: var(--muted);
		white-space: pre-wrap;
		overflow-wrap: anywhere;
	}
</style>
