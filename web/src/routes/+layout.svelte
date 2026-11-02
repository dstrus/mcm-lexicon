<script lang="ts">
	import '#lib/ui/theme.css';
	import { invalidate, invalidateAll } from '$app/navigation';
	import { page } from '$app/state';
	import { ApiError } from '#lib/api/client.ts';
	import { live, watchStatus } from '#lib/api/live.svelte.ts';
	import { menusDependency } from '#lib/menus/api.ts';
	import QuickFind from '#lib/ui/QuickFind.svelte';
	import { setSource } from '#lib/api/status.ts';
	import { m } from '#lib/copy/messages.ts';
	import SourceForm from '#lib/source/SourceForm.svelte';
	import HintBar from '#lib/ui/HintBar.svelte';
	import ThemeSwitch from '#lib/ui/ThemeSwitch.svelte';
	import { clipped } from '#lib/ui/clipped.ts';
	import { fitToViewport } from '#lib/ui/fit.ts';
	import { handleKey } from '#lib/ui/shortcuts.svelte.ts';
	import type { LayoutProps } from './$types';

	let { data, children }: LayoutProps = $props();

	$effect(() => watchStatus(data.status, () => invalidate(menusDependency)));

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
	// The sections that aren't built yet, named in the column as they'll be.
	const unbuilt = [m.nav.recordings, m.nav.compare];

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
		<nav class="sections" aria-label={m.nav.sections} use:fitToViewport>
			<ul>
				<li><a class="caps" href="/menus" aria-current={here('/menus') ? 'page' : undefined}>{m.nav.menus}</a></li>
				<li><a class="caps" href="/search" aria-current={here('/search') ? 'page' : undefined}>{m.nav.search}</a></li>
			</ul>
			<p class="group caps">{m.nav.notBuiltYet}</p>
			<ul aria-label={m.nav.notBuiltYet}>
				{#each unbuilt as label (label)}
					<li><span class="caps">{label}</span></li>
				{/each}
			</ul>
		</nav>
		<main>{@render children()}</main>
		<QuickFind menus={data.menus?.menus ?? []} />
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
	.sections a,
	.sections span {
		display: block;
		padding: var(--space-1) 0;
		font-size: var(--step-1);
		text-decoration: none;
		color: var(--muted);
	}
	.sections a:hover {
		color: var(--ink);
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
</style>
