<!-- The controller button a key step takes. In Skyrim VR, a list of its
     own, since a select can't show pictures: the motion controllers'
     buttons by family, a Quest-style button with its picture beside its
     name, then the gamepad's own buttons. The button shows the choice, its
     picture too; arrows move in the open list, Enter or a click chooses,
     and Esc closes it. Anywhere else, the plain select of gamepad buttons. -->
<script lang="ts">
	import { asset } from '$app/paths';
	import { m } from '#lib/copy/messages.ts';
	import { gamepadButtons, vrFamilies, type Glyph } from '#lib/menus/keys.ts';
	import { clipped } from '#lib/ui/clipped.ts';
	import Diamond from '#lib/ui/Diamond.svelte';
	import { tick } from 'svelte';

	let {
		value,
		vr,
		label,
		placeholder,
		size = 'compact',
		choose,
		onkeydown
	}: {
		value: string; // the step's code, or "" for none
		vr: boolean;
		label: string; // for assistive technology: what the button is for
		placeholder: string;
		size?: 'compact' | 'full'; // a row's field, or a pane's
		// Takes the code chosen; false when it wasn't kept, and the select
		// goes back to what it showed.
		choose: (code: string) => Promise<boolean> | boolean | void;
		onkeydown?: (e: KeyboardEvent) => void; // while the list is closed
	} = $props();

	const k = m.keys;
	const padName = (code: number) => gamepadButtons.find((b) => b.code === code)?.name ?? String(code);
	// "A · A button" says it twice: a button named as the gamepad's is shown once.
	const same = (name: string, code: number) => padName(code) === name || padName(code).startsWith(name + ' ');
	const named = (name: string, code: number) => (same(name, code) ? padName(code) : k.vrAs({ button: name, pad: padName(code) }));

	// short is what the button shows of the choice: the controller's own
	// name, since the key box beside it names the gamepad's.
	type Entry = { code: string; text: string; short: string; glyph?: Glyph; group: number };
	const groups = [...vrFamilies.map((f) => f.name), k.gamepadGroup];
	const entries: Entry[] = [
		...vrFamilies.flatMap((f, g) => f.buttons.map((b) => ({ code: String(b.code), text: named(b.name, b.code), short: same(b.name, b.code) ? padName(b.code) : b.name, glyph: b.glyph, group: g }))),
		...gamepadButtons.map((b) => ({ code: String(b.code), text: b.name, short: b.name, group: groups.length - 1 }))
	];
	// The entry the value names: the first with its code, since families
	// share the gamepad's codes.
	let current = $derived(entries.findIndex((e) => e.code === value));
	let padValue = $derived(gamepadButtons.some((b) => String(b.code) === value) ? value : '');

	const src = (glyph: Glyph) => asset(`controller/${glyph}.png`);
	const srcset = (glyph: Glyph) => `${src(glyph)} 1x, ${asset(`controller/${glyph}@2x.png`)} 2x`;

	const id = $props.id();
	let button = $state<HTMLButtonElement>();
	let list = $state<HTMLElement>();
	let open = $state(false);
	let active = $state(0);

	async function show() {
		if (open || !list || !button) return;
		active = Math.max(0, current);
		list.showPopover();
		open = true;
		place();
		await tick();
		list.focus();
		reveal();
	}
	function hide(refocus = true) {
		if (!open) return;
		list?.hidePopover();
		open = false;
		if (refocus) button?.focus();
	}
	// Under the button, its right edge on the button's, or over it when
	// there's more room there.
	function place() {
		if (!list || !button) return;
		const r = button.getBoundingClientRect();
		const width = Math.max(r.width, list.offsetWidth);
		const below = window.innerHeight - r.bottom - 8;
		const above = r.top - 8;
		const up = list.offsetHeight > below && above > below;
		list.style.maxHeight = `${Math.max(160, up ? above : below)}px`;
		list.style.left = `${Math.max(8, Math.min(r.right - width, window.innerWidth - width - 8))}px`;
		list.style.top = up ? `${Math.max(8, r.top - 2 - Math.min(list.offsetHeight, above))}px` : `${r.bottom + 2}px`;
	}
	function reveal() {
		list?.querySelector(`#${id}-${active}`)?.scrollIntoView({ block: 'nearest' });
	}
	async function pick(i: number) {
		hide();
		const e = entries[i];
		if (e && e.code !== value) await choose(e.code);
	}

	function buttonKeys(e: KeyboardEvent) {
		if (e.key === 'ArrowDown' || e.key === 'ArrowUp' || e.key === 'Enter' || e.key === ' ') {
			e.preventDefault();
			e.stopPropagation();
			show();
			return;
		}
		onkeydown?.(e);
	}
	function listKeys(e: KeyboardEvent) {
		const move = (to: number) => {
			e.preventDefault();
			active = Math.max(0, Math.min(entries.length - 1, to));
			reveal();
		};
		switch (e.key) {
			case 'ArrowDown':
				return move(active + 1);
			case 'ArrowUp':
				return move(active - 1);
			case 'Home':
				return move(0);
			case 'End':
				return move(entries.length - 1);
			case 'PageDown':
				return move(active + 8);
			case 'PageUp':
				return move(active - 8);
			case 'Enter':
			case ' ':
				e.preventDefault();
				e.stopPropagation();
				pick(active);
				return;
			case 'Escape':
				e.preventDefault();
				e.stopPropagation();
				hide();
				return;
			case 'Tab':
				hide(false);
				return;
		}
	}
	// Leaving the list closes it; a click elsewhere leaves it.
	function left(e: FocusEvent) {
		const to = e.relatedTarget;
		if (to instanceof Node && (list?.contains(to) || button?.contains(to))) return;
		hide(false);
	}
	// The page scrolling under the open list closes it; the list's own
	// scrolling doesn't.
	$effect(() => {
		if (!open) return;
		const away = (e: Event) => {
			if (!(e.target instanceof Node && list?.contains(e.target))) hide(false);
		};
		window.addEventListener('scroll', away, { capture: true, passive: true });
		window.addEventListener('resize', away);
		return () => {
			window.removeEventListener('scroll', away, { capture: true });
			window.removeEventListener('resize', away);
		};
	});
