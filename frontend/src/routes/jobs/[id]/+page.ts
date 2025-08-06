import type { PageLoad } from './$types';
import { BASE_API_URL } from '../../../lib/config';


export const load: PageLoad = async ({ params, fetch }) => {
    const res = await fetch(BASE_API_URL+`/api/jobs/${params.id}`);
    if (!res.ok) {
        return { job: null };
    }
    const job = await res.json();
    return { job };
};