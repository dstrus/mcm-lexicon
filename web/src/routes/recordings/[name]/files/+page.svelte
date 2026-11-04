<!-- Every file Save will write, as it will be written, against what the
     folder holds now: the Draft Rule's proof. A new file is all added
     lines; one Save removes is all removed. -->
<script lang="ts">
	import { goto, invalidate } from '$app/navigation';
	import { page } from '$app/state';
	import type { Status } from '#lib/api/types/api.ts';
	import { ApiError } from '#lib/api/client.ts';
	import type { PlannedFile } from '#lib/api/types/recordings.ts';
	import Fill from '#lib/copy/Fill.svelte';
	import { m } from '#lib/copy/messages.ts';
	import { putFileText, recordingsDependency } from '#lib/recordings/api.ts';
	import CodeArea from '#lib/ui/CodeArea.svelte';
	import LineDiff from '#lib/ui/LineDiff.svelte';
	import { useShortcuts } from '#lib/ui/shortcuts.svelte.ts';
	import { tick } from 'svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const t = m.files;
	let ed = $derived(data.editor);
	let canEdit = $derived((page.data.status as Status).editing && !ed.deleted);
	// The mod the recording is in now, when Save writes elsewhere: a
	// recording from another mod, whose files here are read against its.
	let home = $derived(ed.recording?.mod && ed.target.kind !== 'data' && ed.recording.mod.toLowerCase() !== ed.target.name.toLowerCase() ? ed.recording.mod : '');
	// The files Save writes, in order: the info file, then the step files,
	// which is how the server numbers them.
	// The files that can be typed: the recording's own, not those Save
	// removes, nor the empty ones covering another mod's.
	let written = $derived(ed.files.filter((f) => f.change !== 'removed' && !f.covers));

	const changeWord = (f: PlannedFile) => t.changes[f.change];

	// One file at a time is typed: its text as Save would write it, until
	// Apply reads it into the draft or Cancel drops it.
	let typing = $state(-1);
	let text = $state('');
	let box = $state<HTMLTextAreaElement>();
	let working = $state(false);
	let error = $state('');
	async function startTyping(i: number) {
		typing = i;
		text = written[i]!.text;
		error = '';
		await tick();
		box?.focus();
	}
	function cancel() {
		typing = -1;
		error = '';
	}
	async function applyText() {
		if (typing < 0 || working) return;
		working = true;
		error = '';
		try {
			await putFileText(ed.draft.name, typing, text);
			await invalidate(recordingsDependency);
			typing = -1;
		} catch (e) {
			error = e instanceof ApiError ? e.message : String(e);
		} finally {
			working = false;
		}
	}
	function keys(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			e.preventDefault();
			e.stopPropagation();
			cancel();
		} else if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
			e.preventDefault();
			e.stopPropagation();
			applyText();
		}
	}

	useShortcuts(() =>
		typing >= 0
			? [
					{ key: 'Enter', label: t.apply, command: true, disabled: working, run: applyText },
					{ key: 'Escape', shown: 'Esc', label: t.cancel, run: cancel }
				]
			: [{ key: 'Escape', shown: 'Esc', label: m.recording.up, run: () => goto(`/recordings?item=${encodeURIComponent(ed.draft.name)}`) }]
	);
</script>

{#if !ed.files.length}
	<p class="empty">{m.recording.noFiles}</p>
{:else}
	{#if home && !ed.drafted}<p class="muted where">{t.asIn({ mod: home })}</p>
	{:else if ed.target.dir}<p class="muted where">{t.inFolder({ folder: ed.target.dir })}{#if home}{' '}{t.against({ mod: home })}{/if}</p>{/if}
	{#each ed.files as f (f.path)}
		{@const i = written.indexOf(f)}
		<section>
			<h2 class="caps">
				{f.name}<span class="word">{changeWord(f)}</span>
				{#if canEdit && i >= 0 && typing !== i}<button type="button" class="textbutton edit" disabled={working} onclick={() => startTyping(i)}>{t.editText}</button>{/if}
			</h2>
			<p class="muted path"><span class="raw">{f.path}</span></p>
			{#if f.covers}<p class="note"><Fill text={t.covers.text}>{#snippet mod()}<span class="raw">{f.covers}</span>{/snippet}</Fill></p>{/if}
			{#if i >= 0 && typing === i}
				<!-- The file as text, in the file font with its parts set apart, read into the draft on Apply. -->
				<CodeArea bind:value={text} bind:box label={t.editing({ file: f.name })} rows={Math.min(40, Math.max(6, text.split('\n').length + 1))} disabled={working} onkeydown={keys} />
				<div class="actions">
					<button type="button" class="button primary" disabled={working} onclick={applyText}>{t.apply}</button>
					<button type="button" class="button" disabled={working} onclick={cancel}>{t.cancel}</button>
					<p class="muted small">{t.typedNote}</p>
				</div>
				{#if error}<p class="notice error" role="alert">{error}</p>{/if}
			{:else if f.change === 'same'}
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
	.edit {
		margin-left: auto;
		font: 400 var(--step--1) var(--font-text);
		text-transform: none;
		letter-spacing: 0;
	}
	.actions {
		display: flex;
		align-items: baseline;
		flex-wrap: wrap;
		gap: var(--space-3);
		margin-top: var(--space-3);
	}
	.small {
		margin: 0;
		font-size: var(--step--1);
		flex-basis: 100%;
	}
	.notice {
		margin-top: var(--space-3);
	}
	.path {
		margin: var(--space-2) 0;
		font-size: var(--step--1);
	}
	.note {
		margin: 0 0 var(--space-2);
		max-width: 70ch;
		font-size: var(--step--1);
		color: var(--muted);
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
