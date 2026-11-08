<!-- Saving a copy of what was read to a file: a folder, chosen in the
     picker or typed, and the file's name, suggested and editable. A file
     already there is replaced only after asking, here, with Replace
     beside the question. Afterward a line says what was written. -->
<script lang="ts">
	import { ApiError } from '#lib/api/client.ts';
	import { browseFolders } from '#lib/api/status.ts';
	import type { ExportReply, ExportRequest } from '#lib/api/types/api.ts';
	import Fill from '#lib/copy/Fill.svelte';
	import { m } from '#lib/copy/messages.ts';
	import Path from '#lib/ui/Path.svelte';
	import PathPicker from './PathPicker.svelte';

	let {
		name: suggested,
		ext,
		action,
		working: workingLabel,
		save,
		done
	}: {
		name: string; // the file name suggested, without its extension
		ext: string; // the extension the server adds when it's missing: ".json"
		action: string; // the button's label
		working: string; // the button's label while it saves
		save: (req: ExportRequest) => Promise<ExportReply>;
		done: (r: ExportReply) => string; // what was written, after the path
	} = $props();

	const t = m.settings.copy;
	const id = $props.id();

	// The folder starts where MCM Lexicon runs from, as the picker does.
	let folder = $state('');
	browseFolders('', 'folder')
		.then((l) => (folder ||= l.path))
		.catch(() => {});
	// svelte-ignore state_referenced_locally
	let name = $state(suggested);
	let browsing = $state(false);
	let working = $state(false);
	let error = $state('');
	let exists = $state(''); // the name of the file already there, waiting on Replace
	let result = $state<ExportReply>();

	async function run(replace = false) {
		working = true;
		error = '';
		exists = '';
		result = undefined;
		try {
			result = await save({ folder: folder.trim(), name: name.trim(), replace });
		} catch (e) {
			if (e instanceof ApiError && e.status === 409) exists = name.trim().toLowerCase().endsWith(ext) ? name.trim() : name.trim() + ext;
			else error = e instanceof ApiError ? e.message : String(e);
		} finally {
			working = false;
		}
	}
	function submit(e: SubmitEvent) {
		e.preventDefault();
		run();
	}
	function endBrowse(path?: string) {
		browsing = false;
		if (path) folder = path;
	}
</script>

<form onsubmit={submit}>
	<div class="field">
		<label class="caps" for="{id}-folder">{t.folder}</label>
		<div class="entry">
			<input id="{id}-folder" bind:value={folder} required autocomplete="off" spellcheck="false" />
			<button type="button" class="button" aria-expanded={browsing} onclick={() => (browsing = true)}>{m.source.picker.browse}</button>
		</div>
		{#if browsing}<PathPicker looking="folder" start={folder} onpick={endBrowse} oncancel={() => endBrowse()} />{/if}
	</div>
	<div class="field">
		<label class="caps" for="{id}-name">{t.name}</label>
		<input id="{id}-name" bind:value={name} required autocomplete="off" spellcheck="false" />
	</div>
	<div class="buttons">
		<button class="button" disabled={working}>{working ? workingLabel : action}</button>
	</div>
	{#if exists}
		<div class="notice warn ask" role="alert">
			<span>{t.exists({ file: exists })}</span>
			<span class="buttons">
				<button type="button" class="button" disabled={working} onclick={() => run(true)}>{t.replace}</button>
				<button type="button" class="button" disabled={working} onclick={() => (exists = '')}>{m.common.cancel}</button>
			</span>
		</div>
	{/if}
	{#if error}<p class="notice error" role="alert">{error}</p>{/if}
	{#if result}
		{@const r = result}
		<p class="result" role="status"><Fill text={t.saved.text}>{#snippet path()}<span class="raw"><Path text={r.path} /></span>{/snippet}</Fill>{' '}{done(r)}</p>
	{/if}
</form>

<style>
	form {
		display: grid;
		gap: var(--space-4);
	}
	.field {
		display: grid;
		gap: var(--space-1);
	}
	label {
		font-size: var(--step--1);
		color: var(--muted);
	}
	.entry {
		display: flex;
		gap: var(--space-3);
	}
	.entry input {
		flex: 1;
		min-width: 0;
	}
	.buttons {
		display: flex;
		gap: var(--space-3);
	}
	.ask {
		display: grid;
		gap: var(--space-3);
		justify-items: start;
	}
	.ask .button {
		color: var(--ink);
	}
	.result {
		margin: 0;
	}
	.raw {
		font-family: var(--font-mono);
		font-size: 0.9em;
	}
</style>
