<!-- The whole draft, for review: every recording it makes, changes, or
     deletes, each file as Save will write it against what's on disk now,
     and the one Save that writes it all. -->
<script lang="ts">
	import { invalidateAll } from '$app/navigation';
	import { ApiError } from '#lib/api/client.ts';
	import type { Change, SavedRecording } from '#lib/api/types/recordings.ts';
	import Fill from '#lib/copy/Fill.svelte';
	import { m } from '#lib/copy/messages.ts';
	import { discardAll, discardDraft, recordingHref, restore, saveAll } from '#lib/recordings/api.ts';
	import LineDiff from '#lib/ui/LineDiff.svelte';
	import { useShortcuts } from '#lib/ui/shortcuts.svelte.ts';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const t = m.changes;

	let changes = $derived(data.changes);
	let editing = $derived(data.status.editing);
	let working = $state(false);
	let error = $state('');
	let done = $state<SavedRecording[]>([]);

	async function run(job: () => Promise<unknown>) {
		working = true;
		error = '';
		try {
			await job();
			await invalidateAll();
		} catch (e) {
			error = e instanceof ApiError ? e.message : String(e);
			await invalidateAll();
		} finally {
			working = false;
		}
	}
	const saveNow = () =>
		run(async () => {
			done = (await saveAll()).saved;
		});
	const discardEverything = () => run(() => discardAll());
	const discardOne = (c: Change) => run(() => (c.kind === 'deleted' ? restore(c.editor.draft.name) : discardDraft(c.editor.draft.name)));

	// Where a recording is written, as a reader names it.
	const folder = (c: Change) => (c.editor.target.kind === 'overwrite' ? m.draft.overwrite : c.editor.target.kind === 'data' ? m.draft.data : c.editor.target.name);
	let canSave = $derived(editing && !working && changes.recordings.length > 0 && !changes.unsaved);

	useShortcuts(() => [{ key: 'Enter', label: t.saveAll, command: true, disabled: !canSave, hidden: !changes.recordings.length, run: saveNow }]);
</script>

<h1 class="caps">{t.title}</h1>

{#if done.length}
	<div class="done" role="status">
		{#each done as r (r.name)}
			<p>{r.kind === 'deleted' ? t.deleted({ name: r.name }) : t.saved({ name: r.name })}{#if r.backup}{' '}{t.backupIn({ backup: r.backup })}{/if}</p>
		{/each}
	</div>
{/if}
{#if error}<p class="notice error" role="alert">{error}</p>{/if}

{#if !editing}
	<p class="lead muted">{t.readOnly}</p>
{:else if !changes.recordings.length}
	<p class="lead muted">{t.none}</p>
{:else}
	<p class="lead">{t.summary({ files: m.recording.fileCount({ count: changes.files }), recordings: t.recordingCount({ count: changes.recordings.length }) })}</p>
	<div class="actions">
		<button type="button" class="button primary" disabled={!canSave} onclick={saveNow}>{t.saveAll}</button>
		<button type="button" class="button" disabled={working} onclick={discardEverything}>{t.discardAll}</button>
	</div>
	{#if changes.unsaved === 'running'}<p class="notice warn">{m.draft.running}</p>{/if}
	{#if changes.unsaved === 'blocked'}<p class="notice warn">{t.blocked}</p>{/if}

	{#each changes.recordings as c (c.editor.draft.name)}
		{@const ed = c.editor}
		<section>
			<h2 class="caps">
				<a href={recordingHref(ed.draft.name)}>{ed.draft.name}</a>
				<span class="word">{t.kinds[c.kind]}</span>
			</h2>
			<p class="muted where">
				{#if c.kind === 'deleted'}{t.removesFrom({ folder: folder(c) })}{:else}{t.writesTo({ folder: folder(c) })}{/if}
				<button type="button" class="textbutton" disabled={working} onclick={() => discardOne(c)}>{c.kind === 'deleted' ? t.restore : c.kind === 'new' ? t.discardNew : t.discard}</button>
			</p>
			{#if ed.target.kind === 'none'}<p class="notice warn">{m.draft.chooseFolder}</p>{/if}
			{#if ed.target.kind === 'new-mod' && c.kind !== 'deleted'}<p class="notice"><Fill text={(ed.target.exists ? m.draft.notListed : m.draft.newMod).text}>{#snippet mod()}<span class="raw">{ed.target.name}</span>{/snippet}</Fill></p>{/if}
			{#if ed.target.losesTo && c.kind !== 'deleted'}<p class="notice warn">{m.draft.losesTo({ mod: ed.target.losesTo })}</p>{/if}
			{#each ed.problems as p, i (i)}<p class="notice error">{p}</p>{/each}
			{#each ed.files.filter((f) => f.change !== 'same') as f (f.path)}
				<h3 class="file"><span class="raw">{f.path}</span><span class="word caps">{m.files.changes[f.change]}</span></h3>
				<LineDiff before={f.old} after={f.text} label={m.files.changesTo({ file: f.name })} />
			{/each}
		</section>
	{/each}
{/if}

<style>
	h1 {
		font-size: var(--step-3);
		font-weight: 600;
		line-height: 1;
	}
	.lead {
		margin: var(--space-2) 0 var(--space-4);
		max-width: 65ch;
	}
	.actions {
		display: flex;
		gap: var(--space-2);
		margin-bottom: var(--space-3);
	}
	.done {
		margin: var(--space-3) 0;
		max-width: 80ch;
		overflow-wrap: anywhere;
	}
	.done p {
		margin: 0 0 var(--space-1);
	}
	.notice {
		max-width: 70ch;
		margin: var(--space-2) 0;
	}
	section {
		margin-top: var(--space-6);
		max-width: 100ch;
	}
	h2 {
		display: flex;
		align-items: baseline;
		gap: var(--space-3);
		font-size: var(--step--1);
		font-weight: 500;
		color: var(--accent);
		padding-bottom: var(--space-1);
		border-bottom: 1px solid var(--rule-strong);
	}
	h2 a {
		color: inherit;
		text-decoration: none;
	}
	h2 a:hover {
		text-decoration: underline;
	}
	.word {
		color: var(--muted);
		font-weight: 400;
	}
	.where {
		margin: var(--space-2) 0;
		font-size: var(--step--1);
	}
	.where .textbutton {
		margin-left: var(--space-2);
		font-size: inherit;
	}
	.file {
		display: flex;
		align-items: baseline;
		gap: var(--space-3);
		margin: var(--space-4) 0 var(--space-1);
		font-size: var(--step--1);
		font-weight: 400;
	}
	.raw {
		font-family: var(--font-mono);
	}
</style>
