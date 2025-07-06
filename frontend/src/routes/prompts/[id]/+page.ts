import type { PageLoad } from './$types';

export const load: PageLoad = async ({ params, fetch }) => {
  const id = params.id;
  const res = await fetch(`/api/prompt?id=${id}`);
  if (res.ok) {
    const prompt = await res.json();
    return { prompt };
  } else {
    return { prompt: null };
  }
};
