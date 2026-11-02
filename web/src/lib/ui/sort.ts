// Sorting tables by a column, with the column's own idea of order.

export interface Sort {
	key: string;
	descending: boolean;
}

// sorted returns items ordered by the value byKey[sort.key] gives. Ties keep
// their order, so a table falls back to its natural order.
export function sorted<T>(items: T[], sort: Sort, byKey: Record<string, (item: T) => string | number>): T[] {
	const value = byKey[sort.key];
	if (!value) return items;
	const dir = sort.descending ? -1 : 1;
	return items.toSorted((a, b) => {
		const x = value(a);
		const y = value(b);
		const c = typeof x === 'string' && typeof y === 'string' ? x.localeCompare(y, undefined, { sensitivity: 'base' }) : Number(x) - Number(y);
		return c * dir;
	});
}
