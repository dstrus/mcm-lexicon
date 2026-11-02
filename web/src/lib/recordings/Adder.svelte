<!-- Adding the selected option to the open recording, from a menu's page or
     a search result: A, or the pane's button, opens the add form in the
     pane (?mode=add); with no recording open, it first asks which one.
     Without the form, the pane shows its usual contents (children). -->
<script lang="ts">
	import { invalidate, invalidateAll } from '$app/navigation';
	import { page } from '$app/state';
	import { ApiError } from '#lib/api/client.ts';
	import type { Status } from '#lib/api/types/api.ts';
	import type { Option } from '#lib/api/types/catalog.ts';
	import type { ListReply } from '#lib/api/types/recordings.ts';
	import { m } from '#lib/copy/messages.ts';
	import { addStep, create, recordingHref, recordingsDependency, setOpen } from '#lib/recordings/api.ts';
	import { stepFor } from '#lib/recordings/steps.ts';
	import ValueEditor from '#lib/recordings/ValueEditor.svelte';
	import Diamond from '#lib/ui/Diamond.svelte';
	import { leave, param, setParams } from '#lib/ui/address.svelte.ts';
	import { useShortcuts } from '#lib/ui/shortcuts.svelte.ts';
	import { tick, type Snippet } from 'svelte';

	let {
		option,
		menu,
		pageName,
		children
	}: {
		option: Option;
		menu: string; // the menu's raw name, as a recording calls it
		pageName: string; // raw; "" for a menu's default page
		children: Snippet;
	} = $props();

	const t = m.adding;

	let status = $derived(page.data.status as Status);
	let list = $derived(page.data.recordings as ListReply | null);
	let openName = $derived(status?.openRecording ?? '');
	// Adding is a change, so it's there only while editing.
	let editing = $derived(!!status?.editing);
	let adding = $derived(editing && param('mode') === 'add' && !!openName);

	// The step being added, started from the option's default. It's made
	// again whenever another option is selected.
	let step = $state(stepFor('', '', { type: 'toggle' } as Option));
	let problem = $state('');
	let error = $state('');
	let added = $state('');
	let working = $state(false);
	$effect(() => {
		step = stepFor(menu, pageName, option);
		error = '';
	});
	$effect(() => {
		void option;
		added = '';
	});

	function start() {
		error = '';
		if (!openName) {
			openDialog();
			return;
		}
		setParams({ mode: 'add' });
	}

	async function add(e?: SubmitEvent) {
		e?.preventDefault();
		if (problem || working) return;
		working = true;
		error = '';
		try {
			const r = await addStep(openName, step);
			added = t.added({ recording: openName, n: r.number });
			await leave('mode');
			invalidate(recordingsDependency);
		} catch (err) {
			error = err instanceof ApiError ? err.message : String(err);
		} finally {
			working = false;
		}
	}

	// Choosing the recording, when none is open.
	let dialog = $state<HTMLDialogElement>();
	let at = $state(0);
	let naming = $state(false);
	let newName = $state('');
	let nameField = $state<HTMLInputElement>();
	let dialogError = $state('');
	let names = $derived((list?.recordings ?? []).map((r) => r.name));
	let rows = $derived([...names, '']); // '' is New recording…

	function openDialog() {
		at = 0;
		naming = false;
		newName = '';
		dialogError = '';
		dialog?.showModal();
	}

	async function choose(name: string) {
		dialogError = '';
		try {
			if (name === '') {
				if (!naming) {
					naming = true;
					await tick();
					nameField?.focus();
					return;
				}
				const ed = await create({ name: newName, author: '', autorun: false });
				name = ed.draft.name;
			}
			await setOpen(name);
			dialog?.close();
			await invalidateAll();
			setParams({ mode: 'add' });
		} catch (err) {
			dialogError = err instanceof ApiError ? err.message : String(err);
		}
	}

	function keys(e: KeyboardEvent) {
		if (naming) return;
		if (e.key === 'ArrowDown') at = Math.min(rows.length - 1, at + 1);
		else if (e.key === 'ArrowUp') at = Math.max(0, at - 1);
		else if (e.key === 'Enter') choose(rows[at]!);
		else return;
		e.preventDefault();
	}

	useShortcuts(() =>
		!editing
			? []
			: adding
			? [
					{ key: 'Enter', label: t.add, command: true, disabled: !!problem || working, run: () => add() },
					{ key: 'Escape', shown: 'Esc', label: m.common.cancel, run: () => leave('mode') }
				]
			: [{ key: 'a', shown: 'A', label: openName ? t.addTo({ recording: openName }) : t.addToAny, run: start }]
	);
