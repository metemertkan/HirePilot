<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  export let data;
  let prompt = data.prompt;
  let error = '';

  async function updatePrompt() {
    error = '';
    const res = await fetch(`/api/prompt`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ id: prompt.id, name: prompt.name, prompt: prompt.prompt })
    });
    if (res.ok) {
      goto(`/prompts/${prompt.id}`);
    } else {
      error = 'Failed to update prompt.';
    }
  }
</script>

<h1>Edit Prompt</h1>
{#if prompt}
  <form on:submit|preventDefault={updatePrompt}>
    <label>
      Name:
      <input type="text" bind:value={prompt.name} required />
    </label>
    <br>
    <label>
      Prompt:
      <textarea bind:value={prompt.prompt} required rows="6"></textarea>
    </label>
    <br>
    <button type="submit">Update</button>
    {#if error}
      <p style="color:red">{error}</p>
    {/if}
  </form>
{:else}
  <p>Prompt not found.</p>
{/if}
