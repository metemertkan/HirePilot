<script lang="ts">
  export let data;
  let prompt = data.prompt;
  // Convert string "1"/"0" to boolean for correct display
  let cvGenDefault = prompt && (prompt.cvGenerationDefault === "1");
  let scoreGenDefault = prompt && (prompt.scoreGenerationDefault === "1");
</script>

<h1>Prompt Details</h1>
{#if prompt}
  <div>
    <strong>ID:</strong> {prompt.id}<br>
    <strong>Name:</strong> {prompt.name}<br>
    <strong>Prompt:</strong> <pre>{prompt.prompt}</pre>
    <strong>CV Generation Default:</strong> {cvGenDefault ? 'Yes' : 'No'}<br>
    <strong>Score Generation Default:</strong> {scoreGenDefault ? 'Yes' : 'No'}<br>
  </div>
  <a href="/prompts/{prompt.id}/edit">Edit Prompt</a>
{:else}
  <p>Prompt not found.</p>
{/if}
