// What can change while a page is open: whether Skyrim is running, and so
// whether anything can be saved. The root layout keeps it current.
import { getStatus } from './status.ts';

export const live = $state({ gameRunning: false });

const every = 10_000;

// watchGame starts keeping live.gameRunning current, and returns a function
// that stops it.
export function watchGame(initial: boolean): () => void {
	live.gameRunning = initial;
	const timer = setInterval(async () => {
		try {
			live.gameRunning = (await getStatus()).gameRunning;
		} catch {
			// MCM Lexicon may have stopped; the next request says so.
		}
	}, every);
	return () => clearInterval(timer);
}
