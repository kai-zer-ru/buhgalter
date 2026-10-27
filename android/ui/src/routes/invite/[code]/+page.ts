import { redirect } from '@sveltejs/kit';
import type { PageLoad } from './$types';

export const load: PageLoad = ({ params }) => {
	const code = params.code ?? '';
	redirect(308, `/settings/household/accept?code=${encodeURIComponent(code)}`);
};
