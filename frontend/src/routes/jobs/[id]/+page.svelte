<script lang="ts">
    export let data: {
        job: { id: number; title: string; company: string; link: string; status: string; cvGenerated: boolean; cv: string; description: string; score:number } | null;
    };
    import { invalidateAll } from '$app/navigation';
    import { onMount } from 'svelte';

    let loading = false;
    let error = '';
    let polling = false;
    let pollTimer: ReturnType<typeof setInterval> | null = null;
    let pollInterval = 3000; // 3 seconds
    let prompts: { id: number; name: string; prompt: string }[] = [];
    let selectedPromptId: number | null = null;
    let scoreLoading = false;

     async function fetchPrompts() {
        const res = await fetch('http://localhost:8080/api/prompts');
        prompts = await res.json();
        if (prompts.length > 0) selectedPromptId = prompts[0].id;
    }

    async function generateScore() {
        if (!data.job || !selectedPromptId) return;
        scoreLoading = true;
        error = '';
        try {
            const res = await fetch(`http://localhost:8080/api/jobs/${data.job.id}/generate-score`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ promptId: selectedPromptId })
            });
            if (!res.ok) throw new Error('Failed to generate score');
            await fetchJobStatus();
        } catch (e) {
            error = e instanceof Error ? e.message : String(e);
        } finally {
            scoreLoading = false;
        }
    }

    async function fetchJobStatus() {
        if (!data.job) return;
        try {
            const res = await fetch(`http://localhost:8080/api/jobs/${data.job.id}`);
            if (!res.ok) throw new Error('Failed to fetch job status');
            const job = await res.json();
            data.job = job;
        } catch (e) {
            if (e instanceof Error) {
                error = e.message;
            } else {
                error = String(e);
            }
            stopPolling();
        }
        }
        
        async function closeJob() {
        if (!data.job) return;
            loading = true;
            error = '';
        try {
            const res = await fetch(`http://localhost:8080/api/jobs/${data.job.id}/close`, {
            method: 'PUT'
        });
            if (!res.ok) throw new Error('Failed to close job');
                await fetchJobStatus();
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

    function startPolling() {
        polling = true;
        fetchJobStatus();
        pollTimer = setInterval(async () => {
            await fetchJobStatus();
            if (data.job && data.job.cvGenerated) {
                stopPolling();
                loading = false;
            }
        }, pollInterval);
    }

    function stopPolling() {
        polling = false;
        if (pollTimer) {
            clearInterval(pollTimer);
            pollTimer = null;
        }
    }

    async function generateCV() {
        if (!data.job) return;
        loading = true;
        error = '';
        try {
            const res = await fetch(`http://localhost:8080/api/jobs/${data.job.id}/generate-cv`, {
                method: 'POST'
            });
            if (!res.ok) throw new Error('Failed to generate CV');
            startPolling();
        } catch (e) {
            if (e instanceof Error) {
                error = e.message;
            } else {
                error = String(e);
            }
            loading = false;
        }
    }

    async function applyJob() {
        if (!data.job) return;
        loading = true;
        error = '';
        try {
            const res = await fetch(`http://localhost:8080/api/jobs/${data.job.id}/apply`, {
                method: 'PUT'
            });
            if (!res.ok) throw new Error('Failed to apply for job');
            await fetchJobStatus();
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
    });
</script>

{#if data.job}
    <header>
        <h1>{data.job.company} — {data.job.title}</h1>
        <p>
            <strong>Link:</strong>
            <a href={data.job.link} target="_blank" rel="noopener">{data.job.link}</a>
        </p>
        <p>
            <strong>Status:</strong> {data.job.status === 'applied' ? 'Applied' : data.job.status === 'closed' ? 'Closed' : 'Open'} |
            <strong>CV:</strong> {data.job.cvGenerated ? 'Generated' : 'Not Generated'}
        </p>
        <p>
            <strong>Score:</strong>
                {#if data.job && data.job.score !== null && data.job.score !== undefined}
                    {data.job.score}
                {:else}
                    <em>No score</em>
                {/if}
            </p>
    </header>
    <div class="split">
        <section class="left">
            <h2>Description</h2>
            <pre>{data.job.description}</pre>
        </section>
        <section class="right">
            <h2>CV</h2>
            {#if data.job.cvGenerated}
                <pre class="cv-pre">{data.job.cv}</pre>
            {:else}
                <em>CV not generated.</em>
            {/if}            
            <button
                on:click={generateCV}
                disabled={loading || polling || (data.job && data.job.cvGenerated)}
                style="margin-top:1rem"
            >
                {data.job && data.job.cvGenerated ? 'CV Generated' : (polling ? 'Generating (Polling)...' : loading ? 'Generating...' : 'Generate CV')}
            </button>
                        <button
                on:click={generateScore}
                disabled={loading || (data.job.cvGenerated && !(data.job.score==0 || data.job.score == null))}
                style="margin-top:1rem"
            >
                {(data.job.cvGenerated && !(data.job.score==0 || data.job.score == null)) ? 'Score Generated' : (polling ? 'Generating (Polling)...' : loading ? 'Generating...' : 'Generate Score')}
            </button>
            <button
                on:click={applyJob}
                disabled={loading || (data.job && data.job.status === 'applied') || (data.job && data.job.status === 'closed')}
                style="margin-top:1rem; margin-left:1rem"
            >
                {data.job && data.job.status === 'applied' ? 'Applied' : data.job && data.job.status === 'closed' ? 'Closed' : 'Apply'}
            </button>
            <button
                on:click={closeJob}
                disabled={loading || (data.job && data.job.status === 'closed')}
                style="margin-top:1rem; margin-left:1rem"
            >
                {data.job && data.job.status === 'closed' ? 'Closed' : 'Close'}
            </button>
            <select bind:value={selectedPromptId}>
                {#each prompts as prompt}
                    <option value={prompt.id}>{prompt.name}</option>
                {/each}
            </select>
            {#if polling}
                <button on:click={stopPolling} style="margin-left:1rem">Cancel</button>
            {/if}
            {#if error}
                <p style="color:red">{error}</p>
            {/if}
        </section>
    </div>
{:else}
    <p>Job not found.</p>
{/if}

<style>
.split {
    display: flex;
    gap: 2rem;
    margin-top: 2rem;
}
.left, .right {
    flex: 1;
    min-width: 0;
}
pre {
    white-space: pre-wrap;
    word-break: break-word;
    background: #f8f8f8;
    padding: 1rem;
    border-radius: 4px;
}
.cv-pre {
    font-family: Arial, sans-serif;
    font-size: 10pt;
}
</style>