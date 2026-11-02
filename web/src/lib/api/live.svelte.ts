// What can change while a page is open: whether Skyrim is running (and so
// whether anything can be saved), and the catalog's scan. The root layout
// keeps it current, more often while a scan runs.
import type { Status } from './types/api.ts';
import type { State } from './types/catalog.ts';
import { getStatus } from './status.ts';

export const live = $state<{ gameRunning: boolean; scan: State }>({
	gameRunning: false,
	scan: { scanning: false, scannedAt: '', error: '', menus: 0 }
});

const every = 10_000;
const whileScanning = 1_000;

// watchStatus starts keeping live current from the status the page loaded
// with, calling onScanned when a scan finishes; it returns a function that
// stops it.
export function watchStatus(initial: Status, onScanned: () => void): () => void {
	live.gameRunning = initial.gameRunning;
	live.scan = initial.scan;
	let timer: ReturnType<typeof setTimeout>;
	let stopped = false;
	async function tick() {
		try {
			const st = await getStatus();
			const finished = live.scan.scanning && !st.scan.scanning;
			const changed = st.scan.scannedAt !== live.scan.scannedAt;
			live.gameRunning = st.gameRunning;
			live.scan = st.scan;
			if (finished || changed) onScanned();
		} catch {
			// MCM Lexicon may have stopped; the next request says so.
		}
		if (!stopped) timer = setTimeout(tick, live.scan.scanning ? whileScanning : every);
	}
	timer = setTimeout(tick, live.scan.scanning ? whileScanning : every);
	return () => {
		stopped = true;
		clearTimeout(timer);
	};
}

// scanStarted marks a scan as running now, so the page polls quickly until
// it ends.
export function scanStarted(state: State) {
	live.scan = state;
}
