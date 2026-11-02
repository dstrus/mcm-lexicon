<script lang="ts">
	import '#lib/ui/theme.css';
	import { invalidate, invalidateAll } from '$app/navigation';
	import { page } from '$app/state';
	import { ApiError } from '#lib/api/client.ts';
	import { live, watchStatus } from '#lib/api/live.svelte.ts';
	import { menusDependency } from '#lib/menus/api.ts';
	import { recordingHref, recordingsDependency, setOpen } from '#lib/recordings/api.ts';
	import QuickFind from '#lib/ui/QuickFind.svelte';
	import Diamond from '#lib/ui/Diamond.svelte';
	import ChooseRecording from '#lib/recordings/ChooseRecording.svelte';
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
	// New recording… asks for the name at once; cancelling it puts the
	// select back to what it showed.
	const newValue = '\u0000new';
	let creator = $state<ChooseRecording>();
	async function chooseRecording(name: string, select: HTMLSelectElement) {
		openError = '';
		if (name === newValue) {
			if (!(await creator?.choose(true))) select.value = openRecording;
			return;
		}
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
		<!-- The column: where to go, and, at its foot, the editing session,
		     whose controls act on the recordings listed above it. -->
		<div class="column" use:fitToViewport>
			<nav class="sections" aria-label={m.nav.sections}>
				<ul>
					<li><a class="caps" href="/menus" aria-current={here('/menus') ? 'page' : undefined}>{m.nav.menus}</a></li>
					<li><a class="caps" href="/search" aria-current={here('/search') ? 'page' : undefined}>{m.nav.search}</a></li>
					<li><a class="caps" href="/recordings" aria-current={page.url.pathname === '/recordings' ? 'page' : undefined}>{m.nav.recordings}</a></li>
					<li><a class="caps" href="/compare" aria-current={here('/compare') ? 'page' : undefined}>{m.nav.compare}</a></li>
				</ul>
				{#if recordings.length}
					<p class="group caps">{m.nav.recordingsGroup}</p>
					<ul aria-label={m.nav.recordingsGroup}>
						{#each recordings as r (r.name)}
							{@const target = editing && r.name === openRecording}
							{@const unsaved = editing && (r.drafted || r.deleted)}
							{@const unticked = r.unlisted}
							<li>
								<a class="caps rec" class:unsaved class:unticked href={recordingHref(r.name)} title={[r.name, unticked ? m.nav.untickedHere : '', target ? m.nav.addingHere : '', unsaved ? m.nav.unsavedHere : ''].filter(Boolean).join(' · ')} aria-current={here(recordingHref(r.name)) ? 'page' : undefined}>
									<span class="marker">{#if target}<Diamond hollow />{/if}</span>
									<span class="clip">{r.name}</span>
									{#if target}<span class="visually-hidden">{m.nav.addingHere}</span>{/if}
									{#if unsaved}<span class="visually-hidden">{m.nav.unsavedHere}</span>{/if}
									{#if unticked}<span class="visually-hidden">{m.nav.untickedHere}</span>{/if}
								</a>
							</li>
						{/each}
					</ul>
				{/if}
			</nav>
			<section class="session" class:editing aria-label={m.shell.session}>
				<span title={modeError || (editing ? m.shell.editingTitle : m.shell.readOnlyTitle)}>
					<Segmented bind:value={modeValue} options={[{ value: 'read', label: m.shell.readOnly, quiet: true }, { value: 'editing', label: m.shell.editing }]} label={m.shell.mode} onchange={chooseMode} />
				</span>
				{#if editing}
					<label class="adding">
						<span class="caps">{m.shell.recording}</span>
						<select value={openRecording} title={openError || m.shell.recordingTitle} onchange={(e) => chooseRecording(e.currentTarget.value, e.currentTarget)}>
							{#if !openRecording}<option value="" disabled>{m.shell.chooseRecording}</option>{/if}
							{#each recordings as r (r.name)}
								<option value={r.name}>{r.unlisted ? m.shell.untickedOption({ name: r.name }) : r.name}</option>
							{/each}
							<option value={newValue}>{m.adding.newRecording}</option>
						</select>
					</label>
					<a class="changes caps" class:pending={pending > 0} href="/changes" title={m.nav.changesTitle({ count: pending })} aria-current={here('/changes') ? 'page' : undefined}>
						{m.nav.changes}<span class="count">{pending}</span>
					</a>
				{:else}
					<p class="note">{m.shell.readOnlyNote}</p>
				{/if}
			</section>
		</div>
		<main>{@render children()}</main>
		<QuickFind menus={data.menus?.menus ?? []} />
		<ChooseRecording bind:this={creator} />
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
	/* The column stays put as the page scrolls, fitted above the hints. Its
	   sections scroll on their own when the recordings run long; the
	   session stays at its foot. */
	.column {
		position: sticky;
		top: 0;
		align-self: start;
		display: flex;
		flex-direction: column;
		min-height: 0;
		border-right: 1px solid var(--rule);
	}
	.sections {
		flex: 1 1 auto;
		min-height: 0;
		overflow-y: auto;
		overscroll-behavior: contain;
		padding: var(--space-5) var(--space-4) var(--space-4) var(--space-6);
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
	/* A recording: a slot for the hollow diamond that marks the one A adds
	   to, then its name, which can outrun the column (its whole name is its
	   title). One with unsaved changes is in full ink, as CHANGES is. */
	.sections a.rec {
		display: flex;
		align-items: center;
		gap: var(--space-2);
	}
	.marker {
		flex: none;
		display: inline-flex;
		width: 9px;
		color: var(--ink);
	}
	.clip {
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.sections a.rec.unsaved {
		color: var(--ink);
	}
	/* In a mod the profile doesn't tick, so the game won't play it: amber, as
	   a step that won't replay is, over unsaved's ink. */
	.sections a.rec.unticked {
		color: var(--warn);
	}
	.sections a.rec[aria-current='page'] {
		color: var(--accent);
	}
	.visually-hidden {
		position: absolute;
		width: 1px;
		height: 1px;
		overflow: hidden;
		clip-path: inset(50%);
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
		/* The column becomes two rows under the bar: the sections, then the
		   session. */
		.column {
			position: static;
			border-right: 0;
			border-bottom: 1px solid var(--rule);
		}
		.sections {
			overflow: visible;
			display: flex;
			gap: var(--space-5);
			padding: var(--space-2) var(--space-6);
			overflow-x: auto;
		}
		.sections ul {
			display: flex;
			gap: var(--space-5);
			margin: 0;
		}
	}
	/* The editing session: the mode, and while editing, where A adds steps
	   and the way to the draft. Set off from the sections by a hairline. */
	.session {
		flex: none;
		display: grid;
		gap: var(--space-3);
		padding: var(--space-4) var(--space-4) var(--space-4) var(--space-6);
		border-top: 1px solid var(--rule);
	}
	.adding {
		display: grid;
		gap: var(--space-1);
		min-width: 0;
	}
	.adding > .caps {
		font-size: var(--step--1);
		color: var(--muted);
	}
	.adding select {
		width: 100%;
		padding: var(--space-1) var(--space-2);
	}
	.changes {
		display: flex;
		align-items: baseline;
		justify-content: space-between;
		font-size: var(--step-1);
		text-decoration: none;
		color: var(--muted);
	}
	/* A draft with something in it is worth noticing: full ink, not the
	   accent, which is for selection. */
	.changes:hover,
	.changes.pending {
		color: var(--ink);
	}
	.changes[aria-current='page'] {
		color: var(--accent);
		font-weight: 600;
	}
	/* The draft's size, in the link's own ink: how many recordings it changes. */
	.count {
		font-variant-numeric: tabular-nums;
		font-weight: 400;
	}
	.note {
		margin: 0;
		font-size: var(--step--1);
		color: var(--muted);
		max-width: 22ch;
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
	/* Under 60rem the session is one row under the sections; after its own
	   rules above, so this wins. */
	@media (max-width: 60rem) {
		.session {
			display: flex;
			flex-wrap: wrap;
			align-items: center;
			gap: var(--space-2) var(--space-5);
			padding: var(--space-2) var(--space-6);
		}
		.adding {
			display: flex;
			align-items: center;
			gap: var(--space-2);
		}
		.adding select {
			width: auto;
			max-width: 16rem;
		}
		.changes {
			justify-content: flex-start;
			gap: var(--space-2);
		}
		.note {
			max-width: none;
		}
	}
</style>
