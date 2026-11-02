<!-- A recording's info file: the name MCM Recorder shows, its author and
     version, and whether it plays by itself. While editing, each field goes
     into the draft as it's changed, like every other change; read-only, it
     shows what's on disk. -->
<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { ApiError } from '#lib/api/client.ts';
	import type { Status } from '#lib/api/types/api.ts';
	import type { Draft } from '#lib/api/types/recordings.ts';
	import { m } from '#lib/copy/messages.ts';
	import { change } from '#lib/recordings/draft.ts';
	import Segmented from '#lib/ui/Segmented.svelte';
	import { useShortcuts } from '#lib/ui/shortcuts.svelte.ts';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const t = m.info;

	let ed = $derived(data.editor);
	let d = $derived(ed.draft);
	let canEdit = $derived((page.data.status as Status).editing && !ed.deleted);
	let working = $state(false);
	let error = $state('');

	// The fields as typed, started again from the draft whenever it changes.
	let title = $state('');
	let author = $state('');
	let version = $state('');
	$effect(() => {
		title = d.title || d.name;
		author = d.author;
		version = d.version;
	});

	async function apply(edit: (x: Draft) => void) {
		working = true;
		error = '';
		try {
			await change(ed, edit);
		} catch (e) {
			error = e instanceof ApiError ? e.message : String(e);
		} finally {
			working = false;
		}
	}
	// A text field goes into the draft when it's left or Enter is pressed;
	// Esc puts it back.
	function commit(field: 'title' | 'author' | 'version', value: string) {
		const v = value.trim();
		const was = field === 'title' ? d.title || d.name : d[field];
		if (v === was || (field === 'title' && !v)) {
			if (field === 'title') title = was;
			return;
		}
		apply((x) => (x[field] = v));
	}
	function keys(e: KeyboardEvent, field: 'title' | 'author' | 'version') {
		if (e.key !== 'Escape') return;
		if (field === 'title') title = d.title || d.name;
		else if (field === 'author') author = d.author;
		else version = d.version;
	}

	useShortcuts(() => [{ key: 'Escape', shown: 'Esc', label: m.recording.up, run: () => goto(`/recordings?item=${encodeURIComponent(d.name)}`) }]);
</script>

<div class="info">
	<div class="field">
		<label for="title" class="caps">{t.name}</label>
		{#if canEdit}
			<input id="title" bind:value={title} disabled={working} onchange={() => commit('title', title)} onkeydown={(e) => keys(e, 'title')} autocomplete="off" spellcheck="false" />
		{:else}
			<p id="title">{d.title || d.name}</p>
		{/if}
		<p class="note">{t.nameNote({ file: `${d.name}.json` })}</p>
	</div>
	<div class="field">
		<label for="author" class="caps">{t.author}</label>
		{#if canEdit}
			<input id="author" bind:value={author} disabled={working} onchange={() => commit('author', author)} onkeydown={(e) => keys(e, 'author')} autocomplete="off" spellcheck="false" />
		{:else}
			<p id="author">{d.author || t.none}</p>
		{/if}
	</div>
	<div class="field">
		<label for="version" class="caps">{t.version}</label>
		{#if canEdit}
			<input id="version" class="short" bind:value={version} disabled={working} onchange={() => commit('version', version)} onkeydown={(e) => keys(e, 'version')} autocomplete="off" spellcheck="false" />
		{:else}
			<p id="version">{d.version || t.none}</p>
		{/if}
	</div>
	<div class="field">
		<span class="caps label">{t.autorun}</span>
		{#if canEdit}
			<Segmented
				value={d.autorun ? 'on' : 'off'}
				options={[
					{ value: 'off', label: m.recordings.off },
					{ value: 'on', label: m.recordings.on }
				]}
				label={t.autorun}
				onchange={(v) => apply((x) => (x.autorun = v === 'on'))}
			/>
		{:else}
			<p>{d.autorun ? m.recordings.on : m.recordings.off}</p>
		{/if}
		<p class="note">{t.autorunNote}</p>
	</div>
	{#if canEdit}<p class="note kept">{t.kept}</p>{/if}
	{#if error}<p class="notice error" role="alert">{error}</p>{/if}
</div>

<style>
	.info {
		display: grid;
		gap: var(--space-5);
		max-width: 36rem;
		margin-top: var(--space-5);
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
	input.short {
		width: 12rem;
	}
	p {
		margin: 0;
	}
	.note {
		font-size: var(--step--1);
		color: var(--muted);
		max-width: 60ch;
	}
	.kept {
		margin-top: calc(-1 * var(--space-2));
	}
</style>
