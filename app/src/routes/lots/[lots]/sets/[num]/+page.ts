// A set's own address from before sets opened in a sheet: links and open tabs
// still land on it, so it goes to the lots' sets with that set open.
import { redirect } from '@sveltejs/kit';

export function load({ params }) {
	redirect(308, `/lots/${params.lots}/sets?set=${encodeURIComponent(params.num)}`);
}
