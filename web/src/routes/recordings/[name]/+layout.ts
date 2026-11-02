import { error } from '@sveltejs/kit';
import { ApiError } from '#lib/api/client.ts';
import { getRecording, recordingsDependency } from '#lib/recordings/api.ts';
import type { LayoutLoad } from './$types';

export const load: LayoutLoad = async ({ params, fetch, depends }) => {
	depends(recordingsDependency);
	try {
		return { detail: await getRecording(params.name, fetch) };
	} catch (e) {
		if (e instanceof ApiError) error(e.status === 404 ? 404 : 500, e.message);
		throw e;
	}
};
