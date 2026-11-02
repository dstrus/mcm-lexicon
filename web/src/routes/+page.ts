import { redirect } from '@sveltejs/kit';
import type { PageLoad } from './$types.ts';

// The app's home is Menus once something is open. Until then the layout
// shows the form that chooses a mod list, and this page is empty.
export const load: PageLoad = async ({ parent }) => {
	const { status } = await parent();
	if (status.source || status.catalogFile) redirect(307, '/menus');
	return {};
};
