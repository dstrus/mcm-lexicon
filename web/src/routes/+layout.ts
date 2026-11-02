import { getStatus } from '#lib/api/status.ts';
import type { LayoutLoad } from './$types.ts';

// MCM Lexicon runs entirely in the browser: the Go server only serves the
// built files and the API, so nothing is rendered or prerendered ahead of time.
export const ssr = false;
export const prerender = false;

export const load: LayoutLoad = async ({ fetch }) => {
	const status = await getStatus(fetch);
	return { status };
};
