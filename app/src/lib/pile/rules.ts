// The filter's rules for what to hide, in words: the filter's switches and a
// set's panel ("hidden: almost all plain bricks") say them the same way.
import { Rule } from '$lib/gen/pile/v1/pile_pb';

export const rules = [
	{
		key: 'hideLooseBricks',
		rule: Rule.LOOSE_BRICKS,
		name: 'Loose bricks',
		hides: 'Buckets, brick boxes and service packs.',
		is: 'a set of loose bricks'
	},
	{
		key: 'hideMostlyBasic',
		rule: Rule.MOSTLY_BASIC,
		name: 'Basic bricks',
		hides: 'Sets almost all plain bricks, plates, tiles and slopes, like The Brick Apple.',
		is: 'almost all plain bricks'
	},
	{
		key: 'hideFewParts',
		rule: Rule.FEW_PARTS,
		name: 'Few parts',
		hides: 'Sets of ten or fewer different parts.',
		is: 'ten or fewer different parts'
	},
	{
		key: 'hideNothingUnusual',
		rule: Rule.NOTHING_UNUSUAL,
		name: 'Nothing unusual',
		hides: 'Sets whose rarest pieces are as common as those of the most common tenth of sets.',
		is: 'nothing unusual in it'
	}
] as const;

// Why the filter hides a set, in a few words.
export const hiddenAs: Record<Rule, string> = {
	[Rule.UNSPECIFIED]: '',
	[Rule.LOOSE_BRICKS]: rules[0].is,
	[Rule.MOSTLY_BASIC]: rules[1].is,
	[Rule.FEW_PARTS]: rules[2].is,
	[Rule.NOTHING_UNUSUAL]: rules[3].is,
	[Rule.KIND]: 'not the kind chosen',
	[Rule.THEME]: 'not the theme chosen',
	[Rule.LICENSED]: 'not licensed',
	[Rule.SIZE]: 'smaller than the size chosen'
};
