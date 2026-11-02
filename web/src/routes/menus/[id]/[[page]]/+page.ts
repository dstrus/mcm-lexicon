import { error } from '@sveltejs/kit';
import { ApiError } from '#lib/api/client.ts';
import { getMenu, menusDependency } from '#lib/menus/api.ts';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ params, fetch, depends }) => {
	depends(menusDependency);
	const at = params.page === undefined ? 0 : Number(params.page);
	try {
		const menu = await getMenu(params.id, fetch);
		if (!Number.isInteger(at) || at < 0 || (at > 0 && at >= menu.pages.length)) error(404, `There’s no page ${params.page} in ${menu.nameText}.`);
		return { menu, at };
	} catch (e) {
		if (e instanceof ApiError) error(e.status === 404 ? 404 : 500, e.message);
		throw e;
	}
};
