<!-- An option's value in a menu's row while a recording is open: the
     control for its kind, always there, so changing it records the option
     with no step before. A menu entry, a color, or a key records as soon
     as it's chosen; a number or text records on Enter, or on leaving the
     field once it's changed, and Esc puts it back. A slider's hairline shows while its
     number has focus. Toggles don't come here: the row's Switch flips them. -->
<script lang="ts">
	import { page } from '$app/state';
	import type { Status } from '#lib/api/types/api.ts';
	import type { Option } from '#lib/api/types/catalog.ts';
	import type { Kind } from '#lib/api/types/recordings.ts';
	import { m } from '#lib/copy/messages.ts';
	import { inVR, keyName, mouseCodeOf, scanCodeOf, watchGamepads } from '#lib/menus/keys.ts';
	import { hex, plain, short, snapped } from '#lib/menus/values.ts';
	import { clipped } from '#lib/ui/clipped.ts';
	import Range from '#lib/ui/Range.svelte';
	import { onDestroy } from 'svelte';
	import ControllerPicker from './ControllerPicker.svelte';

	let {
		kind,
		option,
		value,
		recorded,
		label,
		apply,
		onfocus
	}: {
		kind: Kind;
		option: Option;
		value: string; // the recorded value, else the option's own; "" for none
		recorded: boolean;
		label: string; // the option's name, which each control is named by
		// Records the value; false when it wasn't (no recording was chosen),
		// and the control goes back to what it showed.
		apply: (value: string) => Promise<boolean> | void;
		onfocus?: (focused: boolean, field: boolean) => void; // field: Enter applies there
	} = $props();

	const t = m.editor;
	let vr = $derived(inVR((page.data.status as Status | undefined)?.source?.game));
	let range = $derived(option.slider);
	let choices = $derived(option.choices ?? []);

	// What a field shows: the value, as its format shows a number, started
	// again whenever the value changes (recorded, or put back).
	const shownNumber = (v: string) => (v.trim() !== '' && !Number.isNaN(Number(v)) ? short(Number(v), range?.format) : v);
	let draft = $state('');
	$effect(() => {
		draft = kind === 'slider' ? shownNumber(value) : value;
	});

	let box = $state<HTMLElement>();
	export function focus() {
		box?.querySelector<HTMLElement>('input:not([type="range"]), select, button')?.focus();
	}
	// The page's hint strip shows Enter and Esc for a field with focus.
	export function commit() {
		if (kind === 'slider') commitNumber(true);
		else if (kind === 'text' || kind === 'choose') commitText(true);
	}
	export function revert() {
		draft = kind === 'slider' ? shownNumber(value) : value;
		if (box?.contains(document.activeElement)) (document.activeElement as HTMLElement).blur();
	}

	// A number applied is held to the range and moved to its nearest step.
	// A field left as it was records nothing, so passing through doesn't add
	// an option; Enter records what it shows, which is how an option not
	// recorded yet is added as it is (exactly, not as it's rounded here).
	function commitNumber(explicit = false) {
		if (draft.trim() === shownNumber(value)) {
			if (explicit && !recorded && value.trim() !== '') send(value);
			return;
		}
		const n = Number(draft.trim());
		if (draft.trim() === '' || Number.isNaN(n)) {
			draft = shownNumber(value);
			return;
		}
		const v = range ? snapped(n, range) : n;
		draft = plain(v, range?.format);
		if (recorded && value.trim() !== '' && Number(value) === v) return;
		send(draft);
	}
	function commitText(explicit = false) {
		if (draft === value) {
			if (explicit && !recorded) send(value);
			return;
		}
		send(draft);
	}
	// send records a value, and puts the control back as it was when it
	// wasn't recorded: a field's text, or a select's or color's own value,
	// which the page's unchanged value doesn't reset.
	async function send(v: string, el?: HTMLInputElement | HTMLSelectElement, was?: string) {
		if ((await apply(v)) !== false) return;
		draft = kind === 'slider' ? shownNumber(value) : value;
		if (el && was !== undefined) el.value = was;
	}
	function fieldKeys(e: KeyboardEvent, commit: () => void) {
		if (e.key === 'Enter' && !e.metaKey && !e.ctrlKey) {
			e.preventDefault();
			e.stopPropagation();
			commit();
		} else if (e.key === 'Escape') {
			e.preventDefault();
			e.stopPropagation();
			draft = kind === 'slider' ? shownNumber(value) : value;
			(e.currentTarget as HTMLElement).blur();
		}
	}
	// Esc leaves a control that applies as it's changed.
	function escapeLeaves(e: KeyboardEvent) {
		if (e.key !== 'Escape') return;
		e.preventDefault();
		e.stopPropagation();
		(e.currentTarget as HTMLElement).blur();
	}

	// A key is set by pressing it in the capture box; Tab, Esc, and the
	// save shortcut keep their jobs. A mouse button other than the left one
	// counts, and a gamepad the browser sees is watched while the box has
	// focus; the list beside it is the sure way to a controller button.
	function captureKey(e: KeyboardEvent) {
		if (e.key === 'Tab' || (e.key === 'Enter' && (e.metaKey || e.ctrlKey))) return;
		if (e.key === 'Escape') return escapeLeaves(e);
		e.preventDefault();
		e.stopPropagation();
		const code = scanCodeOf(e);
		if (code !== undefined) send(String(code));
	}
	function captureMouse(e: MouseEvent) {
		const code = mouseCodeOf(e);
		if (e.button === 0 || code === undefined) return;
		e.preventDefault();
		send(String(code));
	}
	let stopWatching: (() => void) | undefined;
	// The box asks for the key only while it has focus; at rest without a
	// key it shows a dash, as any value not known does.
	let capturing = $state(false);
	const watch = () => {
		capturing = true;
		stopWatching = watchGamepads((code) => send(String(code)));
	};
	const unwatch = () => {
		capturing = false;
		stopWatching?.();
		stopWatching = undefined;
	};
	onDestroy(unwatch);

	// Focus coming into the control, and leaving it altogether (moving
	// between a slider and its number isn't leaving).
	function focusIn(e: FocusEvent) {
		const el = e.target;
		onfocus?.(true, el instanceof HTMLInputElement && el.type !== 'color' && el.type !== 'range');
	}
	function focusOut(e: FocusEvent) {
		const to = e.relatedTarget;
		if (to instanceof Node && box?.contains(to)) return;
		onfocus?.(false, false);
	}

	let colorHex = $derived(value === '' || Number.isNaN(Number(value)) ? '#000000' : hex(Number(value)).toLowerCase());
	let known = $derived(choices.some((c) => c.raw.toLowerCase() === value.toLowerCase()));
	// What the selects show: the entry the value names, in the menu's own
	// case, or the value itself when the menu doesn't list it.
	let selected = $derived(choices.find((c) => c.raw.toLowerCase() === value.toLowerCase())?.raw ?? value);
