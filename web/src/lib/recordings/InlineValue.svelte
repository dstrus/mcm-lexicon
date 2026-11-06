<!-- A step's value edited in its table cell: the smallest control for its
     kind, focused on arrival. Enter applies and Esc cancels; a menu entry,
     a color, or a key pressed applies as soon as it's chosen. A slider's
     number applies on leaving it too, held to the range and moved to the
     nearest step. Toggles don't come here: their cell's switch flips them. -->
<script lang="ts">
	import type { Option } from '#lib/api/types/catalog.ts';
	import type { Kind } from '#lib/api/types/recordings.ts';
	import { m } from '#lib/copy/messages.ts';
	import { gamepadButtons, keyName, mouseCodeOf, scanCodeOf, watchGamepads } from '#lib/menus/keys.ts';
	import { hex, number, plain, snapped } from '#lib/menus/values.ts';
	import Range from '#lib/ui/Range.svelte';
	import { onDestroy, onMount, tick } from 'svelte';

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
		// The option isn't recorded yet: Enter records the entry the menu
		// shows, since choosing the one already selected changes nothing.
		fresh?: boolean;
		apply: (value: string) => void;
		cancel: () => void;
	} = $props();

	const t = m.editor;
	let draft = $state('');
	let field = $state<HTMLElement>();
	// The field is filled before its text is selected, so typing replaces it.
	// A slider's number shows the decimals its format gives: 5.0, not 5.
	onMount(async () => {
		draft = kind === 'slider' && value.trim() !== '' && !Number.isNaN(Number(value)) ? plain(Number(value), option?.slider?.format) : value;
		await tick();
		field?.focus();
		if (field instanceof HTMLInputElement && field.type !== 'color') field.select();
	});

	let range = $derived(option?.slider);
	let choices = $derived(option?.choices ?? []);
	// A slider's number past the range is held to it when applied, so only
	// text that isn't a number can't be.
	let problem = $derived.by(() => {
		if (kind !== 'slider') return '';
		const n = Number(draft.trim());
		if (draft.trim() === '' || Number.isNaN(n)) return t.notANumber;
		return '';
	});

	// The editor ends once: applied or cancelled, the focus leaving as it
	// closes doesn't apply it again.
	let ended = false;
	const end = (f: () => void) => {
		if (ended) return;
		ended = true;
		f();
	};

	// done applies what's typed, if the value is one the step can hold; a
	// menu's, the entry its select shows.
	export function done() {
		if (kind === 'choose' && field instanceof HTMLSelectElement) {
			if (field.value === '') return;
			if (field.value === value && !fresh) cancel();
			else apply(field.value);
			return;
		}
		if (problem) return;
		if (kind === 'slider') {
			const n = range ? snapped(Number(draft), range) : Number(draft);
			draft = plain(n, range?.format);
			const out = draft;
			if (value.trim() !== '' && Number(value) === n) end(cancel);
			else end(() => apply(out));
			return;
		}
		if (draft === value) cancel();
		else apply(draft);
	}

	// Enter and Esc belong to the cell while it edits, before the page's
	// shortcuts see them.
	function keys(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			e.preventDefault();
			e.stopPropagation();
			end(cancel);
		} else if (e.key === 'Enter' && !e.metaKey && !e.ctrlKey) {
			e.preventDefault();
			e.stopPropagation();
			done();
		}
	}

	// A slider and its field are one editor: moving between them isn't
	// leaving it. Leaving applies the number, unless it isn't one.
	function left(e: FocusEvent) {
		const to = e.relatedTarget;
		if (ended || (to instanceof Node && (e.currentTarget as HTMLElement).contains(to))) return;
		if (problem) end(cancel);
		else done();
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
	// A gamepad the browser sees is watched while the box has focus; the
	// list beside it is the sure way to a controller button. Choosing one
	// in the list applies it, as a key pressed does.
	let stopWatching: (() => void) | undefined;
	const watch = () => (stopWatching = watchGamepads((code) => end(() => apply(String(code)))));
	const unwatch = () => {
		stopWatching?.();
		stopWatching = undefined;
	};
	onDestroy(unwatch);
	// Moving from the box to the list isn't leaving the editor.
	function leftKey(e: FocusEvent) {
		const to = e.relatedTarget;
		if (to instanceof Node && (e.currentTarget as HTMLElement).contains(to)) return;
		end(cancel);
	}

	let colorHex = $derived(value === '' || Number.isNaN(Number(value)) ? '#000000' : hex(Number(value)).toLowerCase());
</script>

<span class="inline" class:level={kind === 'slider'}>
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
		<!-- The entry the option has, selected; Enter records it, and choosing
		     another records that. The placeholder heads the list, chosen
		     only when no entry is known. -->
		<select bind:this={field} aria-label={t.entry} onchange={(e) => apply(e.currentTarget.value)} onkeydown={keys} onblur={cancel}>
			<option value="" disabled selected={!value}>{t.chooseEntry}</option>
			{#each choices as c, i (i)}
				<option value={c.raw} selected={!!value && c.raw.toLowerCase() === value.toLowerCase()}>{c.text}{option?.default?.kind === 'number' && option.default.number === i && !/default/i.test(c.text) ? ` ${t.defaultMark}` : ''}</option>
			{/each}
			{#if value && !choices.some((c) => c.raw.toLowerCase() === value.toLowerCase())}<option value={value} selected>{t.notInMenu({ entry: value })}</option>{/if}
		</select>
	{:else if kind === 'choose' || kind === 'text'}
		<input bind:this={field} bind:value={draft} aria-label={kind === 'text' ? t.text : t.entry} onkeydown={keys} onblur={cancel} autocomplete="off" spellcheck="false" />
	{:else if kind === 'shortcut'}
		<span class="pair key" onfocusout={leftKey}>
			<button type="button" bind:this={field} class="capture" aria-label={t.pressKey} onkeydown={captureKey} onmousedown={captureMouse} oncontextmenu={(e) => e.preventDefault()} onfocus={watch} onblur={unwatch}>
				{t.pressKey}<span class="muted">{keyName(Number(value))}</span>
			</button>
			<select class="pad" aria-label={t.controllerButton} title={t.controllerNote} value={gamepadButtons.some((b) => String(b.code) === value) ? value : ''} onchange={(e) => end(() => apply(e.currentTarget.value))} onkeydown={keys}>
				<option value="" disabled>{t.chooseButtonShort}</option>
				{#each gamepadButtons as b (b.code)}<option value={String(b.code)}>{b.name}</option>{/each}
			</select>
		</span>
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
	/* A slider's number is the resting value's field, the same size, so the
	   row neither jumps nor clips it as editing starts. */
	.inline.level {
		margin: 0;
	}
	input:not([type='color']):not([type='range']),
	select {
		width: 9rem;
		--chevron-inset: var(--space-2);
		padding-block: 0.15em;
		padding-left: var(--space-2);
		font: inherit;
	}
	/* As SkyUI's slider dialog reads, and as the site's demo draws it: the
	   hairline and diamond to the left, the number right-aligned in its
	   field, which keeps its place at the cell's right edge. */
	.pair {
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
	}
	.pair :global(input[type='range']) {
		flex: none;
		width: 6.5rem;
		min-width: 0;
	}
	.pair input.exact {
		width: 4.5rem;
		flex: none;
		padding: 1.5px var(--space-2);
		text-align: right;
		font-variant-numeric: tabular-nums;
	}
	input[type='color'] {
		width: 2.5rem;
		height: 1.6rem;
		padding: 0;
		border: 1px solid var(--rule-strong);
		background: none;
	}
	/* Focus on the field's own edge, since the row clips a ring drawn
	   outside it. */
	input:not([type='color']):not([type='range']):focus-visible,
	select:focus-visible {
		border-color: var(--accent);
		outline: 1px solid var(--accent);
		outline-offset: -1px;
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
	/* The box and the list, which drops under the box where the cell is
	   narrow rather than squeezing the table's other columns. */
	.pair.key {
		flex-wrap: wrap;
		max-width: 15rem;
	}
	.pair select.pad {
		width: 10rem;
		flex: none;
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
