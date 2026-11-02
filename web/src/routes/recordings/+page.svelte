<!-- Every recording in the list, with how its steps check against the
     catalog, and the selected one in the pane. -->
<script lang="ts">
	import { goto, invalidateAll } from '$app/navigation';
	import Switch from '#lib/ui/Switch.svelte';
	import { leave, param, setParams } from '#lib/ui/address.svelte.ts';
	import { ApiError } from '#lib/api/client.ts';
	import type { Summary } from '#lib/api/types/recordings.ts';
	import { m } from '#lib/copy/messages.ts';
	import { count } from '#lib/format.ts';
	import { create, deleteRecording, recordingHref, restore, setOpen } from '#lib/recordings/api.ts';
	import { countsText } from '#lib/recordings/steps.ts';
	import Diamond from '#lib/ui/Diamond.svelte';
	import SortHeader from '#lib/ui/SortHeader.svelte';
	import { clipped } from '#lib/ui/clipped.ts';
	import { fitToViewport } from '#lib/ui/fit.ts';
	import { Selection } from '#lib/ui/selection.svelte.ts';
	import { deleteShortcuts, useShortcuts } from '#lib/ui/shortcuts.svelte.ts';
	import { sorted, type Sort } from '#lib/ui/sort.ts';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const t = m.recordings;

	let reply = $derived(data.recordings);
	let all = $derived(reply?.recordings ?? []);
	let query = $state('');
	let sort = $state<Sort>({ key: 'name', descending: false });
	let error = $state('');

	const failing = (r: Summary) => r.counts.notFound + r.counts.notAllowed + r.counts.modMissing;
	let shown = $derived.by(() => {
		const q = query.trim().toLowerCase();
		const list = all.filter((r) => !q || r.name.toLowerCase().includes(q) || r.title.toLowerCase().includes(q) || r.mod.toLowerCase().includes(q));
		return sorted(list, sort, {
			name: (r) => r.name,
			mod: (r) => r.mod,
			steps: (r) => r.steps,
			menus: (r) => r.menus,
			check: (r) => failing(r) * 1000 + r.counts.unchecked
		});
	});
	const selection = new Selection(() => shown, (r) => r.name, { param: 'item' });
	let current = $derived(selection.current);
	const open = (r: Summary) => goto(recordingHref(r.name));

	async function openForAdding(r: Summary) {
		error = '';
		try {
			await setOpen(r.name);
			await invalidateAll();
		} catch (e) {
			error = e instanceof ApiError ? e.message : String(e);
		}
	}

	// The pane's forms: a new recording (?mode=new), and confirming a
	// delete (?mode=delete).
	let mode = $derived(param('mode'));
	let newName = $state('');
	let newAuthor = $state('');
	let newAutorun = $state('false');
	let working = $state(false);
	// Read-only unless editing is on.
	let editing = $derived(data.status.editing);
	function startNew() {
		newName = '';
		newAuthor = reply?.author ?? '';
		newAutorun = 'false';
		error = '';
		setParams({ mode: 'new' });
	}
	async function createNew(e?: SubmitEvent) {
		e?.preventDefault();
		if (!newName.trim() || working) return;
		working = true;
		error = '';
		try {
			const ed = await create({ name: newName, author: newAuthor, autorun: newAutorun === 'true' });
			await setOpen(ed.draft.name);
			await goto(recordingHref(ed.draft.name), { invalidateAll: true });
		} catch (err) {
			error = err instanceof ApiError ? err.message : String(err);
		} finally {
			working = false;
		}
	}
	// Why the selected recording can't be deleted, in words; "" when it can.
	const whyNot = (r: Summary) => (r.deletes === '' ? '' : r.deletes === 'new' ? t.deleteNew : t.deleteNotOurs({ mod: r.mod }));
	let canDelete = $derived(editing && !!current && current.deletes === '' && !current.deleted);
	// Deleting puts the deletion in the draft, where it's reviewed with the
	// rest and can be undone; Save does it.
	async function markDeleted(on: boolean) {
		if (!current || working) return;
		working = true;
		error = '';
		try {
			await (on ? deleteRecording(current.name) : restore(current.name));
			await invalidateAll();
		} catch (err) {
			error = err instanceof ApiError ? err.message : String(err);
		} finally {
			working = false;
		}
	}

	useShortcuts(() =>
		mode === 'new' && editing
			? [
					...(mode === 'new' ? [{ key: 'Enter', label: t.create, command: true, disabled: !newName.trim() || working, run: () => createNew() }] : []),
					{ key: 'Escape', shown: 'Esc', label: m.common.cancel, run: () => leave('mode') }
				]
			: [
					{ key: 'ArrowDown', label: m.common.next, hidden: true, run: () => selection.move(1) },
					{ key: 'ArrowUp', label: m.common.previous, hidden: true, run: () => selection.move(-1) },
					{ key: 'Enter', label: m.common.open, disabled: !current, run: () => current && open(current) },
					...(editing
						? [
								{ key: 'n', shown: 'N', label: t.new, run: startNew },
								...(current?.deleted
									? [{ key: 'r', shown: 'R', label: t.restore, disabled: working, run: () => markDeleted(false) }]
									: deleteShortcuts({ label: t.delete, disabled: !canDelete || working, run: () => markDeleted(true) }))
							]
						: [])
				]
	);