</script>

{#if vr}
	<span class="picker {size}">
		<button
			type="button"
			bind:this={button}
			class="choice"
			class:empty={current < 0}
			aria-haspopup="listbox"
			aria-expanded={open}
			aria-controls="{id}-list"
			aria-label={current < 0 ? label : m.editor.padChosen({ label, choice: entries[current]!.text })}
			onclick={() => (open ? hide() : show())}
			onkeydown={buttonKeys}
		>
			{#if current >= 0 && entries[current]!.glyph}<img class="glyph" src={src(entries[current]!.glyph!)} srcset={srcset(entries[current]!.glyph!)} alt="" />{/if}
			<span class="text" use:clipped>{current < 0 ? placeholder : entries[current]!.short}</span>
		</button>
		<div bind:this={list} id="{id}-list" class="list" popover="manual" role="listbox" tabindex="-1" aria-label={label} aria-activedescendant="{id}-{active}" onkeydown={listKeys} onfocusout={left}>
			{#each groups as name, g (g)}
				<div role="group" aria-labelledby="{id}-g{g}">
					<div class="group caps" id="{id}-g{g}">{name}</div>
					{#each entries as e, i (i)}
						{#if e.group === g}
							<!-- svelte-ignore a11y_click_events_have_key_events -->
							<div id="{id}-{i}" class="option" class:active={i === active} role="option" aria-selected={i === current} tabindex="-1" onmousedown={(ev) => ev.preventDefault()} onmousemove={() => (active = i)} onclick={() => pick(i)}>
								<span class="mark">{#if i === current}<Diamond size={7} />{/if}</span>
								{#if e.glyph}<img class="glyph" src={src(e.glyph)} srcset={srcset(e.glyph)} alt="" />{:else}<span class="glyph none"></span>{/if}
								<span>{e.text}</span>
							</div>
						{/if}
					{/each}
				</div>
			{/each}
		</div>
	</span>
{:else}
	<select
		class="plain {size}"
		use:clipped
		aria-label={label}
		value={padValue}
		onchange={async (e) => {
			const el = e.currentTarget;
			if ((await choose(el.value)) === false) el.value = padValue;
		}}
		{onkeydown}
	>
		<option value="" disabled>{placeholder}</option>
		{#each gamepadButtons as b (b.code)}<option value={String(b.code)}>{b.name}</option>{/each}
	</select>
{/if}

<style>
	.picker {
		display: inline-flex;
		min-width: 0;
	}
	.picker.full {
		display: flex;
		width: 100%;
	}
	/* The button reads as the select it stands in for: the field's glass
	   and hairline, with the theme's chevron at its right. */
	.choice,
	select.plain {
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
		min-width: 0;
		width: 100%;
		background-color: var(--glass-strong);
		background-image: var(--chevron);
		background-repeat: no-repeat;
		background-position: right var(--space-2) center;
		background-size: 10px 6px;
		border: 1px solid var(--rule-strong);
		border-radius: var(--radius);
		font: inherit;
		color: inherit;
		text-align: left;
		cursor: pointer;
	}
	.compact .choice,
	select.plain.compact {
		width: 10rem;
		padding: 0.1em calc(var(--space-2) * 2 + 10px) 0.1em var(--space-2);
	}
	.full .choice,
	select.plain.full {
		padding: var(--space-2) calc(var(--space-3) * 2 + 10px) var(--space-2) var(--space-3);
		background-position: right var(--space-3) center;
	}
	.choice:hover,
	select.plain:hover {
		border-color: var(--accent);
	}
	/* Focus on the field's own edge, since a row clips a ring outside it. */
	.choice:focus-visible,
	select.plain:focus-visible {
		border-color: var(--accent);
		outline: 1px solid var(--accent);
		outline-offset: -1px;
	}
	.choice.empty {
		color: var(--muted);
	}
	.text {
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	/* The pictures are drawn at twice their size for sharp screens; a row's
	   is the height of its text, the list's a little taller. */
	.glyph {
		flex: none;
		height: 1.2em;
		width: auto;
	}
	.full .choice .glyph {
		height: 1.5em;
	}
	/* The open list, over everything: Pane Glass on smoke, as a dialog. */
	.list {
		position: fixed;
		text-align: left;
		inset: auto;
		margin: 0;
		min-width: 15rem;
		padding: var(--space-1) 0;
		overflow-y: auto;
		background: var(--smoke);
		border: 1px solid var(--rule-strong);
		border-radius: var(--radius);
		color: var(--ink);
		box-shadow: 0 8px 24px rgb(0 0 0 / 0.45);
	}
	.list:focus {
		outline: none;
	}
	.group {
		padding: var(--space-2) var(--space-3) var(--space-1);
		font-size: var(--step--1);
		color: var(--accent);
	}
	.option {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		padding: var(--space-1) var(--space-3) var(--space-1) var(--space-2);
		white-space: nowrap;
		cursor: pointer;
	}
	.option.active {
		background: var(--select);
	}
	/* Each picture in a box of one width, a grip's wider than a button's,
	   and the box kept where there's none, so the names line up. */
	.option .glyph {
		width: 2.4em;
		height: 1.6em;
		object-fit: contain;
	}
	.option .glyph.none {
		display: inline-block;
	}
	.mark {
		flex: none;
		display: inline-flex;
		width: 7px;
		color: var(--accent);
	}
</style>
