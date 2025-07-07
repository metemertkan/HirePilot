<script lang="ts">
    import { goto } from '$app/navigation';
    import { onMount } from 'svelte';
    let promptName = '';
    let promptText = '';
    let errorPrompt = '';

    async function addPrompt() {
        errorPrompt = '';
        try {
            const res = await fetch(PROMPT_API_URL, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ name: promptName, prompt: promptText })
            });
            if (!res.ok) throw new Error('Failed to add prompt');
            promptName = '';
            promptText = '';
            await fetchPrompts();
        } catch (e) {
            if (e instanceof Error) {
                errorPrompt = e.message;
            } else {
                errorPrompt = String(e);
            }
        }
    }

    // Job state
    let jobs: { id: number; title: string; company: string; link: string; applied: boolean; cvGenerated: boolean; cv: string; description: string}[] = [];
    let title = '';
    let company = '';
    let link = '';
    let applied = false;
    let cvGenerated = false;
    let cv = '';
    let loading = false;
    let error = '';
    let description = '';

    // Prompt state
    let prompts: { id: number; name: string; prompt: string }[] = [];
    let loadingPrompts = false;
    let errorPrompts = '';

    // For each job, track selected prompt id (default: none)
    let selectedPromptIds: Record<number, number | null> = {};

    const JOB_API_URL = 'http://localhost:8080/api/jobs';
    const PROMPT_API_URL = 'http://localhost:8080/api/prompts';

    async function fetchJobs() {
        loading = true;
        error = '';
        try {
            const res = await fetch(JOB_API_URL);
            if (!res.ok) throw new Error('Failed to fetch jobs');
            jobs = await res.json();
            // Initialize selectedPromptIds for each job (default: first prompt if available)
            for (const job of jobs) {
                if (!(job.id in selectedPromptIds)) {
                    selectedPromptIds[job.id] = (Array.isArray(prompts) && prompts.length > 0) ? prompts[0].id : null;
                }
            }
        }  catch (e) {
            if (e instanceof Error) {
                error = e.message;
            } else {
                error = String(e);
            }
        } finally {
            loading = false;
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

    async function addJob() {
        error = '';
        try {
            const res = await fetch(JOB_API_URL, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ title, company, link, applied, cvGenerated, cv , description })
            });
            if (!res.ok) throw new Error('Failed to add job');
            title = company = link = cv = description = '';
            applied = false;
            cvGenerated = false;
            await fetchJobs();
            } catch (e) {
                if (e instanceof Error) {
                    error = e.message;
                } else {
                    error = String(e);
                }
            }
    }

    // Generate CV for a job with selected prompt id
    async function generateCV(jobId: number) {
        try {
            loading = true;
            error = '';
            // Find selected prompt id for this job
            const promptId = selectedPromptIds[jobId];
            const res = await fetch(`${JOB_API_URL}/${jobId}/generate-cv`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ promptId })
            });
            if (!res.ok) throw new Error('Failed to generate CV');
            await fetchJobs();
        } catch (e) {
            if (e instanceof Error) {
                error = e.message;
            } else {
                error = String(e);
            }
        } finally {
            loading = false;
        }
    }

    onMount(async () => {
        await fetchPrompts();
        await fetchJobs();
    });
</script>

<style>
    .container {
        display: flex;
        gap: 2rem;
        align-items: flex-start;
    }
    .left-column {
        flex: 2;
        min-width: 0;
    }
    .right-column {
        flex: 1;
        min-width: 250px;
        background: #f9f9f9;
        padding: 1rem;
        border-radius: 8px;
        border: 1px solid #eee;
    }
    .prompt-list {
        list-style: none;
        padding: 0;
    }
    .prompt-list li {
        margin-bottom: 1rem;
        padding: 0.5rem;
        background: #fff;
        border-radius: 4px;
        border: 1px solid #ddd;
    }
    .prompt-select {
        margin: 0 0.5rem;
    }
</style>

<h1>HirePilot Job Board</h1>

<div class="container">
    <!-- Left: Jobs and Form -->
    <div class="left-column">
        <h2>Jobs</h2>
        <form on:submit|preventDefault={addJob}>
            <input placeholder="Job Title" bind:value={title} required />
            <input placeholder="Company" bind:value={company} required />
            <input placeholder="Link" bind:value={link} required />
            <textarea placeholder="Description" bind:value={description} required rows="4"></textarea>
            <button type="submit">Add Job</button>
        </form>

        {#if loading}
            <p>Loading...</p>
        {:else if error}
            <p style="color: red">{error}</p>
        {:else}
            <h2>Jobs</h2>
            <ul>
                {#each jobs as job}
                    <li>
                        <strong>{job.title}</strong> at {job.company} — 
                        {job.applied ? 'Applied' : 'Not Applied'} —
                        {job.cvGenerated ? 'CV Generated' : 'CV Not Generated'} —
                        <button on:click={() => goto(`/jobs/${job.id}`)}>View</button>
                        <!-- Prompt selection dropdown -->
                        <select
                            class="prompt-select"
                            bind:value={selectedPromptIds[job.id]}
                            on:change={(e) => {
                                selectedPromptIds[job.id] = +e.target.value;
                            }}
                            disabled={loading || prompts.length === 0}
                        >
                            {#each prompts as prompt}
                                <option value={prompt.id}>{prompt.name}</option>
                            {/each}
                        </select>
                        <button
                            on:click={() => generateCV(job.id)}
                            disabled={loading || job.cvGenerated}
                        >
                            {job.cvGenerated ? 'CV Generated' : 'Generate CV'}
                        </button>
                    </li>
                {/each}
            </ul>
        {/if}
    </div>

    <!-- Right: Prompts -->
    <div class="right-column">
        <h2>Prompts</h2>
        {#if loadingPrompts}
            <p>Loading prompts...</p>
        {:else if errorPrompts}
            <p style="color: red">{errorPrompts}</p>
        {:else}
            <form on:submit|preventDefault={addPrompt} style="margin-bottom: 1rem;">
                <input placeholder="Prompt Name" bind:value={promptName} required />
                <textarea placeholder="Prompt Text" bind:value={promptText} required rows="3"></textarea>
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
                            <button on:click={() => goto(`/prompts/${prompt.id}`)}>View</button>
                        </li>
                    {/each}
                </ul>
            {:else}
                <p>No prompts found. Add a prompt above.</p>
            {/if}
        {/if}
    </div>
</div>