</script>

{#if adding}
	<form class="add" onsubmit={add}>
		<h2 class="caps">{option.text || m.values.noLabel}</h2>
		<p class="muted">{t.into({ recording: openName })}</p>
		<ValueEditor kind={step.kind} {option} bind:value={step.value} bind:side={step.side} bind:problem />
		{#if error}<p class="notice error" role="alert">{error}</p>{/if}
		<div class="footer">
			<button class="button primary" disabled={!!problem || working}>{t.add}</button>
			<button type="button" class="button" onclick={() => leave('mode')}>{m.common.cancel}</button>
		</div>
	</form>
{:else}
	{@render children()}
	{#if editing}
		{#if added}<p class="notice" role="status">{added} <a href={recordingHref(openName)}>{t.openIt}</a></p>{/if}
		<div class="actions">
			<button type="button" class="button primary" onclick={start}>{openName ? t.addTo({ recording: openName }) : t.addToAny}</button>
		</div>
	{/if}
{/if}

<dialog bind:this={dialog} aria-label={t.choose} onclick={(e) => e.target === dialog && dialog?.close()} onkeydown={keys}>
	<h2 class="caps">{t.choose}</h2>
	<p class="muted">{t.chooseLead}</p>
	<ul role="listbox" aria-label={t.choose}>
		{#each rows as name, i (name)}
			<li role="option" aria-selected={i === at}>
				<button type="button" onclick={() => choose(name)} onmouseenter={() => (at = i)}>
					<span class="mark">{#if i === at}<Diamond />{/if}</span>
					<span>{name || t.newRecording}</span>
				</button>
			</li>
		{/each}
	</ul>
	{#if naming}
		<form class="name" onsubmit={(e) => (e.preventDefault(), choose(''))}>
			<label>
				<span class="caps">{t.name}</span>
				<input bind:this={nameField} bind:value={newName} required autocomplete="off" spellcheck="false" />
			</label>
			<p class="muted small">{t.nameNote}</p>
			<button class="button primary">{t.create}</button>
		</form>
	{/if}
	{#if dialogError}<p class="notice error" role="alert">{dialogError}</p>{/if}
</dialog>

<style>
	h2 {
		font-size: var(--step-2);
		font-weight: 600;
		margin-bottom: var(--space-1);
		overflow-wrap: anywhere;
	}
	.add {
		display: flex;
		flex-direction: column;
		gap: var(--space-3);
		min-height: 100%;
	}
	.add > p {
		margin: 0;
	}
	/* The form's buttons stay in reach above the hint strip. */
	.footer {
		position: sticky;
		bottom: 0;
		margin-top: auto;
		display: flex;
		gap: var(--space-2);
		padding-top: var(--space-3);
		background: var(--panel);
		border-top: 1px solid var(--rule);
	}
	.actions {
		display: flex;
		gap: var(--space-2);
		margin-top: var(--space-4);
	}
	.notice {
		margin-top: var(--space-3);
		max-width: 60ch;
	}
	dialog {
		width: min(28rem, 90vw);
		margin: 18vh auto auto;
		padding: var(--space-4);
		background: var(--smoke);
		color: var(--ink);
		border: 1px solid var(--rule-strong);
		border-radius: var(--radius);
		box-shadow: 0 1.5rem 3rem rgb(0 0 0 / 0.45);
	}
	dialog::backdrop {
		background: rgb(0 0 0 / 0.45);
	}
	dialog h2 {
		font-size: var(--step-1);
	}
	ul {
		list-style: none;
		margin: var(--space-3) 0 0;
		padding: 0;
		max-height: 50vh;
		overflow-y: auto;
	}
	li button {
		display: flex;
		align-items: baseline;
		gap: var(--space-2);
		width: 100%;
		background: none;
		border: 0;
		padding: var(--space-2) var(--space-1);
		text-align: left;
		cursor: pointer;
	}
	[aria-selected='true'] button {
		background: var(--select);
	}
	[aria-selected='false'] button:hover {
		background: var(--glass);
	}
	.mark {
		width: 1rem;
		flex: none;
		display: inline-flex;
		align-self: center;
	}
	.name {
		display: grid;
		gap: var(--space-2);
		margin-top: var(--space-3);
		padding-top: var(--space-3);
		border-top: 1px solid var(--rule);
	}
	.name label {
		display: grid;
		gap: var(--space-1);
	}
	.name label span {
		color: var(--muted);
		font-size: var(--step--1);
	}
	.name .small {
		font-size: var(--step--1);
		margin: 0;
	}
	.name .button {
		justify-self: start;
	}
</style>