</script>

<span class="live" class:unrecorded={!recorded} bind:this={box} onfocusin={focusIn} onfocusout={focusOut}>
	{#if kind === 'choose' && choices.length}
		<!-- The entry the option has, selected, with a placeholder when none
		     is known; choosing one records it. -->
		<select class="field" use:clipped aria-label={label} value={selected} onchange={(e) => send(e.currentTarget.value, e.currentTarget, selected)} onkeydown={escapeLeaves}>
			<option value="" disabled>{t.chooseEntry}</option>
			{#each choices as c, i (i)}
				<option value={c.raw}>{c.text}{option.default?.kind === 'number' && option.default.number === i && !/default/i.test(c.text) ? ` ${t.defaultMark}` : ''}</option>
			{/each}
			{#if value && !known}<option value={value}>{t.notInMenu({ entry: value })}</option>{/if}
		</select>
	{:else if kind === 'slider'}
		<!-- The number, right-aligned, with its hairline and diamond to the
		     left while it has focus; the knob records as it's let go. -->
		<span class="pair">
			{#if range}<Range bind:value={draft} min={range.min} max={range.max} step={range.step} label={t.slideFor({ option: label })} oncommit={() => commitNumber(true)} onkeydown={(e) => e.key === 'Escape' && escapeLeaves(e)} />{/if}
			<input class="field exact" type="text" inputmode="decimal" bind:value={draft} aria-label={label} onkeydown={(e) => fieldKeys(e, () => commitNumber(true))} onblur={() => commitNumber()} autocomplete="off" />
		</span>
	{:else if kind === 'shortcut'}
		<!-- The key box, with the controller list beside it while either has
		     focus. -->
		<span class="pair key">
			<button type="button" class="capture" aria-label={t.keyFor({ option: label, key: value ? keyName(Number(value)) : t.noKey })} onkeydown={captureKey} onmousedown={captureMouse} oncontextmenu={(e) => e.preventDefault()} onfocus={watch} onblur={unwatch}>
				{#if value}{keyName(Number(value))}{:else if capturing}<span class="muted">{t.pressKey}</span>{:else}—{/if}
			</button>
			<span class="pad"><ControllerPicker {value} {vr} label={t.padFor({ option: label })} placeholder={t.chooseButtonShort} choose={(code) => apply(code)} onkeydown={escapeLeaves} /></span>
		</span>
	{:else if kind === 'color'}
		<input type="color" value={colorHex} aria-label={label} onchange={(e) => send(String(parseInt(e.currentTarget.value.slice(1), 16)), e.currentTarget, colorHex)} onkeydown={escapeLeaves} />
	{:else}
		<input class="field" type="text" bind:value={draft} aria-label={label} onkeydown={(e) => fieldKeys(e, () => commitText(true))} onblur={() => commitText()} autocomplete="off" spellcheck="false" />
	{/if}
</span>

<style>
	.live {
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
		max-width: 100%;
	}
	/* The row's field: the smoked field and hairline, compact, so the row
	   keeps its height; named by element to outrank the theme's field rule. */
	.live input.field,
	.live select.field,
	.live .capture {
		min-width: 3.5rem;
		max-width: 100%;
		padding: 0.1em var(--space-2);
		background-color: var(--glass-strong);
		border: 1px solid var(--rule-strong);
		border-radius: var(--radius);
		font: inherit;
		color: inherit;
		text-align: right;
	}
	.live input.field:hover,
	.live select.field:hover,
	.live .capture:hover {
		border-color: var(--accent);
	}
	/* Focus on the field's own edge, since the row clips a ring outside it. */
	.live input.field:focus-visible,
	.live select.field:focus-visible,
	.live .capture:focus-visible {
		border-color: var(--accent);
		outline: 1px solid var(--accent);
		outline-offset: -1px;
	}
	/* Not recorded yet: the value the option has, in muted ink. */
	.unrecorded .field,
	.unrecorded .capture {
		color: var(--muted);
	}
	/* The theme's room for the chevron, which the compact padding above
	   would take away, so a long entry is cut off before the arrow. */
	.live select.field {
		--chevron-inset: var(--space-2);
		padding-block: 0.1em;
		padding-left: var(--space-2);
		padding-right: calc(var(--chevron-inset) * 2 + 10px);
		text-align: left;
		max-width: 11rem;
		cursor: pointer;
	}
	.pair {
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
	}
	/* The hairline and diamond, only while the number is being edited. */
	.pair :global(input[type='range']) {
		display: none;
		flex: none;
		width: 6.5rem;
		min-width: 0;
	}
	.pair:focus-within :global(input[type='range']) {
		display: block;
	}
	.live .pair input.exact {
		width: 4.5rem;
		flex: none;
		font-variant-numeric: tabular-nums;
	}
	.pair.key .pad {
		display: none;
	}
	/* Its open list counts too: focus passes through nowhere on its way
	   into the list, and a hidden list can't take it. */
	.pair.key:focus-within .pad,
	.pair.key .pad:has(:global(:popover-open)) {
		display: inline-flex;
	}
	.capture {
		cursor: pointer;
		white-space: nowrap;
	}
	input[type='color'] {
		width: 2.5rem;
		height: 1.6rem;
		padding: 0;
		border: 1px solid var(--rule-strong);
		background: none;
		cursor: pointer;
	}
</style>
