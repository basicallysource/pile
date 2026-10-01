// How numbers and times read on every page.

export function percent(share: number): string {
	return `${Math.round(share * 100)}%`;
}

export function count(n: number): string {
	return n.toLocaleString();
}

const day = new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric' });
const dayTime = new Intl.DateTimeFormat(undefined, {
	month: 'short',
	day: 'numeric',
	hour: 'numeric',
	minute: '2-digit'
});

export function dayOf(unix: bigint | number): string {
	return day.format(new Date(Number(unix) * 1000));
}

export function dayTimeOf(unix: bigint | number): string {
	return dayTime.format(new Date(Number(unix) * 1000));
}
