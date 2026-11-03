<script lang="ts">
	import '#lib/ui/theme.css';
	import { goto, invalidate, invalidateAll } from '$app/navigation';
	import { page } from '$app/state';
	import { ApiError } from '#lib/api/client.ts';
	import { live, watchStatus } from '#lib/api/live.svelte.ts';
	import { menuHref, menusDependency } from '#lib/menus/api.ts';
	import { recordingHref, recordingsDependency, saveAll, setOpen } from '#lib/recordings/api.ts';
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
	import { handleKey, useShortcuts } from '#lib/ui/shortcuts.svelte.ts';
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
	// The recording chosen, as listed: its page is one click away.
	let chosen = $derived(recordings.find((r) => r.name === openRecording) ?? null);

	// The menus, as SkyUI lists them: by display text, the inactive ones
	// (which the game wouldn't list at all) after the rest, and only those
	// matching the filter.
	let filter = $state('');
	let allMenus = $derived(data.menus?.menus ?? []);
	let menus = $derived.by(() => {
		const q = filter.trim().toLowerCase();
		const list = allMenus.filter((s) => !q || s.nameText.toLowerCase().includes(q) || s.name.toLowerCase().includes(q) || s.mod.toLowerCase().includes(q));
		const byText = (a: (typeof list)[number], b: (typeof list)[number]) => a.nameText.localeCompare(b.nameText, undefined, { sensitivity: 'base' });
		return [...list.filter((s) => s.active).sort(byText), ...list.filter((s) => !s.active).sort(byText)];
	});
	let firstInactive = $derived(menus.findIndex((s) => !s.active));
	// The menu on screen, by its page's address.
	let currentMenu = $derived.by(() => {
		const m = /^\/menus\/([^/]+)/.exec(page.url.pathname);
		return m ? decodeURIComponent(m[1]!) : '';
	});
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

	// Save writes the whole draft from anywhere, as the Changes page's Save
	// does. What can't be written, that page explains, so a failure goes
	// there. The Changes page's own ⌘⏎ takes over while it's open.
	let saving = $state(false);
	let saveError = $state('');
	let savedCount = $state(0);
	async function saveNow() {
		if (saving || !pending) return;
		saving = true;
		saveError = '';
		try {
			savedCount = (await saveAll()).saved.length;
			await invalidateAll();
		} catch (e) {
			saveError = e instanceof ApiError ? e.message : String(e);
			await invalidateAll();
			if (!here('/changes')) await goto('/changes');
		} finally {
			saving = false;
		}
	}
	useShortcuts(() => (editing && pending > 0 ? [{ key: 'Enter', label: m.nav.save, command: true, disabled: saving, run: saveNow }] : []));

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
		<!-- The column, as SkyUI's MCM lays its left side out: the session at
		     the top, where to go, then the menus, which take the rest. -->
		<div class="column" use:fitToViewport>
			<section class="session" class:editing aria-label={m.shell.session}>
				<span title={modeError || (editing ? m.shell.editingTitle : m.shell.readOnlyTitle)}>
					<Segmented bind:value={modeValue} options={[{ value: 'read', label: m.shell.readOnly, quiet: true }, { value: 'editing', label: m.shell.editing }]} label={m.shell.mode} onchange={chooseMode} />
				</span>
				<label class="adding">
					<span class="caps">{editing ? m.shell.recording : m.shell.recordingReadOnly}</span>
					<select value={openRecording} title={openError || (editing ? m.shell.recordingTitle : m.shell.recordingReadOnlyTitle)} onchange={(e) => chooseRecording(e.currentTarget.value, e.currentTarget)}>
						{#if !openRecording}<option value="" disabled>{m.shell.chooseRecording}</option>{/if}
						{#each recordings as r (r.name)}
							<option value={r.name}>{r.unlisted ? m.shell.disabledOption({ name: r.name }) : r.name}</option>
						{/each}
						{#if editing}<option value={newValue}>{m.adding.newRecording}</option>{/if}
					</select>
				</label>
				{#if chosen}
					{@const unsaved = editing && (chosen.drafted || chosen.deleted)}
					<a class="caps rec" class:unsaved class:disabled={chosen.unlisted} href={recordingHref(chosen.name)} title={[m.nav.openRecording({ name: chosen.name }), chosen.unlisted ? m.nav.disabledHere : '', unsaved ? m.nav.unsavedHere : ''].filter(Boolean).join(' · ')} aria-current={here(recordingHref(chosen.name)) ? 'page' : undefined}>
						<span class="marker"><Diamond hollow /></span>
						<span class="clip">{chosen.name}</span>
						{#if unsaved}<span class="visually-hidden">{m.nav.unsavedHere}</span>{/if}
						{#if chosen.unlisted}<span class="visually-hidden">{m.nav.disabledHere}</span>{/if}
					</a>
				{/if}
				{#if editing && pending > 0}
					<div class="save">
						{#key pending}<button type="button" class="button unsaved" disabled={saving} onclick={saveNow}>{m.nav.save}</button>{/key}
						<a class="caps review" href="/changes" aria-current={here('/changes') ? 'page' : undefined}>{m.nav.review}</a>
					</div>
					<p class="note" role="status">{m.nav.changesTitle({ count: pending })}</p>
					{#if saveError}<p class="note error" role="alert">{saveError}</p>{/if}
				{:else if editing}
					<a class="changes caps" href="/changes" aria-current={here('/changes') ? 'page' : undefined}>
						{m.nav.changes}<span class="count">{pending}</span>
					</a>
					{#if savedCount}<p class="note" role="status">{m.nav.savedNote({ count: savedCount })}</p>{/if}
				{/if}
			</section>
			<nav class="sections" aria-label={m.nav.sections}>
				<ul>
					<li><a class="caps" href="/search" aria-current={here('/search') ? 'page' : undefined}>{m.nav.search}</a></li>
					<li><a class="caps" href="/compare" aria-current={here('/compare') ? 'page' : undefined}>{m.nav.compare}</a></li>
					<li><a class="caps" href="/recordings" aria-current={here('/recordings') ? 'page' : undefined}>{m.nav.recordings}</a></li>
				</ul>
			</nav>
			<nav class="menus" aria-label={m.nav.menus}>
				<a class="caps group" href="/menus" aria-current={page.url.pathname === '/menus' ? 'page' : undefined}>{m.nav.menus}<span class="count">{allMenus.length}</span></a>
				<input type="search" bind:value={filter} placeholder={m.nav.menusFilter} aria-label={m.nav.menusFilter} spellcheck="false" autocomplete="off" />
				{#if allMenus.length && !menus.length}
					<p class="note">{m.common.noMatch({ query: filter })}</p>
				{:else}
					<ul>
						{#each menus as s, i (s.id)}
							{#if i === firstInactive}<li class="group caps" aria-hidden="true">{m.nav.inactiveMenus}</li>{/if}
							<li>
								<a class="caps menu" class:inactive={!s.active} href={menuHref(s.id)} title={s.name !== s.nameText ? s.name : undefined} aria-current={currentMenu === s.id ? 'page' : undefined}>
									<span class="marker">{#if currentMenu === s.id}<Diamond />{/if}</span>
									<span class="clip">{s.nameText}</span>
								</a>
							</li>
						{/each}
					</ul>
				{/if}
			</nav>
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
		grid-template-columns: 18rem 1fr;
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
		--chevron-inset: var(--space-2);
		padding-block: var(--space-1);
		padding-left: var(--space-2);
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
	/* The column stays put as the page scrolls, fitted above the hints:
	   the session, the sections, then the menus, which take what's left and
	   scroll on their own. */
	.column {
		position: sticky;
		top: 0;
		align-self: start;
		display: flex;
		flex-direction: column;
		min-height: 0;
		border-right: 1px solid var(--rule);
	}
	/* The editing session: the mode, the recording, and while editing the
	   way to the draft. */
	.session {
		flex: none;
		display: grid;
		gap: var(--space-3);
		padding: var(--space-4) var(--space-4) var(--space-4) var(--space-6);
		border-bottom: 1px solid var(--rule);
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
		--chevron-inset: var(--space-2);
		padding-block: var(--space-1);
		padding-left: var(--space-2);
	}
	.sections {
		flex: none;
		padding: var(--space-3) var(--space-4) var(--space-3) var(--space-6);
		border-bottom: 1px solid var(--rule);
	}
	.sections ul,
	.menus ul {
		list-style: none;
		margin: 0;
		padding: 0;
	}
	.sections a,
	.menus a,
	.changes,
	.session .rec {
		display: block;
		padding: var(--space-1) 0;
		font-size: var(--step-1);
		text-decoration: none;
		color: var(--muted);
	}
	.sections a:hover,
	.menus a:hover,
	.session .rec:hover {
		color: var(--ink);
	}
	.sections a[aria-current='page'],
	.menus a[aria-current='page'],
	.session .rec[aria-current='page'] {
		color: var(--accent);
		font-weight: 600;
	}
	/* The menus, as SkyUI's mod list: a filter, then every menu by name,
	   the current one marked with the diamond. */
	.menus {
		flex: 1 1 auto;
		min-height: 0;
		display: flex;
		flex-direction: column;
		padding: var(--space-4) var(--space-4) var(--space-4) var(--space-6);
	}
	.menus input {
		margin: var(--space-2) 0 var(--space-2);
		padding: var(--space-1) var(--space-2);
	}
	.menus ul {
		flex: 1 1 auto;
		min-height: 0;
		overflow-y: auto;
		overscroll-behavior: contain;
		margin-right: calc(-1 * var(--space-2));
		padding-right: var(--space-2);
	}
	/* A row with a slot for the marker, then its name, which can outrun the
	   column (a $KEY name is its title). */
	.menus a.menu,
	.session a.rec {
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
	.menu.inactive {
		opacity: 0.55;
	}
	/* The group label over the inactive menus, and the MENUS label itself,
	   never as bright as the current item. */
	.group {
		margin: 0;
		padding: 0;
		font-size: var(--step--1);
		color: var(--muted);
	}
	li.group {
		margin-top: var(--space-3);
		padding-bottom: var(--space-1);
	}
	a.group {
		display: flex;
		justify-content: space-between;
		align-items: baseline;
	}
	a.group[aria-current='page'] {
		color: var(--accent);
	}
	/* The recording chosen: a hollow diamond, its name. Unsaved changes are
	   full ink, as CHANGES is; a mod the profile has disabled is amber. */
	.session .rec.unsaved {
		color: var(--ink);
	}
	.session .rec.disabled {
		color: var(--warn);
	}
	.session .rec[aria-current='page'] {
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
	.changes {
		display: flex;
		align-items: baseline;
		justify-content: space-between;
	}
	/* A draft with something in it: Save, the one amber fill in the app,
	   since unsaved work is the one warning the player can lose something
	   to, with the way to review it beside, and a line saying what's
	   waiting. It rises in when the draft starts and again as each change
	   lands, then breathes a soft amber light so it's never mistaken for
	   settled. */
	.save {
		display: flex;
		align-items: center;
		gap: var(--space-4);
	}
	.save .button.unsaved {
		background: var(--warn);
		border-color: var(--warn);
		color: var(--panel);
		font-weight: 600;
		box-shadow: 0 2px 14px -2px color-mix(in oklab, var(--warn) 55%, transparent);
		animation:
			arrive 320ms cubic-bezier(0.16, 1, 0.3, 1),
			breathe 2.8s ease-in-out 320ms infinite;
	}
	.save .button.unsaved:hover:not(:disabled) {
		border-color: var(--ink);
		box-shadow: 0 3px 18px -2px color-mix(in oklab, var(--warn) 85%, transparent);
	}
	@keyframes arrive {
		from {
			opacity: 0;
			transform: translateY(6px);
		}
	}
	@keyframes breathe {
		50% {
			box-shadow: 0 2px 22px 0 color-mix(in oklab, var(--warn) 80%, transparent);
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.save .button.unsaved {
			animation: none;
		}
	}
	.review {
		font-size: var(--step-0);
		text-decoration: none;
		color: var(--muted);
	}
	.review:hover {
		color: var(--ink);
	}
	.review[aria-current='page'] {
		color: var(--accent);
	}
	.session .note {
		margin: 0;
	}
	.note.error {
		color: var(--danger);
	}
	/* A count in the link's own ink: how many menus, or how many recordings
	   the draft changes. */
	.count {
		font-variant-numeric: tabular-nums;
		font-weight: 400;
	}
	.note {
		margin: var(--space-2) 0 0;
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
		/* The column becomes rows under the bar: the session, the sections,
		   and the menus with a short list under their filter. */
		.column {
			position: static;
			border-right: 0;
			border-bottom: 1px solid var(--rule);
		}
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
			gap: var(--space-2);
		}
		.sections {
			padding: var(--space-2) var(--space-6);
		}
		.sections ul {
			display: flex;
			gap: var(--space-5);
		}
		.menus {
			flex: none;
			padding: var(--space-2) var(--space-6) var(--space-3);
		}
		.menus ul {
			max-height: 12rem;
		}
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
