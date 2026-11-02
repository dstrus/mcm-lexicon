<!-- The game's selection marker. Hollow, in ink, it marks something chosen
     for a job rather than selected: the recording A adds steps to. -->
<script lang="ts">
	let { size = 9, hollow = false }: { size?: number; hollow?: boolean } = $props();
</script>

<svg width={size} height={size} viewBox="0 0 10 10" aria-hidden="true">
	{#if hollow}
		<path d="M5 0.9 9.1 5 5 9.1 0.9 5Z" fill="none" stroke="currentColor" stroke-width="1.3" />
	{:else}
		<path d="M5 0 10 5 5 10 0 5Z" fill="var(--accent)" />
	{/if}
</svg>
