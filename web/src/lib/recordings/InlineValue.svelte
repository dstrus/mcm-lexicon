<!-- A step's value edited in its table cell: the smallest control for its
     kind, focused on arrival. Enter applies and Esc cancels; a menu entry,
     a color, or a key pressed applies as soon as it's chosen. Toggles
     don't come here: their cell's checkbox flips them. -->
<script lang="ts">
	import type { Option } from '#lib/api/types/catalog.ts';
	import type { Kind } from '#lib/api/types/recordings.ts';
	import { m } from '#lib/copy/messages.ts';
	import { keyName, mouseCodeOf, scanCodeOf } from '#lib/menus/keys.ts';
	import { hex, number } from '#lib/menus/values.ts';
	import Range from '#lib/ui/Range.svelte';
	import { onMount, tick } from 'svelte';

	let {
		kind,
		option,
		value,
		apply,
		cancel,
		fresh = false
	}: {
		kind: Kind;
		option: Option | null | undefined;
		value: string;
		// The option isn't recorded yet: a menu opens on no entry, so choosing
		// any one, the one it has included, records it.
		fresh?: boolean;
		apply: (value: string) => void;
		cancel: () => void;
	} = $props();

	const t = m.editor;
	let draft = $state('');
	let field = $state<HTMLElement>();
	// The field is filled before its text is selected, so typing replaces it.
	onMount(async () => {
		draft = value;
		await tick();
		field?.focus();
		if (field instanceof HTMLInputElement && field.type !== 'color') field.select();
	});

	let range = $derived(option?.slider);
	let choices = $derived(option?.choices ?? []);
	let problem = $derived.by(() => {
		if (kind !== 'slider') return '';
		const n = Number(draft.trim());
		if (draft.trim() === '' || Number.isNaN(n)) return t.notANumber;
		if (range && (n < range.min - 1e-6 || n > range.max + 1e-6)) return t.outOfRange({ min: number(range.min), max: number(range.max) });
		return '';
	});

	// done applies what's typed, if the value is one the step can hold.
	export function done() {
		if (problem) return;
		if (draft === value) cancel();
		else apply(draft);
	}

	// Enter and Esc belong to the cell while it edits, before the page's
	// shortcuts see them.
	function keys(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			e.preventDefault();
			e.stopPropagation();
			cancel();
		} else if (e.key === 'Enter' && !e.metaKey && !e.ctrlKey) {
			e.preventDefault();
			e.stopPropagation();
			done();
		}
	}

	// A slider and its field are one editor: moving between them isn't
	// leaving it.
	function left(e: FocusEvent) {
		const to = e.relatedTarget;
		if (!(to instanceof Node) || !(e.currentTarget as HTMLElement).contains(to)) cancel();
	}

	function captureKey(e: KeyboardEvent) {
		if (e.key === 'Escape') return keys(e);
		if (e.key === 'Tab') return;
		e.preventDefault();
		e.stopPropagation();
		const code = scanCodeOf(e);
		if (code !== undefined) apply(String(code));
	}
	function captureMouse(e: MouseEvent) {
		const code = mouseCodeOf(e);
		if (e.button === 0 || code === undefined) return;
		e.preventDefault();
		apply(String(code));
	}

	let colorHex = $derived(value === '' || Number.isNaN(Number(value)) ? '#000000' : hex(Number(value)).toLowerCase());
</script>

