<script lang="ts">
	import '#lib/ui/theme.css';
	import { invalidate, invalidateAll } from '$app/navigation';
	import { page } from '$app/state';
	import { ApiError } from '#lib/api/client.ts';
	import { live, watchStatus } from '#lib/api/live.svelte.ts';
	import { menusDependency } from '#lib/menus/api.ts';
	import { recordingHref, recordingsDependency, setOpen } from '#lib/recordings/api.ts';
	import QuickFind from '#lib/ui/QuickFind.svelte';
	import { setEditing, setSource } from '#lib/api/status.ts';
	import Segmented from '#lib/ui/Segmented.svelte';
	import { m } from '#lib/copy/messages.ts';
	import SourceForm from '#lib/source/SourceForm.svelte';
	import HintBar from '#lib/ui/HintBar.svelte';
	import ThemeSwitch from '#lib/ui/ThemeSwitch.svelte';
	import { clipped } from '#lib/ui/clipped.ts';
	import { fitToViewport } from '#lib/ui/fit.ts';
	import { handleKey } from '#lib/ui/shortcuts.svelte.ts';
	import type { LayoutProps } from './$types';

	let { data, children }: LayoutProps = $props();

	$effect(() =>
		watchStatus(data.status, () => {
			invalidate(menusDependency);
			invalidate(recordingsDependency);
		})
	);

	let source = $derived(data.status.source);
	let catalog = $derived(data.status.catalogFile);
	// Something is open: a list, or a catalog file in its place.
	let open = $derived(!!source || !!catalog);
	let profileError = $state('');

	// Choosing a profile reads the same instance again under it.
	async function chooseProfile(profile: string) {
		if (!source) return;
		profileError = '';
		try {
			await setSource({ path: source.path, profile });
			await invalidateAll();
		} catch (e) {
			profileError = e instanceof ApiError ? e.message : String(e);
		}
	}

	// A catalog by its file name; the path is in its tip.
	const fileName = (path: string) => path.split(/[\\/]/).pop() ?? path;
	// A section is current at its URL or below it.
	const here = (path: string) => page.url.pathname === path || page.url.pathname.startsWith(`${path}/`);
	let recordings = $derived(data.recordings?.recordings ?? []);
	let openRecording = $derived(data.status.openRecording);
	let openError = $state('');

	// Choosing the recording new steps go to; MCM Lexicon remembers it.
	async function chooseRecording(name: string) {
		openError = '';
		try {
			await setOpen(name);
			await invalidateAll();
		} catch (e) {
			openError = e instanceof ApiError ? e.message : String(e);
		}
	}

	// Editing: off, everything is read-only; on, every change goes to the
	// draft until it's saved from Changes. Leaving with changes asks first.
	let editing = $derived(data.status.editing);
	let pending = $derived(data.status.changes);
	let leaveDialog = $state<HTMLDialogElement>();
	// The switch shows the mode as it is: a choice the dialog holds back
	// goes back to it when the dialog closes.
	let modeValue = $state('read');
	function syncMode() {
		modeValue = editing ? 'editing' : 'read';
	}
	$effect(syncMode);
	let modeError = $state('');
	async function chooseMode(value: string) {
		modeError = '';
		const on = value === 'editing';
		if (!on && pending > 0) {
			leaveDialog?.showModal();
			return;
		}
		try {
			await setEditing(on);
			await invalidateAll();
		} catch (e) {
			modeError = e instanceof ApiError ? e.message : String(e);
		}
	}
	async function discardAndLeave() {
		try {
			await setEditing(false, true);
			leaveDialog?.close();
			await invalidateAll();
		} catch (e) {
			modeError = e instanceof ApiError ? e.message : String(e);
		}
	}

	// When the catalog was scanned: the time today, the date before that.
	function scannedLabel(iso: string): string {
		const when = new Date(iso);
		const today = new Date().toDateString() === when.toDateString();
		return m.shell.scannedAt({ when: today ? when.toLocaleTimeString(undefined, { hour: 'numeric', minute: '2-digit' }) : when.toLocaleDateString(undefined, { dateStyle: 'medium' }) });
	}
</script>

<svelte:window onkeydown={handleKey} />

