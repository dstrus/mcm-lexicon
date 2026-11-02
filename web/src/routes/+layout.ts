import { getStatus } from '#lib/api/status.ts';
import type { MenusReply } from '#lib/api/types/catalog.ts';
import { getMenus, menusDependency } from '#lib/menus/api.ts';
import type { ListReply } from '#lib/api/types/recordings.ts';
import { getRecordings, recordingsDependency } from '#lib/recordings/api.ts';
import type { LayoutLoad } from './$types.ts';

// MCM Lexicon runs entirely in the browser: the Go server only serves the
// built files and the API, so nothing is rendered or prerendered ahead of time.
export const ssr = false;
export const prerender = false;

export const load: LayoutLoad = async ({ fetch, depends }) => {
	depends(menusDependency);
	depends(recordingsDependency);
	const status = await getStatus(fetch);
	// The menus, once there's a catalog to list them from. A first scan
	// still running doesn't hold the page up; it reloads them when it ends.
	let menus: MenusReply | null = null;
	let menusError = '';
	let recordings: ListReply | null = null;
	const open = !!status.source || !!status.catalogFile;
	if (open && (status.scan.scannedAt || !status.scan.scanning)) {
		try {
			menus = await getMenus(fetch);
		} catch (e) {
			menusError = e instanceof Error ? e.message : String(e);
		}
		// The recordings, for the sections column and the top bar; a problem
		// reading them shows on their own page.
		recordings = await getRecordings(fetch).catch(() => null);
	}
	return { status, menus, menusError, recordings };
};
