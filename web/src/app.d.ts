// See https://svelte.dev/docs/kit/types#app.d.ts
declare global {
	namespace App {
		// The message shown when a page fails to load (see +error.svelte).
		interface Error {
			message: string;
		}
		// What an entry in the browser's history was pushed for: the address
		// parameters it set, so leaving them can go back (see address.svelte.ts).
		interface PageState {
			pushed?: string[];
		}
	}
}

export {};
