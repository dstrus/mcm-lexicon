<!-- Tabs: pages of one section, each its own URL, as a row of title-size
     caps over a hairline; the current one is ink with an accent underline.
     A tab's count follows its label, quieter. -->
<script lang="ts">
	import { page } from '$app/state';

	let { tabs, label }: { tabs: { href: string; label: string; count?: number }[]; label: string } = $props();
</script>

<nav aria-label={label}>
	{#each tabs as tab (tab.href)}
		<a class="caps" href={tab.href} aria-current={page.url.pathname === tab.href ? 'page' : undefined}>{tab.label}{#if tab.count !== undefined}<span class="count">{tab.count.toLocaleString()}</span>{/if}</a>
	{/each}
</nav>

<style>
	nav {
		display: flex;
		gap: var(--space-5);
		margin-top: var(--space-4);
		border-bottom: 1px solid var(--rule);
	}
	a {
		padding: var(--space-2) 0;
		font-size: var(--step-1);
		color: var(--muted);
		text-decoration: none;
		margin-bottom: -1px;
		border-bottom: 1px solid transparent;
	}
	a:hover {
		color: var(--ink);
	}
	a[aria-current='page'] {
		color: var(--ink);
		font-weight: 600;
		border-bottom: 2px solid var(--accent);
	}
	.count {
		margin-left: var(--space-2);
		font-family: var(--font-text);
		font-size: var(--step--1);
		font-weight: 400;
		letter-spacing: 0;
		color: var(--muted);
	}
</style>
