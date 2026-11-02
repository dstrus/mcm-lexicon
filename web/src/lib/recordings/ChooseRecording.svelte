<!-- Which recording new steps go to, asked when none is chosen: the
     recordings found, and New recording…, named in place. The choice
     becomes the top bar's. choose() opens it and says what was chosen. -->
<script lang="ts">
	import { invalidateAll } from '$app/navigation';
	import { page } from '$app/state';
	import { ApiError } from '#lib/api/client.ts';
	import type { ListReply } from '#lib/api/types/recordings.ts';
	import { m } from '#lib/copy/messages.ts';
	import { create, setOpen } from '#lib/recordings/api.ts';
	import Diamond from '#lib/ui/Diamond.svelte';
	import { tick } from 'svelte';

	const t = m.adding;

	let dialog = $state<HTMLDialogElement>();
	let at = $state(0);
	let naming = $state(false);
	let only = $state(false); // opened for a new recording only: no list to choose from
	let newName = $state('');
	let nameField = $state<HTMLInputElement>();
	let error = $state('');
	let names = $derived(((page.data.recordings as ListReply | null)?.recordings ?? []).map((r) => r.name));
	let rows = $derived([...names, '']); // '' is New recording…
	let settle: ((name: string | null) => void) | null = null;

	// choose opens the dialog; it resolves to the recording chosen, now the
	// open one, or null when the dialog is closed without a choice. With
	// fresh, it goes straight to naming a new recording.
	export async function choose(fresh = false): Promise<string | null> {
		at = names.length;
		naming = fresh;
		only = fresh;
		newName = '';
		error = '';
		dialog?.showModal();
		const chosen = new Promise<string | null>((resolve) => (settle = resolve));
		if (fresh) {
			await tick();
			nameField?.focus();
		} else {
			at = 0;
		}
		return chosen;
	}

	async function pick(name: string) {
		error = '';
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
			await invalidateAll();
			const done = settle;
			settle = null;
			dialog?.close();
			done?.(name);
		} catch (err) {
			error = err instanceof ApiError ? err.message : String(err);
		}
	}

	function closed() {
		settle?.(null);
		settle = null;
	}

	function keys(e: KeyboardEvent) {
		if (naming) return;
		if (e.key === 'ArrowDown') at = Math.min(rows.length - 1, at + 1);
		else if (e.key === 'ArrowUp') at = Math.max(0, at - 1);
		else if (e.key === 'Enter') pick(rows[at]!);
		else return;
		e.preventDefault();
	}
</script>

<dialog bind:this={dialog} aria-label={only ? t.newTitle : t.choose} onclick={(e) => e.target === dialog && dialog?.close()} onkeydown={keys} onclose={closed}>
	<h2 class="caps">{only ? t.newTitle : t.choose}</h2>
	<p class="muted">{only ? t.newLead : t.chooseLead}</p>
	{#if !only}<ul role="listbox" aria-label={t.choose}>
		{#each rows as name, i (name)}
			<li role="option" aria-selected={i === at}>
				<button type="button" onclick={() => pick(name)} onmouseenter={() => (at = i)}>
					<span class="mark">{#if i === at}<Diamond />{/if}</span>
					<span>{name || t.newRecording}</span>
				</button>
			</li>
		{/each}
	</ul>{/if}
	{#if naming}
		<form class="name" class:alone={only} onsubmit={(e) => (e.preventDefault(), pick(''))}>
			<label>
				<span class="caps">{t.name}</span>
				<input bind:this={nameField} bind:value={newName} required autocomplete="off" spellcheck="false" />
			</label>
			<p class="muted small">{t.nameNote}</p>
			<button class="button primary">{t.create}</button>
		</form>
	{/if}
	{#if error}<p class="notice error" role="alert">{error}</p>{/if}
</dialog>

<style>
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
	h2 {
		font-size: var(--step-1);
		font-weight: 600;
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
	/* With no list above it, the name needs no rule to set it apart. */
	.name.alone {
		padding-top: 0;
		border-top: 0;
	}
	.name label {
		display: grid;
		gap: var(--space-1);
	}
	.name label span {
		color: var(--muted);
		font-size: var(--step--1);
	}
	.small {
		font-size: var(--step--1);
		margin: 0;
	}
	.name .button {
		justify-self: start;
	}
	.notice {
		margin-top: var(--space-3);
	}
</style>
