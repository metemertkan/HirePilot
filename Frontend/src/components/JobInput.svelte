<script>
  import { createEventDispatcher } from 'svelte';
  let title = '';
  let company = '';
  let link = '';
  const dispatch = createEventDispatcher();

  function addJob() {
    if (title && company && link) {
      dispatch('add', { title, company, link });
      title = '';
      company = '';
      link = '';
      applied = false;
    }
  }
</script>

<div class="job-input">
  <input placeholder="Job Title" bind:value={title} />
  <input placeholder="Company" bind:value={company} />
  <input placeholder="Link" bind:value={link}/>
  <button on:click={addJob}>Add Job</button>
</div>

<style>
.job-input {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  max-width: 400px;
}
</style>
