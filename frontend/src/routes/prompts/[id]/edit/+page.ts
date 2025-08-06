import type { PageLoad } from './$types';
import { BASE_API_URL } from '../../../../lib/config';

export const load: PageLoad = async ({ params, fetch }) => {
  const id = params.id;
  const res = await fetch(BASE_API_URL+`/api/prompts?id=${id}`);
  if (res.ok) {
    const prompt = await res.json();
    return { prompt };
  } else {
    return { prompt: null };
  }
};
