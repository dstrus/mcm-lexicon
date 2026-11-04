<!-- MCM Lexicon's mark: the face of a Dwemer lexicon, a bound square plate
     with a round dial and the game's diamond at its center, in the current
     text color, with the face and the dial's inside cut out so it sits on
     any ground. The icon files in assets/ are the same drawing in verdigris
     on the game's near-black. -->
<script lang="ts">
	let { size = '1.25em' }: { size?: string } = $props();
</script>

<svg viewBox="0 0 64 64" fill="currentColor" style:height={size} aria-hidden="true">
	<path fill-rule="evenodd" d="M2 2H62V62H2ZM8 8H16V5H48V8H56V16H59V48H56V56H48V59H16V56H8V48H5V16H8ZM15 32a17 17 0 1 0 34 0a17 17 0 1 0-34 0ZM19 32a13 13 0 1 0 26 0a13 13 0 1 0-26 0ZM32 25l7 7l-7 7l-7-7Z" />
</svg>

<style>
	svg {
		width: auto;
		vertical-align: -0.2em;
	}
</style>
