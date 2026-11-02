<!-- / from any page: find a menu by name and open it. -->
<script lang="ts">
	import { goto } from '$app/navigation';
	import type { Summary } from '#lib/api/types/catalog.ts';
	import { m } from '#lib/copy/messages.ts';
	import { menuHref } from '#lib/menus/api.ts';
	import Diamond from './Diamond.svelte';
	import { useShortcuts } from './shortcuts.svelte.ts';

	let { menus }: { menus: Summary[] } = $props();

	let dialog: HTMLDialogElement;
	let input = $state<HTMLInputElement>();
	let query = $state('');
	let at = $state(0);

	let matches = $derived.by(() => {
		const q = query.trim().toLowerCase();
		const all = q ? menus.filter((s) => s.nameText.toLowerCase().includes(q) || s.mod.toLowerCase().includes(q)) : menus;
		return all.slice(0, 12);
	});

	function open() {
		query = '';
		at = 0;
		dialog.showModal();
		input?.focus();
	}

	function go(s: Summary | undefined) {
		if (!s) return;
		dialog.close();
		goto(menuHref(s.id));
	}

	function keys(e: KeyboardEvent) {
		if (e.key === 'ArrowDown') at = Math.min(matches.length - 1, at + 1);
		else if (e.key === 'ArrowUp') at = Math.max(0, at - 1);
		else if (e.key === 'Enter') go(matches[at]);
		else return;
		e.preventDefault();
	}

	useShortcuts(() => (menus.length ? [{ key: '/', label: m.find.shortcut, run: open }] : []));
</script>

<dialog bind:this={dialog} aria-label={m.find.title} onclick={(e) => e.target === dialog && dialog.close()}>
	<input
		bind:this={input}
		bind:value={query}
		oninput={() => (at = 0)}
		onkeydown={keys}
		placeholder={m.find.title}
		aria-label={m.find.input}
		spellcheck="false"
		autocomplete="off"
	/>
	{#if matches.length}
		<ul role="listbox" aria-label={m.find.matches}>
			{#each matches as s, i (s.id)}
				<li role="option" aria-selected={i === at}>
					<button type="button" onclick={() => go(s)} onmouseenter={() => (at = i)}>
						<span class="mark">{#if i === at}<Diamond />{/if}</span>
						<span class="name">{s.nameText}</span>
						<span class="mod muted">{s.mod}</span>
					</button>
				</li>
			{/each}
		</ul>
	{:else}
		<p class="muted">{m.common.noMatch({ query })}</p>
	{/if}
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
	input {
		width: 100%;
	}
	ul {
		list-style: none;
		margin: var(--space-3) 0 0;
		padding: 0;
	}
	button {
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
		flex: none;
		max-width: 60%;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.mod {
		font-size: var(--step--1);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	p {
		margin-top: var(--space-3);
	}
</style>
