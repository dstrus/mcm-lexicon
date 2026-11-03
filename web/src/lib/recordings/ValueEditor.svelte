<!-- What a step sets, edited in its option's own terms: one control per
     kind of step, each with a keyboard path. Shared by adding a step and
     editing one. value is as the file writes it ("on", 25, a menu entry's
     raw text, a scan code, a color as a number). -->
<script lang="ts">
	import type { Option } from '#lib/api/types/catalog.ts';
	import type { Kind } from '#lib/api/types/recordings.ts';
	import { m } from '#lib/copy/messages.ts';
	import { keyName, mouseCodeOf, scanCodeOf } from '#lib/menus/keys.ts';
	import { hex, number } from '#lib/menus/values.ts';
	import Segmented from '#lib/ui/Segmented.svelte';
	import Switch from '#lib/ui/Switch.svelte';

	let {
		kind,
		option,
		value = $bindable(),
		side = $bindable(),
		problem = $bindable()
	}: {
		kind: Kind;
		option: Option | null | undefined;
		value: string;
		side: string;
		problem?: string; // why the value can't be written; "" when it can
	} = $props();

	const t = m.editor;

	let range = $derived(option?.slider);
	let choices = $derived(option?.choices ?? []);
	let known = $derived(choices.some((c) => c.raw.toLowerCase() === value.toLowerCase()));

	$effect(() => {
		problem = check(kind, value);
	});

	function check(k: Kind, v: string): string {
		switch (k) {
			case 'slider': {
				const n = Number(v.trim());
				if (v.trim() === '' || Number.isNaN(n)) return t.notANumber;
				if (range && (n < range.min - 1e-6 || n > range.max + 1e-6)) return t.outOfRange({ min: number(range.min), max: number(range.max) });
				return '';
			}
			case 'shortcut':
				return v === '' ? t.noKeyYet : '';
			case 'choose':
				return v === '' ? t.noChoice : '';
		}
		return '';
	}

	// A key is set by pressing it in the capture box. Tab and Esc keep their
	// jobs, and so does the save shortcut; a mouse button other than the
	// left one (which focuses the box) counts too.
	function captureKey(e: KeyboardEvent) {
		if (e.key === 'Tab' || e.key === 'Escape' || (e.key === 'Enter' && (e.metaKey || e.ctrlKey))) return;
		const code = scanCodeOf(e);
		e.preventDefault();
		e.stopPropagation();
		if (code !== undefined) value = String(code);
	}
	function captureMouse(e: MouseEvent) {
		const code = mouseCodeOf(e);
		if (e.button === 0 || code === undefined) return;
		e.preventDefault();
		value = String(code);
	}

	// A color as the game writes it is a number; the picker speaks #rrggbb.
	let colorHex = $derived(value === '' || Number.isNaN(Number(value)) ? '#000000' : hex(Number(value)).toLowerCase());
	const fromHex = (h: string) => String(parseInt(h.replace('#', ''), 16));
</script>

<div class="editor">
	{#if kind === 'toggle'}
		<span class="caps label">{t.value}</span>
		<span class="row">
			<Switch checked={value === 'on'} label={t.value} onchange={(on) => (value = on ? 'on' : 'off')} />
			<span>{value === 'on' ? m.values.on : m.values.off}</span>
		</span>
	{:else if kind === 'slider'}
		<label>
			<span class="caps label">{t.value}</span>
			<input type="number" bind:value={() => value, (v) => (value = v == null ? '' : String(v))} min={range?.min} max={range?.max} step={range?.step || 'any'} inputmode="decimal" autocomplete="off" />
		</label>
		{#if range}<p class="muted small">{t.range({ min: number(range.min), max: number(range.max), step: number(range.step) })}{#if option?.default}{' '}{t.defaultIs({ value: number(option.default.number, range.format) })}{/if}</p>{/if}
	{:else if kind === 'choose'}
		<label>
			<span class="caps label">{t.entry}</span>
			{#if choices.length}
				<select bind:value>
					{#each choices as c, i (i)}
						<option value={c.raw}>{c.text}{c.raw !== c.text ? ` (${c.raw})` : ''}</option>
					{/each}
					{#if value && !known}<option value={value}>{t.notInMenu({ entry: value })}</option>{/if}
				</select>
			{:else}
				<input bind:value autocomplete="off" spellcheck="false" />
			{/if}
		</label>
		{#if !choices.length}<p class="muted small">{t.entriesUnknown}</p>{/if}
	{:else if kind === 'shortcut'}
		<span class="caps label">{t.key}</span>
		<button type="button" class="capture" aria-label={t.pressKey} onkeydown={captureKey} onmousedown={captureMouse} oncontextmenu={(e) => e.preventDefault()}>
			{value === '' ? t.pressKey : keyName(Number(value))}{#if value !== ''}<span class="muted code">{value}</span>{/if}
		</button>
		<p class="muted small">{t.pressKeyHint}</p>
	{:else if kind === 'color'}
		<label class="color">
			<span class="caps label">{t.color}</span>
			<span class="row">
				<input type="color" value={colorHex} oninput={(e) => (value = fromHex(e.currentTarget.value))} />
				<span class="raw">{colorHex.toUpperCase()}</span>
			</span>
		</label>
	{:else if kind === 'text'}
		<label>
			<span class="caps label">{t.text}</span>
			<input bind:value autocomplete="off" spellcheck="false" />
		</label>
	{:else if kind === 'click'}
		<p class="muted">{t.clickOnly}</p>
		{#if !option?.label}
			<span class="caps label">{t.foundBy}</span>
			<Segmented bind:value={() => side || 'left', (v) => (side = v === 'left' ? '' : v)} options={[{ value: 'left', label: t.byLabel }, { value: 'right', label: t.byValue }]} label={t.foundBy} />
			<p class="muted">{t.foundByNote}</p>
		{/if}
	{/if}
	<!-- Nothing chosen yet isn't a mistake to point out: the button waits. -->
	{#if problem && !((kind === 'shortcut' || kind === 'choose') && value === '')}<p class="notice warn small" role="alert">{problem}</p>{/if}
</div>

<style>
	.editor {
		display: grid;
		gap: var(--space-2);
		justify-items: start;
	}
	label {
		display: grid;
		gap: var(--space-1);
		width: 100%;
	}
	.label {
		color: var(--muted);
		font-size: var(--step--1);
	}
	label input,
	label select {
		width: 100%;
	}
	.row {
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
	}
	.small {
		font-size: var(--step--1);
		margin: 0;
	}
	/* The box a key is pressed into: a field that listens rather than types. */
	.capture {
		display: flex;
		align-items: baseline;
		gap: var(--space-2);
		width: 100%;
		padding: var(--space-2) var(--space-3);
		background: var(--glass-strong);
		border: 1px solid var(--rule-strong);
		border-radius: var(--radius);
		text-align: left;
		cursor: pointer;
	}
	.capture:focus-visible {
		border-color: var(--accent);
		outline: none;
	}
	.code {
		font-family: var(--font-mono);
		font-size: var(--step--1);
	}
	.color .row {
		display: flex;
		align-items: center;
		gap: var(--space-3);
	}
	.color input {
		width: 3rem;
		height: 2rem;
		padding: 0;
		border: 1px solid var(--rule-strong);
		background: none;
	}
	.raw {
		font-family: var(--font-mono);
		font-size: var(--step--1);
	}
	.notice {
		max-width: 60ch;
	}
</style>