</script>

<div class="bar">
	<h1 class="caps">{t.title}</h1>
	{#if all.length}
		<span class="muted n">{t.count({ count: all.length })}</span>
		<input type="search" bind:value={query} placeholder={t.filter} aria-label={t.filterLabel} />
	{/if}
</div>

{#snippet newForm()}
	<form class="new" onsubmit={createNew}>
		<h2 class="caps">{t.newTitle}</h2>
		<label>
			<span class="caps">{t.name}</span>
			<!-- svelte-ignore a11y_autofocus -->
			<input bind:value={newName} required autocomplete="off" spellcheck="false" autofocus />
		</label>
		<p class="muted note">{t.nameNote}</p>
		<label>
			<span class="caps">{t.author}</span>
			<input bind:value={newAuthor} autocomplete="off" spellcheck="false" />
		</label>
		<span class="caps label">{t.autorun}</span>
		<span class="row">
			<Switch checked={newAutorun === 'true'} label={t.autorun} onchange={(on) => (newAutorun = on ? 'true' : 'false')} />
			<span>{newAutorun === 'true' ? t.on : t.off}</span>
		</span>
		<p class="muted note">{t.autorunNote}</p>
		{#if error}<p class="notice error" role="alert">{error}</p>{/if}
		<div class="footer">
			<button class="button primary" disabled={!newName.trim() || working}>{t.create}</button>
			<button type="button" class="button" onclick={() => leave('mode')}>{m.common.cancel}</button>
		</div>
	</form>
{/snippet}

{#if mode === 'new' && editing && !all.length}
	<div class="split"><p class="empty">{reply && !reply.readable ? t.fromCatalog : t.none}</p><aside class="pane" aria-label={t.pane} use:fitToViewport>{@render newForm()}</aside></div>
{:else if !all.length}
	<p class="empty">{reply && !reply.readable ? t.fromCatalog : t.none}</p>
	{#if editing}<p><button type="button" class="button primary" onclick={startNew}>{t.new}</button></p>{/if}
{:else if !shown.length}
	<p class="muted">{m.common.noMatch({ query })}</p>
{:else}
	{#if reply && !reply.readable}<p class="muted lead">{t.onlyNew}</p>{/if}
	<div class="split">
		<table>
			<thead>
				<tr>
					<th class="mark" aria-label={m.common.selected}></th>
					<SortHeader label={t.recording} key="name" bind:sort />
					<SortHeader label={t.in} key="mod" bind:sort class="mod" />
					<SortHeader label={t.steps} key="steps" bind:sort descendingFirst align="end" />
					<SortHeader label={t.menus} key="menus" bind:sort descendingFirst align="end" />
					<th class="caps plain">{t.autorun}</th>
					<SortHeader label={t.check} key="check" bind:sort descendingFirst />
				</tr>
			</thead>
			<tbody>
				{#each shown as r (r.name)}
					<tr aria-selected={selection.is(r)} onclick={() => !mode && selection.select(r)} ondblclick={() => open(r)}>
						<td class="mark">{#if selection.is(r)}<Diamond />{/if}</td>
						<td class="name">
							<a href={recordingHref(r.name)}>{r.name}</a>
							{#if r.title && r.title !== r.name}<span class="muted sub">{r.title}</span>{/if}
							{#if r.deleted}<span class="muted sub">{t.deletedInDraft}</span>{:else if r.unlisted}<span class="sub warn">{t.unlistedShort}</span>{:else if r.new}<span class="muted sub">{t.notSavedYet}</span>{:else if r.drafted}<span class="muted sub">{t.unsavedChanges}</span>{/if}
						</td>
						<td class="mod muted"><span use:clipped>{r.mod}</span>{#if r.losing.length}<span class="sub">{t.losing({ mods: r.losing.join(', ') })}</span>{/if}</td>
						<td class="n">{count(r.steps)}</td>
						<td class="n">{count(r.menus)}</td>
						<td class="caps small">{r.autorun ? t.on : t.off}</td>
						<td class="caps small check" class:warn={failing(r) > 0}>{countsText(r.counts)}</td>
					</tr>
				{/each}
			</tbody>
		</table>
		<aside class="pane" aria-label={t.pane} use:fitToViewport>
			{#if mode === 'new' && editing}
				{@render newForm()}
			{:else if current}
				<h2 class="caps">{current.name}</h2>
				{#if current.title && current.title !== current.name}<p class="muted">{current.title}</p>{/if}
				<dl>
					{#if current.author}<dt class="caps">{t.author}</dt><dd>{current.author}</dd>{/if}
					{#if current.version}<dt class="caps">{t.version}</dt><dd>{current.version}</dd>{/if}
					<dt class="caps">{t.autorun}</dt><dd>{current.autorun ? t.on : t.off}</dd>
					<dt class="caps">{t.in}</dt><dd>{current.mod}</dd>
					{#if current.info}<dt class="caps">{t.info}</dt><dd class="raw">{current.info}</dd>{/if}
					<dt class="caps">{t.files}</dt><dd>{count(current.files)}</dd>
					<dt class="caps">{t.steps}</dt><dd>{count(current.steps)}</dd>
				</dl>
				<div class="summary">
					{#if current.unlisted && !current.deleted}<p class="notice warn">{t.unlisted({ mod: current.mod })}</p>{/if}
					{#if current.deleted}<p class="notice warn">{t.deletedSays({ mod: current.mod })}</p>{:else if current.new}<p class="notice">{t.newNotSaved}</p>{:else if current.drafted}<p class="notice">{t.draftHeld}</p>{/if}
					{#if current.steps && failing(current) === 0 && current.counts.unchecked === 0 && current.counts.modMissing === 0}
						<p>{t.allMatch({ count: current.steps })}</p>
					{/if}
					{#if current.counts.notFound + current.counts.notAllowed}<p class="notice warn">{t.wontReplay({ count: current.counts.notFound + current.counts.notAllowed })}</p>{/if}
					{#if current.counts.modMissing}<p class="notice warn">{t.modMissing({ count: current.counts.modMissing })}</p>{/if}
					{#if current.counts.unchecked}<p class="notice">{t.unchecked({ count: current.counts.unchecked })}</p>{/if}
					{#each current.problems as p, i (i)}
						{#if p.kind === 'no-info'}<p class="notice warn">{t.noInfo}</p>{:else if p.kind === 'no-kind'}<p class="notice warn">{t.problems.noKind({ path: p.path, detail: p.detail })}</p>{:else}<p class="notice error">{t.problems.unreadable({ path: p.path, detail: p.detail })}</p>{/if}
					{/each}
					{#if current.losing.length}<p class="notice">{t.losing({ mods: current.losing.join(', ') })}</p>{/if}
				</div>
				<div class="actions">
					<a class="button primary" href={recordingHref(current.name)}>{m.common.open}</a>
					{#if editing && !current.deleted && data.status.openRecording !== current.name}
						<button type="button" class="button" onclick={() => openForAdding(current!)}>{t.openForAdding}</button>
					{/if}
					{#if editing && current.deleted}
						<button type="button" class="button" disabled={working} onclick={() => markDeleted(false)}>{t.restore}</button>
					{:else if canDelete}
						<button type="button" class="button" disabled={working} onclick={() => markDeleted(true)}>{t.delete}</button>
					{/if}
				</div>
				{#if editing && data.status.openRecording === current.name}<p class="notice">{t.isOpen}</p>{/if}
				{#if editing && whyNot(current) && current.deletes !== 'new'}<p class="muted why">{whyNot(current)}</p>{/if}
				{#if error}<p class="notice error" role="alert">{error}</p>{/if}
			{/if}
		</aside>
	</div>
{/if}

<style>
	.bar {
		display: flex;
		align-items: baseline;
		gap: var(--space-4);
		margin-bottom: var(--space-4);
		margin-right: calc(clamp(18rem, 26vw, 24rem) + var(--space-5));
	}
	h1 {
		font-size: var(--step-3);
		font-weight: 600;
		line-height: 1;
		position: relative;
		top: 0.21em;
	}
	@media (max-width: 90rem) {
		.bar {
			margin-right: 0;
		}
	}
	.bar .n {
		white-space: nowrap;
	}
	.bar input {
		margin-left: auto;
		width: 16rem;
		flex: 0 1 16rem;
		min-width: 9rem;
	}
	.empty {
		max-width: 60ch;
	}
	.split {
		display: grid;
		grid-template-columns: minmax(0, 1fr) clamp(18rem, 26vw, 24rem);
		gap: var(--space-5);
		align-items: start;
	}
	table {
		width: 100%;
		border-collapse: collapse;
	}
	th.plain {
		text-align: left;
		font-size: var(--step--1);
		color: var(--muted);
		font-weight: inherit;
		padding: var(--space-2) var(--space-3);
		border-bottom: 1px solid var(--rule-strong);
	}
	th.mark {
		border-bottom: 1px solid var(--rule-strong);
	}
	td {
		padding: var(--space-2) var(--space-3);
		border-bottom: 1px solid var(--rule);
		vertical-align: baseline;
	}
	tbody tr {
		cursor: default;
	}
	tbody tr[aria-selected]:not([aria-selected='true']):hover td {
		background: var(--glass);
	}
	tr[aria-selected='true'] td {
		background: var(--select);
	}
	.mark {
		width: 1.5rem;
		padding-right: 0;
		vertical-align: middle;
	}
	.name a {
		font-size: var(--step-1);
		text-decoration: none;
	}
	.name a:hover {
		text-decoration: underline;
	}
	.sub.warn {
		color: var(--warn);
	}
	.sub {
		display: block;
		font-size: var(--step--1);
	}
	td.mod {
		max-width: 14rem;
		font-size: var(--step--1);
	}
	td.mod > span:first-child {
		display: block;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	td.n {
		text-align: right;
		width: 5rem;
	}
	.small {
		font-size: var(--step--1);
		color: var(--muted);
		white-space: nowrap;
	}
	/* The check wraps rather than running under the pane. */
	.check {
		white-space: normal;
		min-width: 6rem;
	}
	.check.warn {
		color: var(--warn);
	}
	.pane {
		position: sticky;
		top: 0;
		display: flex;
		flex-direction: column;
		min-height: calc(100vh - 7rem);
		padding: var(--space-5);
		background: var(--panel);
		border-left: 1px solid var(--rule-strong);
		overflow-y: auto;
	}
	.pane h2 {
		font-size: var(--step-2);
		font-weight: 600;
		margin-bottom: var(--space-1);
		overflow-wrap: anywhere;
	}
	.pane dl {
		display: grid;
		grid-template-columns: 7.5rem 1fr;
		gap: var(--space-1) var(--space-3);
		margin: var(--space-4) 0;
	}
	.pane dt {
		color: var(--muted);
		font-size: var(--step--1);
		padding-top: 0.15em;
	}
	.pane dd {
		margin: 0;
		overflow-wrap: anywhere;
	}
	.raw {
		font-family: var(--font-mono);
		font-size: var(--step--1);
	}
	.summary {
		display: grid;
		gap: var(--space-2);
		margin-bottom: var(--space-4);
	}
	.summary p {
		max-width: 60ch;
	}
	.actions {
		display: flex;
		gap: var(--space-2);
	}
	.actions a {
		text-decoration: none;
	}
	.new {
		display: grid;
		gap: var(--space-2);
		justify-items: start;
	}
	.new .row {
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
	}
	.new label {
		display: grid;
		gap: var(--space-1);
		width: 100%;
	}
	.new label span,
	.label {
		color: var(--muted);
		font-size: var(--step--1);
	}
	.new .note,
	.why {
		font-size: var(--step--1);
		margin: 0;
	}
	.why {
		margin-top: var(--space-3);
	}
	/* A form's buttons stay in reach above the hint strip. */
	.footer {
		position: sticky;
		bottom: 0;
		display: flex;
		gap: var(--space-2);
		width: 100%;
		margin-top: var(--space-3);
		padding-top: var(--space-3);
		background: var(--panel);
		border-top: 1px solid var(--rule);
	}
	.lead {
		margin: 0 0 var(--space-3);
		max-width: 65ch;
	}
	.pane > .notice {
		margin-top: var(--space-3);
	}
	@media (max-width: 60rem) {
		.bar {
			margin-right: 0;
			flex-wrap: wrap;
		}
		.split {
			grid-template-columns: 1fr;
		}
		.pane {
			position: static;
			min-height: 0;
			border-left: 0;
			border-top: 1px solid var(--rule-strong);
		}
		.split :global(.mod) {
			display: none;
		}
	}
</style>
