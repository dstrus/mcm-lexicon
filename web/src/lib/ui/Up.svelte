<!-- The way up from an item to the list it came from, named for where it
     goes: the same move as the hint bar's Esc, where the eye lands first. -->
<script lang="ts">
	let { href, label }: { href: string; label: string } = $props();
</script>

<a class="up caps" {href}>
	<svg viewBox="0 0 8 12" width="6" height="9" aria-hidden="true"><path d="M6.5 1 1.5 6l5 5" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" /></svg>
	{label}
</a>

<style>
	.up {
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
		margin-bottom: var(--space-3);
		font-size: var(--step--1);
		color: var(--muted);
		text-decoration: none;
	}
	.up:hover {
		color: var(--ink);
		text-decoration: underline;
		text-decoration-color: var(--accent);
		text-underline-offset: 0.3em;
	}
</style>
