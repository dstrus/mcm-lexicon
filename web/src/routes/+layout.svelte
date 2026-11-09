<script lang="ts">
	import '#lib/ui/theme.css';
	import { goto, invalidate, invalidateAll } from '$app/navigation';
	import { page } from '$app/state';
	import { ApiError } from '#lib/api/client.ts';
	import { live, watchStatus } from '#lib/api/live.svelte.ts';
	import { menuHref, menusDependency } from '#lib/menus/api.ts';
	import { column } from '#lib/menus/column.svelte.ts';
	import { recordingHref, recordingsDependency, redo, saveAll, setOpen, undo } from '#lib/recordings/api.ts';
	import QuickFind from '#lib/ui/QuickFind.svelte';
	import Cog from '#lib/ui/Cog.svelte';
	import Diamond from '#lib/ui/Diamond.svelte';
	import Mark from '#lib/ui/Mark.svelte';
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
	// The list by its folder's name, for the top bar: the instance's, or for
	// a Data folder, the game's folder it's in.
	const folderName = (p: string) => p.replace(/[\\/]+$/, '').split(/[\\/]/).pop() ?? p;
	let listName = $derived(source ? folderName(source.kind === 'data' ? source.path.replace(/[\\/][^\\/]+[\\/]*$/, '') : source.path) : '');
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
	// The recording chosen, as listed: its page is one click away. One
	// remembered but gone (deleted, or from another list) is none, and the
	// select shows None rather than nothing.
	let chosen = $derived(recordings.find((r) => r.name.toLowerCase() === openRecording.toLowerCase()) ?? null);
	let shownRecording = $derived(chosen?.name ?? '');

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
	// The menu screen's arrow keys move through the menus as listed here.
	$effect(() => {
		column.ids = menus.map((s) => s.id);
	});
	// Off a menu, the keys aren't in the list; in it, the menu on screen
	// stays in view as they move.
	$effect(() => {
		if (!currentMenu) column.active = false;
		else if (column.active) document.querySelector('.menus a.menu[aria-current="page"]')?.scrollIntoView({ block: 'nearest' });
	});
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
			if (!(await creator?.choose(true))) select.value = shownRecording;
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
	useShortcuts(() => (editing && pending > 0 ? [{ key: 'Enter', label: m.nav.save, command: true, writes: true, disabled: saving, run: saveNow }] : []));

	// Undo and redo step through the draft's changes, from any screen, while
	// editing; a text field keeps ⌘Z for its own typing. Save starts the
	// history afresh, since a write can't be undone.
	let stepping = $state(false);
	async function stepHistory(back: boolean) {
		if (stepping) return;
		stepping = true;
		saveError = '';
		try {
			await (back ? undo() : redo());
		} catch (e) {
			saveError = e instanceof ApiError ? e.message : String(e);
		} finally {
			await invalidateAll();
			stepping = false;
		}
	}
	let undoable = $derived(data.status.undo ?? 0);
	let redoable = $derived(data.status.redo ?? 0);
	useShortcuts(() =>
		editing
			? [
					...(undoable ? [{ key: 'z', shown: 'Z', label: m.nav.undo, command: true, notTyping: true, disabled: stepping, run: () => stepHistory(true) }] : []),
					...(redoable ? [{ key: 'z', shown: 'Z', label: m.nav.redo, command: true, shift: true, notTyping: true, disabled: stepping, run: () => stepHistory(false) }] : [])
				]
			: []
	);

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
		<a class="name caps" href="/"><Mark size="1.35em" /> {m.shell.name}</a>
		<div class="state">
			<!-- The list in use, linked to where it's changed. -->
			{#if source}
				<a class="source list" href="/settings#mod-list" title={m.shell.listTitle({ path: source.path })}>
					<span class="caps muted">{m.shell.list}</span>
					<span class="file" use:clipped>{listName}</span>
				</a>
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
				<a class="source list" href="/settings#mod-list" title={m.shell.listTitle({ path: catalog })}>
					<span class="caps muted">{m.shell.catalog}</span>
					<span class="file" use:clipped>{fileName(catalog)}</span>
				</a>
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
		<!-- The column, as SkyUI's MCM lays its left side out: Settings on its
		     own at the top, as Hermaeus's Config is, then the session, where
		     to go, and the menus, which take the rest. -->
		<div class="column" use:fitToViewport>
			<a class="config caps" href="/settings" aria-current={here('/settings') ? 'page' : undefined}><Cog size="0.8em" />{m.nav.settings}</a>
			<section class="session" class:editing aria-label={m.shell.session}>
				<span title={modeError || (editing ? m.shell.editingTitle : m.shell.readOnlyTitle)}>
					<Segmented bind:value={modeValue} options={[{ value: 'read', label: m.shell.readOnly, quiet: true }, { value: 'editing', label: m.shell.editing }]} label={m.shell.mode} onchange={chooseMode} />
				</span>
				<div class="adding">
					<label class="caps" for="adding-to">{editing ? m.shell.recording : m.shell.recordingReadOnly}</label>
					<span class="pick">
						<select id="adding-to" value={shownRecording} title={openError || (editing ? m.shell.recordingTitle : m.shell.recordingReadOnlyTitle)} onchange={(e) => chooseRecording(e.currentTarget.value, e.currentTarget)}>
						<!-- None is a choice too: browsing while editing adds nothing,
						     and the first change asks which recording it goes to. -->
						<option value="">{m.shell.noRecording}</option>
						{#each recordings as r (r.name)}
							<option value={r.name}>{r.unlisted ? m.shell.disabledOption({ name: r.name }) : r.name}</option>
						{/each}
						{#if editing}<option value={newValue}>{m.adding.newRecording}</option>{/if}
					</select>
					<!-- The way to the chosen recording's page, beside it: in
					     full ink while it has unsaved changes, amber while its
					     mod is disabled. -->
					{#if chosen}
						{@const unsaved = editing && (chosen.drafted || chosen.deleted)}
						{@const says = [m.nav.openRecording({ name: chosen.name }), chosen.unlisted ? m.nav.disabledHere : '', unsaved ? m.nav.unsavedHere : ''].filter(Boolean).join(' · ')}
						<a class="open" class:unsaved class:disabled={chosen.unlisted} href={recordingHref(chosen.name)} title={says} aria-label={says} aria-current={here(recordingHref(chosen.name)) ? 'page' : undefined}>
							<svg viewBox="0 0 8 12" width="6" height="9" aria-hidden="true"><path d="M1.5 1l5 5-5 5" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" /></svg>
						</a>
					{/if}
					</span>
				</div>
				{#if editing && pending > 0}
					<div class="saving">
						{#key pending}<button type="button" class="button save" disabled={saving} onclick={saveNow}>{m.nav.save}</button>{/key}
						<a class="caps review" href="/changes" title={m.nav.changesTitle({ count: pending })} aria-current={here('/changes') ? 'page' : undefined}>{m.nav.review}<span class="count">{pending}</span></a>
					</div>
					<p class="visually-hidden" role="status">{m.nav.changesTitle({ count: pending })}</p>
					{#if saveError}<p class="note error" role="alert">{saveError}</p>{/if}
				{:else if editing}
					<a class="changes caps" href="/changes" aria-current={here('/changes') ? 'page' : undefined}>
						{m.nav.changes}<span class="count">{pending}</span>
					</a>
					{#if savedCount}<p class="note" role="status">{m.nav.savedNote({ count: savedCount })}</p>{/if}
				{/if}
			</section>
			<!-- Where else to go, on one row, so the menus start high: they're
			     where the time is spent. -->
			<nav class="sections" aria-label={m.nav.sections}>
				<ul>
					<li><a class="caps" href="/search" aria-current={here('/search') ? 'page' : undefined}>{m.nav.search}</a></li>
					<li><a class="caps" href="/recordings" aria-current={here('/recordings') ? 'page' : undefined}>{m.nav.recordings}</a></li>
					<li><a class="caps" href="/compare" aria-current={here('/compare') ? 'page' : undefined}>{m.nav.compare}</a></li>
				</ul>
			</nav>
			<!-- The filter heads the menus and counts them; the list's home is
			     the mark in the top bar. -->
			<nav class="menus" class:active={column.active} aria-label={m.nav.menus}>
				<input type="search" bind:value={filter} placeholder={m.nav.menusFilter({ count: allMenus.length })} aria-label={m.nav.menusFilter({ count: allMenus.length })} spellcheck="false" autocomplete="off" />
				{#if allMenus.length && !menus.length}
					<p class="note">{m.common.noMatch({ query: filter })}</p>
				{:else}
					<ul>
						{#each menus as s, i (s.id)}
							{#if i === firstInactive}<li class="group caps" aria-hidden="true">{m.nav.inactiveMenus}</li>{/if}
							<li>
								<a class="caps menu" class:inactive={!s.active} href={menuHref(s.id)} onclick={() => (column.active = false)} aria-current={currentMenu === s.id ? 'page' : undefined}>
									<span class="marker">{#if currentMenu === s.id}<Diamond />{/if}</span>
									<span class="clip" use:clipped={s.name !== s.nameText ? s.name : undefined}>{s.nameText}</span>
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
	/* The list's name links to Settings, muted until pointed at, like the
	   column's sections. */
	.source.list {
		text-decoration: none;
		color: var(--ink);
	}
	.source.list .file {
		text-decoration: underline;
		text-decoration-color: color-mix(in oklab, var(--accent) 75%, var(--ground));
		text-underline-offset: 0.2em;
	}
	.source.list:hover .file {
		text-decoration-color: var(--accent);
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
	/* Settings heads the column on its own, in full ink after a drawn cog,
	   as Hermaeus's Config does: MCM Lexicon's own settings, not one of
	   the list's sections. */
	.config {
		flex: none;
		display: flex;
		align-items: center;
		gap: var(--space-2);
		padding: var(--space-3) var(--space-4) var(--space-3) var(--space-6);
		border-bottom: 1px solid var(--rule);
		font-size: var(--step-1);
		text-decoration: none;
		color: var(--ink);
	}
	.config:hover {
		color: var(--accent);
	}
	.config[aria-current='page'] {
		color: var(--accent);
		font-weight: 600;
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
	/* The recording's select, and the way to its page beside it, a square
	   of the field's own glass and height. */
	.pick {
		display: flex;
		gap: var(--space-1);
		min-width: 0;
	}
	.adding select {
		flex: 1 1 auto;
		min-width: 0;
		--chevron-inset: var(--space-2);
		padding-block: var(--space-1);
		padding-left: var(--space-2);
	}
	.open {
		flex: none;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 2rem;
		background: var(--glass-strong);
		border: 1px solid var(--rule-strong);
		border-radius: var(--radius);
		color: var(--muted);
	}
	.open:hover {
		border-color: var(--accent);
		color: var(--ink);
	}
	.open:focus-visible {
		border-color: var(--accent);
		outline: 1px solid var(--accent);
		outline-offset: -1px;
	}
	.open.unsaved {
		color: var(--ink);
	}
	.open.disabled {
		color: var(--warn);
	}
	.open[aria-current='page'] {
		border-color: var(--accent);
		color: var(--accent);
	}
	/* The sections on one row, in the session's label size, so they read as
	   a way out rather than the column's subject. */
	/* Its rule heads the menus, so it has to be seen: mixed from the muted
	   ink, as the slider's track is, since even the strong rule is lost
	   against the column's dark. */
	.sections {
		flex: none;
		padding: var(--space-2) var(--space-4) var(--space-2) var(--space-6);
		border-bottom: 1px solid color-mix(in srgb, var(--muted) 40%, var(--ground));
	}
	.sections ul {
		display: flex;
		flex-wrap: wrap;
		justify-content: space-between;
		gap: 0 var(--space-3);
	}
	.sections ul,
	.menus ul {
		list-style: none;
		margin: 0;
		padding: 0;
	}
	.sections a,
	.menus a,
	.changes {
		display: block;
		padding: var(--space-1) 0;
		font-size: var(--step-1);
		text-decoration: none;
		color: var(--muted);
	}
	.sections a {
		font-size: var(--step-0);
	}
	.sections a:hover,
	.menus a:hover {
		color: var(--ink);
	}
	.sections a[aria-current='page'],
	.menus a[aria-current='page'] {
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
		padding: var(--space-3) var(--space-4) var(--space-4) var(--space-6);
	}
	.menus input {
		margin: 0 0 var(--space-2);
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
	/* The arrow keys in the menu list: the menu on screen takes the
	   selected haze, as the selected row of a list does. */
	/* Room inside the keys' hairline, without moving the row. */
	.menus a.menu {
		margin-inline: calc(-1 * var(--space-2));
		padding-inline: var(--space-2);
	}
	.menus.active a.menu[aria-current='page'] {
		background: var(--select);
		outline: 1px solid var(--keys);
		outline-offset: -1px;
	}
	.menus a.menu {
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
	/* The group label over the inactive menus, never as bright as the
	   current item. */
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
	.saving {
		display: flex;
		align-items: center;
		gap: var(--space-4);
	}
	/* Review, with how many recordings the draft changes beside it. */
	.review {
		display: inline-flex;
		align-items: baseline;
		gap: var(--space-2);
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
	/* A count in the link's own ink: how many recordings the draft changes. */
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
			justify-content: flex-start;
			gap: var(--space-5);
		}
		.config {
			padding: var(--space-2) var(--space-6);
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
