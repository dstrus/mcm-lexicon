import { request } from '#lib/api/client.ts';
import type { Menu, MenusReply, SearchReply, State } from '#lib/api/types/catalog.ts';

// The dependency the layout's menus load under, so a finished scan can
// reload them.
export const menusDependency = 'app:menus';

export function getMenus(fetchFn?: typeof fetch): Promise<MenusReply> {
	return request<MenusReply>('GET', '/api/menus', undefined, fetchFn);
}

export function getMenu(id: string, fetchFn?: typeof fetch): Promise<Menu> {
	return request<Menu>('GET', `/api/menus/${encodeURIComponent(id)}`, undefined, fetchFn);
}

export function search(query: string, fetchFn?: typeof fetch): Promise<SearchReply> {
	return request<SearchReply>('GET', `/api/search?q=${encodeURIComponent(query)}`, undefined, fetchFn);
}

export function rescan(): Promise<State> {
	return request<State>('POST', '/api/scan');
}

// menuHref is a menu's URL, at one of its pages; its first page is the
// menu's own URL.
export function menuHref(id: string, page = 0, item?: number): string {
	const base = `/menus/${encodeURIComponent(id)}${page > 0 ? `/${page}` : ''}`;
	return item === undefined ? base : `${base}?item=${item}`;
}
