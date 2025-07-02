<script>
  import { createEventDispatcher } from 'svelte';
  let title = '';
  let company = '';
  let link = '';
  let loading = false;
  let error = '';

  const dispatch = createEventDispatcher();

  async function addJob() {
    loading = true;
    error = '';
    try {
      const res = await fetch('http://localhost:8080/api/jobs', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ title, company, link })
      });
      if (!res.ok) throw new Error('Failed to add job');
      title = '';
      company = '';
      link = '';
      dispatch('jobAdded');
    } catch (e) {
      error = e.message;
    }
    loading = false;
  }
</script>

<form on:submit|preventDefault={addJob}>
  <input placeholder="Job Title" bind:value={title} required />
  <input placeholder="Company" bind:value={company} required />
  <input placeholder="Link" bind:value={link} required />
  <button type="submit" disabled={loading}>Add Job</button>
  {#if error}
    <div style="color: red">{error}</div>
  {/if}
</form>