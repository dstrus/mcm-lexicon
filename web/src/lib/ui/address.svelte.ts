// What a page's pane is doing lives in the page's address: which entry is
// open (?item=) and what the pane is doing with it (?mode=edit). Each state
// has its own URL, so it can be linked to or reloaded, and its own place in
// the browser's history, so Back closes a form or returns to the entry
// before it.
import { goto } from '$app/navigation';
import { navigating, page } from '$app/state';

// Changes to the address's parameters; null drops one.
export type Params = Record<string, string | null | undefined>;

// The address as the pane sees it: after a shallow change, the router keeps
// the target apart from the page's own URL.
function address(): typeof page.url {
	return page.shallow?.url ?? page.url;
}

export function param(name: string): string | null {
	return address().searchParams.get(name);
}

// The router applies a change asynchronously, so changes queue: each reads
// the address as the one before it left it.
let queue: Promise<void> = Promise.resolve();
function then(change: () => void | Promise<void>): Promise<void> {
	return (queue = queue.then(change, change));
}

// setParams changes the address. Pushing adds a history entry: an entry
// opening, a form starting. Replacing changes the entry in place: the
// selection moving under the keys, a choice within one view. A change that
// builds on the address (the next row after the selected one) is given as
// a function, so it reads the address once the changes before it are in.
export function setParams(changes: Params | ((current: URLSearchParams) => Params), how: 'push' | 'replace' = 'push'): Promise<void> {
	return then(() => {
		const url = new URL(address().href);
		const made = typeof changes === 'function' ? changes(url.searchParams) : changes;
		for (const [name, value] of Object.entries(made)) {
			if (value == null) url.searchParams.delete(name);
			else url.searchParams.set(name, value);
		}
		if (url.href === address().href) return;
		const touched = Object.keys(made);
		// The entry remembers which parameters it was pushed for, so leaving
		// them can go back instead of leaving a dead entry behind.
		const pushed = how === 'push' ? touched : (page.state.pushed ?? []).filter((n) => !touched.includes(n));
		return goto(url, { shallow: true, replace: how === 'replace', state: { pushed } });
	});
}

// leave drops the parameters a form or view is done with. When the current
// entry was pushed for exactly those, it goes back to the entry before, so
// Back never lands on a form that's already closed.
export function leave(...names: string[]): Promise<void> {
	return then(() => {
		const present = names.filter((n) => address().searchParams.has(n));
		if (!present.length) return;
		const pushed = page.state.pushed ?? [];
		const exact = pushed.length === present.length && present.every((n) => pushed.includes(n));
		if (exact) {
			// Settled once the router has taken the step back. It treats the
			// step as a navigation, and a reload begun before that ends is
			// dropped, so this waits for the router to be idle.
			return new Promise((done) => {
				addEventListener(
					'popstate',
					async () => {
						const pause = (ms: number) => new Promise((r) => setTimeout(r, ms));
						await pause(0);
						const until = Date.now() + 2000;
						while (navigating.to && Date.now() < until) await pause(10);
						done();
					},
					{ once: true }
				);
				history.back();
			});
		}
		const url = new URL(address().href);
		for (const n of present) url.searchParams.delete(n);
		return goto(url, { shallow: true, replace: true, state: { pushed: pushed.filter((n) => !present.includes(n)) } });
	});
}
