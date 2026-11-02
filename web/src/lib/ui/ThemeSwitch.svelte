<script lang="ts">
	import { ApiError, request } from '#lib/api/client.ts';
	import type { Status, ThemeRequest } from '#lib/api/types/api.ts';
	import { m } from '#lib/copy/messages.ts';

	let { theme }: { theme: string } = $props();

	// Follows what was saved, and changes at once when chosen; the page's
	// data-theme does the rest.
	let current = $derived(theme);
	let error = $state('');

	const choices = [
		{ value: 'system', label: m.shell.theme.auto, title: m.shell.theme.autoTitle },
		{ value: 'dark', label: m.shell.theme.dark, title: m.shell.theme.dark },
		{ value: 'light', label: m.shell.theme.light, title: m.shell.theme.light }
	];

	async function choose(value: string) {
		const before = current;
		current = value;
		document.documentElement.dataset.theme = value;
		error = '';
		try {
			await request<Status>('PUT', '/api/theme', { theme: value } satisfies ThemeRequest);
		} catch (e) {
			current = before;
			document.documentElement.dataset.theme = before;
			error = e instanceof ApiError ? e.message : String(e);
		}
	}
</script>

<div class="switch caps" role="radiogroup" aria-label={m.shell.theme.label} title={error || undefined}>
	{#each choices as c (c.value)}
		<button
			type="button"
			role="radio"
			aria-checked={current === c.value}
			title={c.title}
			onclick={() => choose(c.value)}>{c.label}</button
		>
	{/each}
</div>

<style>
	.switch {
		display: flex;
		border: 1px solid var(--rule-strong);
		border-radius: var(--radius);
		overflow: hidden;
	}
	button {
		background: none;
		border: 0;
		padding: 0 var(--space-2);
		color: var(--muted);
		font-size: var(--step--1);
		letter-spacing: var(--track);
		cursor: pointer;
	}
	button[aria-checked='true'] {
		background: var(--accent);
		color: var(--on-accent);
		font-weight: 600;
	}
	button:hover:not([aria-checked='true']) {
		color: var(--ink);
	}
</style>
