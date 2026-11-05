<!-- The folder picker's row icons, drawn in one thin stroke: a folder, a
     file, or the way up. -->
<script lang="ts">
	let { kind }: { kind: 'folder' | 'file' | 'up' } = $props();
</script>

<svg viewBox="0 0 16 16" width="16" height="16" aria-hidden="true">
	{#if kind === 'folder'}
		<path d="M1.75 3.25h4.5l1.5 1.75h6.5v7.75H1.75Z" />
	{:else if kind === 'file'}
		<path d="M3.75 1.75h5.5l3 3v9.5h-8.5Z" />
		<path d="M9.25 1.75v3h3" />
	{:else}
		<path d="M8 13.25V3.25M4 7.25l4-4 4 4" />
	{/if}
</svg>

<style>
	svg {
		flex: none;
		fill: none;
		stroke: currentColor;
		stroke-width: 1.4;
		stroke-linejoin: round;
		stroke-linecap: round;
	}
</style>
