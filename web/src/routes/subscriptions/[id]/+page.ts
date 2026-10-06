import { redirect } from '@sveltejs/kit';
import type { PageLoad } from './$types';

/** Notify / messenger links use `/subscriptions/{id}`; web edits inline on the list page. */
export const load: PageLoad = ({ params, url }) => {
	const search = new URLSearchParams(url.searchParams);
	search.set('edit', params.id);
	redirect(308, `/subscriptions?${search.toString()}`);
};
