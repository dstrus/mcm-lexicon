// The selected row of a list, moved with ↑ and ↓ like the game's menus.
// Rows are found by key, so the selection survives the list being sorted,
// filtered or reloaded. A list beside a pane keeps its selection in the
// page's address, so the open entry has its own URL and place in history.
import { tick } from 'svelte';
import { param, setParams } from './address.svelte.ts';

export class Selection<T> {
	#own = $state<string | null>(null); // the key, when it isn't kept in the address

	constructor(
		private items: () => T[],
		private keyOf: (item: T) => string,
		private options: {
			marker?: string; // how the selected row is marked, for scrolling to it
			param?: string; // the address parameter that holds the key
		} = {}
	) {}

	get key(): string | null {
		return this.options.param ? param(this.options.param) : this.#own;
	}

	// Setting the key moves the selection in place, without a history entry.
	set key(key: string | null) {
		if (this.options.param) setParams({ [this.options.param]: key }, 'replace');
		else this.#own = key;
	}

	get current(): T | undefined {
		const items = this.items();
		return items.find((i) => this.keyOf(i) === this.key) ?? items[0];
	}

	is(item: T): boolean {
		return this.current !== undefined && this.keyOf(item) === this.keyOf(this.current);
	}

	// select opens an item, as a click does: in the address, that's its own
	// history entry.
	select(item: T) {
		if (this.options.param) setParams({ [this.options.param]: this.keyOf(item) });
		else this.#own = this.keyOf(item);
	}

	// move selects the row by rows away, and scrolls it into view.
	async move(by: number) {
		// From the key as it stands once earlier moves are in, so a run of
		// key presses steps once each.
		const step = (key: string | null): string | null => {
			const items = this.items();
			const from = items.find((i) => this.keyOf(i) === key) ?? items[0];
			const at = from ? items.indexOf(from) : -1;
			const next = items[Math.min(items.length - 1, Math.max(0, at + by))];
			return next ? this.keyOf(next) : key;
		};
		const { param } = this.options;
		if (param) await setParams((q) => ({ [param]: step(q.get(param)) }), 'replace');
		else this.#own = step(this.#own);
		await tick();
		document.querySelector(this.options.marker ?? '[aria-selected="true"]')?.scrollIntoView({ block: 'nearest' });
	}
}
