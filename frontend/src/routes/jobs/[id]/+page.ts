import type { PageLoad } from './$types';

export const load: PageLoad = async ({ params, fetch }) => {
    const res = await fetch(`http://localhost:8080/api/jobs/${params.id}`);
    if (!res.ok) {
        return { job: null };
    }
    const job = await res.json();
    return { job };
};