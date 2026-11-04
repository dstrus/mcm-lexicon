<!-- MCM Lexicon's own settings: the mod folder recordings are saved to in
     an MO2 instance. Each change is kept as it's made, editing or not,
     since it changes no recording. -->
<script lang="ts">
	import { invalidateAll } from '$app/navigation';
	import { page } from '$app/state';
	import { ApiError } from '#lib/api/client.ts';
	import { setSaveFolder } from '#lib/api/status.ts';
	import type { Status } from '#lib/api/types/api.ts';
	import Fill from '#lib/copy/Fill.svelte';
	import { m } from '#lib/copy/messages.ts';
	import Switch from '#lib/ui/Switch.svelte';

	const t = m.settings;

	let status = $derived(page.data.status as Status);
	let folder = $derived(status.saveFolder);
	let working = $state(false);
	let error = $state('');

	// The name as typed, started again from the settings whenever they change.
	let name = $state('');
	$effect(() => {
		name = folder.name;
	});

	async function save(nextName: string, noDelete: boolean) {
		working = true;
		error = '';
		try {
			await setSaveFolder(nextName, noDelete);
			await invalidateAll();
		} catch (e) {
			error = e instanceof ApiError ? e.message : String(e);
			name = folder.name;
		} finally {
			working = false;
		}
	}
	// The name is kept when the field is left or Enter is pressed; Esc puts
	// it back.
	function commit() {
		const v = name.trim();
		if (v === folder.name) return;
		save(v, folder.noDelete);
	}
	function keys(e: KeyboardEvent) {
		if (e.key === 'Escape') name = folder.name;
	}
</script>

<h1 class="caps">{t.title}</h1>

<section class="settings" aria-labelledby="save-folder">
	<h2 id="save-folder" class="caps">{t.saveFolder}</h2>
	<div class="field">
		<label for="name" class="caps">{t.name}</label>
		<input id="name" bind:value={name} disabled={working} onchange={commit} onkeydown={keys} autocomplete="off" spellcheck="false" />
	</div>
	<div class="field">
		<span class="caps label">{t.noDelete}</span>
		<span class="row">
			<Switch checked={folder.noDelete} label={t.noDelete} disabled={working} onchange={(on) => save(name.trim(), on)} />
			<span>{folder.noDelete ? m.recordings.on : m.recordings.off}</span>
		</span>
		<p class="note">{t.noDeleteNote}</p>
	</div>
	<p class="result"><Fill text={t.folder.text}>{#snippet folder()}<span class="raw">{status.saveFolder.folder}</span>{/snippet}</Fill></p>
	<p class="note">{t.otherMods}</p>
	<p class="note">{t.moved}</p>
	{#if status.source?.kind === 'data'}<p class="notice">{t.dataFolder}</p>
	{:else if status.catalogFile}<p class="notice">{t.catalogFile}</p>{/if}
	{#if error}<p class="notice error" role="alert">{error}</p>{/if}
</section>

<style>
	h1 {
		font-size: var(--step-3);
		font-weight: 600;
		line-height: 1;
		position: relative;
		top: 0.21em;
	}
	h2 {
		font-size: var(--step-1);
		font-weight: 600;
		color: var(--accent);
	}
	.settings {
		display: grid;
		gap: var(--space-4);
		max-width: 36rem;
		margin-top: var(--space-6);
	}
	.field {
		display: grid;
		gap: var(--space-1);
		justify-items: start;
	}
	label,
	.label {
		font-size: var(--step--1);
		color: var(--muted);
	}
	input {
		width: 100%;
	}
	p {
		margin: 0;
	}
	.row {
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
	}
	.note {
		font-size: var(--step--1);
		color: var(--muted);
		max-width: 60ch;
	}
	.raw {
		font-family: var(--font-mono);
		font-size: var(--step--1);
	}
</style>
