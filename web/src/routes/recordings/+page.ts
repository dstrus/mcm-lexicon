import { getDeletedBackups, recordingsDependency } from '#lib/recordings/api.ts';
import type { PageLoad } from './$types';

// The recordings themselves come with the layout; this is the ones the list
// no longer has, which a backup can bring back.
export const load: PageLoad = async ({ fetch, depends }) => {
	depends(recordingsDependency);
	try {
		return { deleted: (await getDeletedBackups(fetch)).recordings };
	} catch {
		return { deleted: [] };
	}
};