<span class="inline">
	{#if kind === 'toggle'}
		<!-- A toggle whose value isn't known yet: On or Off, chosen outright. -->
		<span class="choice" role="group" aria-label={t.value}>
			<button type="button" bind:this={field} onclick={() => apply('on')} onkeydown={keys}>{m.values.on}</button>
			<button type="button" onclick={() => apply('off')} onkeydown={keys}>{m.values.off}</button>
		</span>
	{:else if kind === 'slider'}
		<!-- The slider when the range is known, which applies as the pointer
		     lets go; the field, which takes an exact value, applies on Enter. -->
		<span class="pair" onfocusout={left}>
			{#if range}<Range bind:value={draft} min={range.min} max={range.max} step={range.step} label={t.slide} oncommit={done} onkeydown={keys} />{/if}
			<input
				bind:this={field}
				bind:value={draft}
				type="text"
				inputmode="decimal"
				class:bad={!!problem}
				class:exact={!!range}
				title={problem || (range ? t.range({ min: number(range.min), max: number(range.max), step: number(range.step) }) : undefined)}
				aria-label={t.value}
				aria-invalid={!!problem}
				onkeydown={keys}
				autocomplete="off"
			/>
		</span>
	{:else if kind === 'choose' && choices.length}
		{@const pick = fresh || !value}
		<select bind:this={field} aria-label={t.entry} onchange={(e) => apply(e.currentTarget.value)} onkeydown={keys} onblur={cancel}>
			{#if pick}<option value="" disabled selected>{t.chooseEntry}</option>{/if}
			{#each choices as c, i (i)}
				<option value={c.raw} selected={!pick && c.raw.toLowerCase() === value.toLowerCase()}>{c.text}{option?.default?.kind === 'number' && option.default.number === i && !/default/i.test(c.text) ? ` ${t.defaultMark}` : ''}</option>
			{/each}
			{#if value && !choices.some((c) => c.raw.toLowerCase() === value.toLowerCase())}<option value={value} selected>{t.notInMenu({ entry: value })}</option>{/if}
		</select>
	{:else if kind === 'choose' || kind === 'text'}
		<input bind:this={field} bind:value={draft} aria-label={kind === 'text' ? t.text : t.entry} onkeydown={keys} onblur={cancel} autocomplete="off" spellcheck="false" />
	{:else if kind === 'shortcut'}
		<button type="button" bind:this={field} class="capture" aria-label={t.pressKey} onkeydown={captureKey} onmousedown={captureMouse} oncontextmenu={(e) => e.preventDefault()} onblur={cancel}>
			{t.pressKey}<span class="muted">{keyName(Number(value))}</span>
		</button>
	{:else if kind === 'color'}
		<input bind:this={field} type="color" value={colorHex} aria-label={t.color} onchange={(e) => apply(String(parseInt(e.currentTarget.value.slice(1), 16)))} onkeydown={keys} />
	{/if}
</span>

<style>
	.inline {
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
		margin: -0.3em 0;
	}
	input:not([type='color']),
	select {
		width: 9rem;
		--chevron-inset: var(--space-2);
		padding-block: 0.15em;
		padding-left: var(--space-2);
		font: inherit;
	}
	.pair {
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
		width: 13rem;
	}
	.pair input.exact {
		width: 4.5rem;
		flex: none;
		font-variant-numeric: tabular-nums;
	}
	input[type='color'] {
		width: 2.5rem;
		height: 1.6rem;
		padding: 0;
		border: 1px solid var(--rule-strong);
		background: none;
	}
	/* A value the slider won't take: the field's own border says so. */
	input.bad {
		border-color: var(--warn);
	}
	.choice {
		display: inline-flex;
		border: 1px solid var(--accent);
		border-radius: var(--radius);
		overflow: hidden;
	}
	.choice button {
		background: none;
		border: 0;
		padding: 0.1em var(--space-2);
		font-size: var(--step--1);
		cursor: pointer;
	}
	.choice button + button {
		border-left: 1px solid var(--rule-strong);
	}
	.choice button:hover,
	.choice button:focus-visible {
		background: var(--select);
		outline: none;
	}
	.capture {
		display: inline-flex;
		align-items: baseline;
		gap: var(--space-2);
		padding: 0.15em var(--space-2);
		background: var(--glass-strong);
		border: 1px solid var(--accent);
		border-radius: var(--radius);
		font-size: var(--step--1);
		cursor: pointer;
	}
</style>
