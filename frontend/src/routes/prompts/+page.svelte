<script lang="ts">
    import { goto } from '$app/navigation';
    import { onMount } from 'svelte';

    onMount(async () => {
        await fetchPrompts();
    });
         // Prompt state
    let prompts: { id: number; name: string; prompt: string, cvGenerationDefault: boolean, scoreGenerationDefault:boolean }[] = [];
    let loadingPrompts = false;
    let errorPrompts = '';
    const PROMPT_API_URL = 'http://localhost:8080/api/prompts';
    let promptName = '';
    let promptText = '';
    let errorPrompt = '';
    let cvGenerationDefault = false;
    let scoreGenerationDefault = false;

    async function addPrompt() {
        errorPrompt = '';
        try {
            const res = await fetch(PROMPT_API_URL, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ name: promptName, prompt: promptText, cvGenerationDefault: cvGenerationDefault, scoreGenerationDefault: scoreGenerationDefault })
            });
            if (!res.ok) throw new Error('Failed to add prompt');
            promptName = '';
            promptText = '';
            cvGenerationDefault = false;
            scoreGenerationDefault = false;
            await fetchPrompts();
        } catch (e) {
            if (e instanceof Error) {
                errorPrompt = e.message;
            } else {
                errorPrompt = String(e);
            }
        }
    }
    async function fetchPrompts() {
        loadingPrompts = true;
        errorPrompts = '';
        try {
            const res = await fetch(PROMPT_API_URL);
            if (!res.ok) throw new Error('Failed to fetch prompts');
            const data = await res.json();
            prompts = Array.isArray(data) ? data : [];
        } catch (e) {
            prompts = [];
            if (e instanceof Error) {
                errorPrompts = e.message;
            } else {
                errorPrompts = String(e);
            }            
        } finally {
            loadingPrompts = false;
        }
    }


</script>

<h2>Prompts</h2>
        {#if loadingPrompts}
            <p>Loading prompts...</p>
        {:else if errorPrompts}
            <p style="color: red">{errorPrompts}</p>
        {:else}
            <form on:submit|preventDefault={addPrompt} style="margin-bottom: 1rem;">
                <input placeholder="Prompt Name" bind:value={promptName} required />
                <textarea placeholder="Prompt Text" bind:value={promptText} required rows="3"></textarea>
                <label>
                    CV Generation Default:
                    <input type="checkbox" bind:checked={cvGenerationDefault} />
                </label>
                <br>
                <label>
                    Score Generation Default:
                    <input type="checkbox" bind:checked={scoreGenerationDefault} />
                </label>
                <br>
                <button type="submit">Add Prompt</button>
                {#if errorPrompt}
                    <p style="color: red">{errorPrompt}</p>
                {/if}
            </form>
            {#if prompts && prompts.length > 0}
                <ul class="prompt-list">
                    {#each prompts as prompt}
                        <li>
                            <strong>{prompt.name}</strong>
                            <div>{prompt.prompt}</div>
                            <div>
                                CV Generation Default: {prompt.cvGenerationDefault === '1' || prompt.cvGenerationDefault === true ? 'Yes' : 'No'}<br>
                                Score Generation Default: {prompt.scoreGenerationDefault === '1' || prompt.scoreGenerationDefault === true ? 'Yes' : 'No'}
                            </div>
                            <button on:click={() => goto(`/prompts/${prompt.id}`)}>View</button>
                        </li>
                    {/each}
                </ul>
            {:else}
                <p>No prompts found. Add a prompt above.</p>
            {/if}
        {/if}