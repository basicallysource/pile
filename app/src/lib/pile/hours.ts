// Hours of sorting as a span a person reads at a glance: hours, then days,
// months and years of a machine running nonstop. -1 is never.
export function span(hours: number): string {
	if (hours < 0) return 'never';
	if (hours < 1) return `${Math.max(1, Math.round(hours * 60))} min`;
	if (hours < 48) return `${Math.round(hours)} h`;
	const days = hours / 24;
	if (days < 60) return `${Math.round(days)} days`;
	const months = days / 30.44;
	if (months < 24) return `${Math.round(months)} months`;
	const years = days / 365.25;
	return years < 100 ? `${years.toFixed(1)} years` : `${Math.round(years).toLocaleString()} years`;
}

// The same as exact hours, for a tooltip.
export function hoursOf(hours: number): string {
	return hours < 0 ? 'never' : `${Math.round(hours).toLocaleString()} hours of sorting`;
}
