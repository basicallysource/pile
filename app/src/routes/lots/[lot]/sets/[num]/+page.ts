// A set's own address from before sets opened in a sheet: links and open tabs
// still land on it, so it goes to the lot's sets with that set open.
import { redirect } from '@sveltejs/kit';

export function load({ params }) {
	redirect(308, `/lots/${params.lot}/sets?set=${encodeURIComponent(params.num)}`);
}
