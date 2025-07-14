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


<div class="row justify-content-center">

    <div class="col-xl-10 col-lg-12 col-md-9">

        <div class="card o-hidden border-0 shadow-lg my-5">
            <div class="card-body p-0">
                <!-- Nested Row within Card Body -->
                <div class="row">
                    <div class="col-lg-12">
                        <div class="p-5">
                            <div class="text-center">
                                <h1 class="h4 text-gray-900 mb-4">Add A Prompt</h1>
                            </div>
                            <form class="user" on:submit|preventDefault={addPrompt}>
                                <div class="form-group">
                                    <input type="text" class="form-control form-control-user"
                                    bind:value={promptName} required placeholder="Prompt Name">
                                </div>
                                <div class="form-group">
                                    <textarea class="form-control form-control-user" placeholder="Prompt Text" 
                                    bind:value={promptText} required></textarea>
                                </div>
                                <div class="form-group">
                                    <label>
                                        CV Generation Default:
                                        <input type="checkbox" bind:checked={cvGenerationDefault} />
                                    </label>
                                </div>
                                <div class="form-group">
                                    <label>
                                        Score Generation Default:
                                        <input type="checkbox" bind:checked={scoreGenerationDefault} />
                                    </label>
                                </div>
                                <button class="btn btn-primary btn-user btn-block" type="submit">Add Prompt</button>
                                {#if errorPrompt}
                                    <div class="alert alert-danger mt-3">{errorPrompt}</div>
                                {/if}
                                {#if loadingPrompts}
                                    <div class="alert alert-info mt-3">Loading...</div>
                                {/if}
                                {#if errorPrompts}
                                    <div class="alert alert-danger mt-3">{errorPrompts}</div>
                                {/if}
                            </form>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </div>
</div>




<div class="card shadow mb-4">
            <div class="card-body">
                <div class="table-responsive">
                    <table class="table table-bordered" id="dataTable" width="100%" cellspacing="0">
                        <thead>
                            <tr>
                                <th>Name</th>
                                <th>Prompt</th>
                                <th>cvGenerationDefault</th>
                                <th>scoreGenerationDefault</th>
                                <th></th>
                            </tr>
                        </thead>
                        <tfoot>
                            <tr>
                                <th>Name</th>
                                <th>Prompt</th>
                                <th>cvGenerationDefault</th>
                                <th>scoreGenerationDefault</th>
                                <th></th>
                            </tr>
                        </tfoot>
                        <tbody>
                            {#each prompts as prompt}
                                <tr>
                                    <td>{prompt.name}</td>
                                    <td>{prompt.prompt.slice(0, 100)}{prompt.prompt.length > 100 ? '...' : ''}</td>
                                    <td>{prompt.cvGenerationDefault === '1' || prompt.cvGenerationDefault === true ? 'Yes' : 'No'}</td>
                                    <td>{prompt.scoreGenerationDefault === '1' || prompt.scoreGenerationDefault === true ? 'Yes' : 'No'}</td>
                                    <td><button on:click={() => goto(`/prompts/${prompt.id}`)}>View</button></td>
                                </tr>
                            {/each}
                        </tbody>
                    </table>
                </div>
            </div>
        </div>  