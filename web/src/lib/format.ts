// Formatting shared across the app. In-game dates come from the server
// already formatted; these are for real-world times and counts.

// realTime is a real-world moment (Unix ms) in the viewer's own format.
export function realTime(ms: number): string {
	return new Date(ms).toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' });
}

export function count(n: number): string {
	return n.toLocaleString();
}

// plural is "1 memory" or "3 memories".
export function plural(n: number, one: string, many: string): string {
	return `${count(n)} ${n === 1 ? one : many}`;
}
