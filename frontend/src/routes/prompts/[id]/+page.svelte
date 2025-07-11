<script lang="ts">
  export let data;
  let prompt = data.prompt;
</script>

<h1>Prompt Details</h1>
{#if prompt}
  <div>
    <strong>ID:</strong> {prompt.id}<br>
    <strong>Name:</strong> {prompt.name}<br>
    <strong>Prompt:</strong> <pre>{prompt.prompt}</pre>
    <strong>CV Generation Default:</strong> {prompt.cvGenerationDefault ? 'Yes' : 'No'}<br>
    <strong>Score Generation Default:</strong> {prompt.scoreGenerationDefault ? 'Yes' : 'No'}<br>
  </div>
  <a href="/prompts/{prompt.id}/edit">Edit Prompt</a>
{:else}
  <p>Prompt not found.</p>
{/if}