<div class="app" class:withsections={open}>
	<header class="top">
		<a class="name caps" href="/">{m.shell.name}</a>
		<div class="state">
			{#if open}
				<span class="mode" title={modeError || (editing ? m.shell.editingTitle : m.shell.readOnlyTitle)}>
					<Segmented bind:value={modeValue} options={[{ value: 'read', label: m.shell.readOnly }, { value: 'editing', label: m.shell.editing }]} label={m.shell.mode} onchange={chooseMode} />
				</span>
			{/if}
			{#if editing && recordings.length}
				<label class="source">
					<span class="caps muted">{m.shell.recording}</span>
					<select value={openRecording} title={openError || m.shell.recordingTitle} onchange={(e) => chooseRecording(e.currentTarget.value)}>
						<option value="">{m.shell.noRecording}</option>
						{#each recordings as r (r.name)}
							<option value={r.name}>{r.name}</option>
						{/each}
						{#if openRecording && !recordings.some((r) => r.name === openRecording)}<option value={openRecording}>{openRecording}</option>{/if}
					</select>
				</label>
			{/if}
			{#if source?.kind === 'mo2'}
				<label class="source">
					<span class="caps muted">{m.shell.profile}</span>
					<select value={source.profile} title={profileError || source.path} onchange={(e) => chooseProfile(e.currentTarget.value)}>
						{#each source.profiles as p (p)}
							<option value={p}>{p}</option>
						{/each}
					</select>
				</label>
			{:else if catalog}
				<span class="source">
					<span class="caps muted">{m.shell.catalog}</span>
					<span class="file" title={catalog} use:clipped>{fileName(catalog)}</span>
				</span>
			{/if}
			{#if open && live.scan.scanning}
				<span class="scan caps" role="status">{m.shell.scanning}</span>
			{:else if open && live.scan.scannedAt}
				<span class="scan caps">{scannedLabel(live.scan.scannedAt)}</span>
			{/if}
			<span class="game caps" class:running={live.gameRunning} role="status">
				{live.gameRunning ? m.shell.gameRunning : m.shell.gameClosed}
			</span>
			<ThemeSwitch theme={data.status.theme} />
		</div>
	</header>

	{#if !open}
		<main class="solo"><SourceForm /></main>
	{:else}
		<nav class="sections" aria-label={m.nav.sections} use:fitToViewport>
			<ul>
				<li><a class="caps" href="/menus" aria-current={here('/menus') ? 'page' : undefined}>{m.nav.menus}</a></li>
				<li><a class="caps" href="/search" aria-current={here('/search') ? 'page' : undefined}>{m.nav.search}</a></li>
				<li><a class="caps" href="/recordings" aria-current={page.url.pathname === '/recordings' ? 'page' : undefined}>{m.nav.recordings}</a></li>
				<li><a class="caps" href="/compare" aria-current={here('/compare') ? 'page' : undefined}>{m.nav.compare}</a></li>
				{#if editing}<li><a class="caps" href="/changes" title={m.nav.changesTitle({ count: pending })} aria-current={here('/changes') ? 'page' : undefined}>{m.nav.changes}{#if pending}<span class="count">{pending}</span>{/if}</a></li>{/if}
			</ul>
			{#if recordings.length}
				<p class="group caps">{m.nav.recordingsGroup}</p>
				<ul aria-label={m.nav.recordingsGroup}>
					{#each recordings as r (r.name)}
						<li><a class="caps clip" href={recordingHref(r.name)} title={r.name} aria-current={here(recordingHref(r.name)) ? 'page' : undefined}>{r.name}</a></li>
					{/each}
				</ul>
			{/if}
		</nav>
		<main>{@render children()}</main>
		<QuickFind menus={data.menus?.menus ?? []} />
		<dialog bind:this={leaveDialog} class="leave" aria-label={m.shell.leaveTitle} onclick={(e) => e.target === leaveDialog && leaveDialog?.close()} onclose={syncMode}>
			<h2 class="caps">{m.shell.leaveTitle}</h2>
			<p>{m.shell.leaveSays({ count: pending })}</p>
			<div class="buttons">
				<a class="button primary" href="/changes" onclick={() => leaveDialog?.close()}>{m.shell.review}</a>
				<button type="button" class="button" onclick={discardAndLeave}>{m.shell.discardAndLeave}</button>
				<button type="button" class="textbutton" onclick={() => leaveDialog?.close()}>{m.shell.keepEditing}</button>
			</div>
		</dialog>
	{/if}

	<HintBar />
</div>

<style>
	.app {
		display: grid;
		grid-template-columns: minmax(0, 1fr);
		grid-template-rows: auto 1fr;
		min-height: 100vh;
	}
	.app.withsections {
		grid-template-columns: 13rem 1fr;
	}
	.top {
		grid-column: 1 / -1;
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: var(--space-6);
		padding: var(--space-3) var(--space-6);
		border-bottom: 1px solid var(--rule);
	}
	.name {
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
		font-size: var(--step-2);
		font-weight: 600;
		text-decoration: none;
	}
	.state {
		margin-left: auto;
		display: flex;
		align-items: center;
		gap: var(--space-5);
	}
	.source {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		min-width: 0;
	}
	.source > .caps {
		font-size: var(--step--1);
	}
	.source select {
		padding: var(--space-1) var(--space-2);
	}
	/* A catalog's file name, in the File font since it's a path's last part. */
	.file {
		font-family: var(--font-mono);
		font-size: var(--step--1);
		max-width: 18rem;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.game,
	.scan {
		font-size: var(--step--1);
		color: var(--muted);
	}
	.game.running {
		color: var(--warn);
	}
	/* The sections column stays put as the page scrolls, fitted above the
	   hints, and scrolls on its own when it runs longer than the window. */
	.sections {
		position: sticky;
		top: 0;
		align-self: start;
		overflow-y: auto;
		overscroll-behavior: contain;
		padding: var(--space-5) var(--space-4) var(--space-5) var(--space-6);
		border-right: 1px solid var(--rule);
	}
	.sections ul {
		list-style: none;
		margin: 0 0 var(--space-5);
		padding: 0;
	}
	.sections a {
		display: block;
		padding: var(--space-1) 0;
		font-size: var(--step-1);
		text-decoration: none;
		color: var(--muted);
	}
	.sections a:hover {
		color: var(--ink);
	}
	/* A recording's name can outrun the column; its whole name is its title. */
	.sections a.clip {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.sections a[aria-current='page'] {
		color: var(--accent);
		font-weight: 600;
	}
	/* A group's label, never as bright as the current section. */
	.group {
		margin-top: var(--space-4);
		font-size: var(--step--1);
		color: var(--muted);
	}
	main {
		padding: var(--space-5) var(--space-6) 5rem;
		min-width: 0;
	}
	main.solo {
		max-width: 64rem;
	}
	@media (max-width: 60rem) {
		.app.withsections {
			grid-template-columns: minmax(0, 1fr);
		}
		.top {
			flex-wrap: wrap;
			gap: var(--space-2) var(--space-5);
		}
		.state {
			flex-wrap: wrap;
			gap: var(--space-2) var(--space-4);
		}
		.group {
			display: none;
		}
		.sections {
			position: static;
			overflow: visible;
			display: flex;
			gap: var(--space-5);
			border-right: 0;
			border-bottom: 1px solid var(--rule);
			padding: var(--space-2) var(--space-6);
			overflow-x: auto;
		}
		.sections ul {
			display: flex;
			gap: var(--space-5);
			margin: 0;
		}
	}
	.mode {
		display: inline-flex;
	}
	/* The draft's size beside CHANGES, in the section's own ink. */
	.sections .count {
		margin-left: var(--space-2);
		font-variant-numeric: tabular-nums;
		color: var(--muted);
	}
	.leave {
		width: min(30rem, 90vw);
		margin: 18vh auto auto;
		padding: var(--space-4);
		background: var(--smoke);
		color: var(--ink);
		border: 1px solid var(--rule-strong);
		border-radius: var(--radius);
		box-shadow: 0 1.5rem 3rem rgb(0 0 0 / 0.45);
	}
	.leave::backdrop {
		background: rgb(0 0 0 / 0.45);
	}
	.leave h2 {
		font-size: var(--step-1);
		font-weight: 600;
	}
	.leave p {
		margin: var(--space-2) 0 var(--space-4);
	}
	.leave .buttons {
		display: flex;
		align-items: baseline;
		flex-wrap: wrap;
		gap: var(--space-2);
	}
	.leave .button {
		text-decoration: none;
	}
</style>
