// The kinds of part a view may leave out of a set's completeness, in words:
// the counting menu, the overview's sentence and a set's panel say them the
// same way. Which Rebrickable categories each is: internal/catalog.
import { PartClass } from '$lib/gen/pile/v1/pile_pb';
import type { Counting } from '$lib/view.svelte';

export const classes = [
	{
		key: 'printed',
		class: PartClass.PRINTED,
		name: 'Printed parts and stickers',
		short: 'printed parts and stickers',
		is: 'A part with a print on it, or a sticker: a sorter rarely tells one from the plain part.'
	},
	{
		key: 'figureParts',
		class: PartClass.FIGURE,
		name: 'Minifigure parts',
		short: 'minifigure parts',
		is: 'Heads, torsos and arms, legs, hair and hats: what Rebrickable files under the minifig body.'
	},
	{
		key: 'accessories',
		class: PartClass.ACCESSORY,
		name: 'Minifigure accessories',
		short: 'minifigure accessories',
		is: 'Taps, cups, life rings, tools and weapons: Rebrickable files them with minifigs, but they are often part of the build.'
	}
] as const;

// What a view leaves out of completeness, as a list in words, or nothing.
export function leftOut(c: Counting): string {
	return classes
		.filter((k) => !c[k.key])
		.map((k) => k.short)
		.join(', ');
}
