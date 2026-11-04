import { error } from '@sveltejs/kit';
import { ApiError } from '#lib/api/client.ts';
import { getBackups, recordingsDependency } from '#lib/recordings/api.ts';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ params, fetch, depends }) => {
	depends(recordingsDependency);
	try {
		return { backups: await getBackups(params.name, fetch) };
	} catch (e) {
		if (e instanceof ApiError) error(e.status === 404 ? 404 : 500, e.message);
		throw e;
	}
};
