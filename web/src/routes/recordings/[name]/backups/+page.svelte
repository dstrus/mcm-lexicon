<!-- A recording's backups: its files as each Save or delete found them,
     newest first. The selected one shows its files against what's on disk
     now, as Save would put them back, and Restore puts it in the draft. -->
<script lang="ts">
	import { goto, invalidate } from '$app/navigation';
	import { page } from '$app/state';
	import { ApiError } from '#lib/api/client.ts';
	import type { Status } from '#lib/api/types/api.ts';
	import type { BackupView } from '#lib/api/types/recordings.ts';
	import { m } from '#lib/copy/messages.ts';
	import { count, realTime } from '#lib/format.ts';
	import { discardDraft, recordingsDependency, restoreFrom } from '#lib/recordings/api.ts';
	import Diamond from '#lib/ui/Diamond.svelte';
	import LineDiff from '#lib/ui/LineDiff.svelte';
	import { Selection } from '#lib/ui/selection.svelte.ts';
	import { useShortcuts } from '#lib/ui/shortcuts.svelte.ts';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const t = m.backups;

	let ed = $derived(data.editor);
	let list = $derived(data.backups.backups);
	let restoring = $derived(data.backups.restoring);
	let editing = $derived((page.data.status as Status).editing && !ed.deleted);
	const selection = new Selection(() => list, (b) => b.id, { param: 'item' });
	let current = $derived(selection.current);

	const when = (b: BackupView) => (b.at ? realTime(Date.parse(b.at)) : b.id);
	const reason = (b: BackupView) => (b.reason === 'save' || b.reason === 'delete' || b.reason === 'restore' ? t.reasons[b.reason] : t.reasons.unknown);
	const same = (b: BackupView) => b.planned.every((f) => f.change === 'same');

	let working = $state(false);
	let error = $state('');
	async function run(job: () => Promise<unknown>) {
		working = true;
		error = '';
		try {
			await job();
			await invalidate(recordingsDependency);
		} catch (e) {
			error = e instanceof ApiError ? e.message : String(e);
		} finally {
			working = false;
		}
	}
	const restore = () => current && run(() => restoreFrom(ed.draft.name, current!.id));
	const discard = () => run(() => discardDraft(ed.draft.name));
	let canRestore = $derived(editing && !!current && restoring !== current.id && !same(current) && !working);

	useShortcuts(() => [
		{ key: 'ArrowDown', label: m.common.next, hidden: true, run: () => selection.move(1) },
		{ key: 'ArrowUp', label: m.common.previous, hidden: true, run: () => selection.move(-1) },
		...(editing && current ? [{ key: 'r', shown: 'R', label: t.restore, disabled: !canRestore, run: restore }] : []),
		{ key: 'Escape', shown: 'Esc', label: m.recording.up, run: () => goto(`/recordings?item=${encodeURIComponent(ed.draft.name)}`) }
	]);
</script>

{#if !list.length}
	<p class="empty">{t.none}</p>
{:else}
	<table>
		<thead>
			<tr>
				<th class="mark" aria-label={m.common.selected}></th>
				<th class="caps">{t.when}</th>
				<th class="caps">{t.madeBy}</th>
				<th class="caps n">{t.steps}</th>
				<th class="caps n">{t.files}</th>
			</tr>
		</thead>
		<tbody>
			{#each list as b (b.id)}
				<tr aria-selected={selection.is(b)} onclick={() => selection.select(b)}>
					<td class="mark">{#if selection.is(b)}<Diamond />{/if}</td>
					<td>{when(b)}{#if restoring === b.id}<span class="word caps">{m.changes.kinds.restored}</span>{/if}</td>
					<td class="caps small">{reason(b)}</td>
					<td class="n">{count(b.steps)}</td>
					<td class="n">{count(b.files)}</td>
				</tr>
			{/each}
		</tbody>
	</table>

	{#if current}
		<section class="chosen" aria-label={t.pane}>
			<div class="actions">
				{#if editing}
					<button type="button" class="button primary" title={t.restoreTitle} disabled={!canRestore} onclick={restore}>{t.restore}</button>
				{/if}
				<p class="muted">{#if current.mod}{t.inFolder({ folder: current.mod })}{/if}</p>
			</div>
			{#if restoring === current.id}
				<p class="notice">{t.restoring} <button type="button" class="textbutton" disabled={working} onclick={discard}>{m.draft.discard}</button></p>
			{:else if same(current)}
				<p class="notice">{t.sameAsDisk}</p>
			{:else if !editing}
				<p class="notice">{t.readOnly}</p>
			{/if}
			{#if error}<p class="notice error" role="alert">{error}</p>{/if}
			{#each current.planned as f (f.path)}
				<div class="file">
					<h2 class="caps">{f.name}<span class="word">{m.files.changes[f.change]}</span></h2>
					<p class="muted path"><span class="raw">{f.path}</span></p>
					{#if f.change === 'same'}
						<pre>{f.text}</pre>
					{:else}
						<LineDiff before={f.old} after={f.text} label={m.files.changesTo({ file: f.name })} />
					{/if}
				</div>
			{/each}
		</section>
	{/if}
{/if}

<style>
	.empty {
		margin-top: var(--space-5);
		max-width: 70ch;
	}
	table {
		width: 100%;
		max-width: 48rem;
		border-collapse: collapse;
		margin-top: var(--space-4);
	}
	th,
	td {
		text-align: left;
		padding: var(--space-1) var(--space-3) var(--space-1) 0;
		border-bottom: 1px solid var(--rule);
	}
	th {
		font-size: var(--step--1);
		font-weight: 500;
		color: var(--muted);
		border-bottom-color: var(--rule-strong);
	}
	.mark {
		width: 1.25rem;
		padding-left: var(--space-2);
	}
	.n {
		text-align: right;
		font-variant-numeric: tabular-nums;
	}
	.small {
		font-size: var(--step--1);
	}
	tbody tr {
		cursor: default;
	}
	tbody tr:hover:not([aria-selected='true']) td {
		background: var(--glass);
	}
	tr[aria-selected='true'] td {
		background: var(--select);
	}
	/* Where the arrow keys are: a hairline of the accent around the row. */
	tr[aria-selected='true'] {
		outline: 1px solid var(--keys);
		outline-offset: -1px;
	}
	.word {
		margin-left: var(--space-2);
		font-size: var(--step--1);
		color: var(--muted);
	}
	.chosen {
		margin-top: var(--space-5);
		max-width: 100ch;
	}
	.actions {
		display: flex;
		align-items: baseline;
		gap: var(--space-3);
		flex-wrap: wrap;
	}
	.actions p {
		margin: 0;
		font-size: var(--step--1);
		overflow-wrap: anywhere;
	}
	.notice {
		margin-top: var(--space-3);
		max-width: 70ch;
	}
	.file {
		margin-top: var(--space-5);
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
	h2 .word {
		margin: 0;
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
