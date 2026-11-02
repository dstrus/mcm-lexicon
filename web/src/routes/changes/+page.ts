import { getChanges, recordingsDependency } from '#lib/recordings/api.ts';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, depends }) => {
	depends(recordingsDependency);
	return { changes: await getChanges(fetch) };
};
