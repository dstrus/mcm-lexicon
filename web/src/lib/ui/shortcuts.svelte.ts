// Keyboard shortcuts, and the hint bar that shows them: the game's button
// hints, backed by real keys. A page registers the shortcuts that apply to
// it; the root layout listens for keys and draws the hint bar.
import { onDestroy } from 'svelte';

export interface Shortcut {
	key: string; // KeyboardEvent.key, e.g. "e", "Delete", "/"
	label: string; // what it does, for the hint bar
	shown?: string; // the key as shown, when not key itself
	run: () => void;
	hidden?: boolean; // works, but isn't shown in the hint bar
	disabled?: boolean;
	command?: boolean; // pressed with Ctrl (⌘ on a Mac), and works while typing
}

// onMac is whether the command key is ⌘ rather than Ctrl.
export const onMac = typeof navigator !== 'undefined' && /Mac|iPhone|iPad/.test(navigator.platform);

// deleteShortcuts are the keys that delete: Delete, and on a Mac also the
// key labelled delete (⌫), which sends Backspace.
export function deleteShortcuts(s: Omit<Shortcut, 'key' | 'shown'>): Shortcut[] {
	if (!onMac) return [{ ...s, key: 'Delete', shown: 'Del' }];
	return [
		{ ...s, key: 'Backspace', shown: '⌫' },
		{ ...s, key: 'Delete', hidden: true }
	];
}

// The registered shortcut sets, in registration order, kept outside
// reactive state: a page leaving reads state as it was when it mounted, so
// filtering a reactive list in onDestroy would drop the page arriving.
// changes is bumped whenever the set changes, so active() recomputes.
const registered = new Map<number, () => Shortcut[]>();
let changes = $state(0);
let next = 0;

function changed() {
	queueMicrotask(() => changes++);
}

// active is every shortcut in force, later registrations winning a key.
export function active(): Shortcut[] {
	void changes;
	const byKey = new Map<string, Shortcut>();
	for (const shortcuts of registered.values()) for (const s of shortcuts()) byKey.set(`${s.command ? 'cmd+' : ''}${s.key.toLowerCase()}`, s);
	return [...byKey.values()];
}

// useShortcuts registers shortcuts for as long as the calling component
// exists. shortcuts is read whenever keys are pressed or hints are drawn,
// so it can depend on the component's state.
export function useShortcuts(shortcuts: () => Shortcut[]) {
	const id = next++;
	registered.set(id, shortcuts);
	changed();
	onDestroy(() => {
		registered.delete(id);
		changed();
	});
}

function typing(target: EventTarget | null): boolean {
	return (
		target instanceof HTMLElement &&
		(target.isContentEditable || ['INPUT', 'TEXTAREA', 'SELECT'].includes(target.tagName))
	);
}

// handleKey runs the shortcut for a key press, if there is one. While
// typing, only Escape and command shortcuts count, so typed letters stay
// letters.
export function handleKey(e: KeyboardEvent) {
	if (e.altKey) return;
	// An open dialog (quick find) handles its own keys, Esc included.
	if (e.target instanceof Element && e.target.closest('dialog[open]')) return;
	const command = onMac ? e.metaKey : e.ctrlKey;
	if (!command && (e.metaKey || e.ctrlKey)) return;
	if (!command && typing(e.target) && e.key !== 'Escape') return;
	// Enter on a focused link or button does what that link or button does.
	if (!command && e.key === 'Enter' && e.target instanceof Element && e.target.closest('a[href], button')) return;
	// Esc in a text field leaves the field; the next Esc runs the shortcut.
	// A textarea is different: it's the editor the shortcut cancels.
	if (e.key === 'Escape' && (e.target instanceof HTMLInputElement || e.target instanceof HTMLSelectElement)) {
		e.target.blur();
		return;
	}
	const s = active().find((x) => x.key.toLowerCase() === e.key.toLowerCase() && !!x.command === command);
	if (!s || s.disabled) return;
	e.preventDefault();
	s.run();
}
