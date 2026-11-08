<!-- A cog, for Settings, drawn as Hermaeus draws it for Config. -->
<script lang="ts">
	let { size = 16 }: { size?: number | string } = $props();
</script>

<svg viewBox="0 0 16 16" width={size} height={size} aria-hidden="true">
	<path
		d="M13.12 6.82L14.68 7.06L14.68 8.94L13.12 9.18L12.45 10.78L13.39 12.06L12.06 13.39L10.78 12.45L9.18 13.12L8.94 14.68L7.06 14.68L6.82 13.12L5.22 12.45L3.94 13.39L2.61 12.06L3.55 10.78L2.88 9.18L1.32 8.94L1.32 7.06L2.88 6.82L3.55 5.22L2.61 3.94L3.94 2.61L5.22 3.55L6.82 2.88L7.06 1.32L8.94 1.32L9.18 2.88L10.78 3.55L12.06 2.61L13.39 3.94L12.45 5.22Z"
	/>
	<circle cx="8" cy="8" r="2.25" />
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
