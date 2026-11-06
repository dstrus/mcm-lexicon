<!-- The controller buttons a key step can take, as a select's options: in
     Skyrim VR, the motion controllers' buttons by controller family first,
     each naming the gamepad button it writes; then the gamepad's own. -->
<script lang="ts">
	import { m } from '#lib/copy/messages.ts';
	import { gamepadButtons, vrFamilies } from '#lib/menus/keys.ts';

	let { vr }: { vr: boolean } = $props();
	const k = m.keys;
	const padName = (code: number) => gamepadButtons.find((b) => b.code === code)?.name ?? String(code);
	// "A · A button" says it twice: a button named as the gamepad's is shown once.
	const label = (name: string, code: number) => (padName(code).startsWith(name + ' ') ? padName(code) : k.vrAs({ button: name, pad: padName(code) }));
</script>

{#if vr}
	{#each vrFamilies as family (family.name)}
		<optgroup label={family.name}>
			{#each family.buttons as b (b.name)}<option value={String(b.code)}>{label(b.name, b.code)}</option>{/each}
		</optgroup>
	{/each}
	<optgroup label={k.gamepadGroup}>
		{#each gamepadButtons as b (b.code)}<option value={String(b.code)}>{b.name}</option>{/each}
	</optgroup>
{:else}
	{#each gamepadButtons as b (b.code)}<option value={String(b.code)}>{b.name}</option>{/each}
{/if}
