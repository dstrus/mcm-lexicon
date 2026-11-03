// The column's list of menus, as the menu screen's arrow keys see it: the
// layout keeps the menus it lists here, filtered and in its order, and the
// menu screen says when the keys are in the list, so the layout marks the
// menu on screen as the keys' place.
export const column = $state({
	ids: [] as string[], // the menus the column lists, in order
	active: false // the arrow keys are in the column's list
});
