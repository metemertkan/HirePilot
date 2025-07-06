<script lang="ts">
    export let data: {
        job: { id: number; title: string; company: string; link: string; applied: boolean; cvGenerated: boolean; cv: string; description: string } | null;
    };
    import { invalidateAll } from '$app/navigation';

    let loading = false;
    let error = '';

    async function generateCV() {
        if (!data.job) return;
        loading = true;
        error = '';
        try {
            const res = await fetch(`http://localhost:8080/api/jobs/${data.job.id}/generate-cv`, {
                method: 'POST'
            });
            if (!res.ok) throw new Error('Failed to generate CV');
            await invalidateAll();
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
</script>

{#if data.job}
    <header>
        <h1>{data.job.company} — {data.job.title}</h1>
        <p>
            <strong>Link:</strong>
            <a href={data.job.link} target="_blank" rel="noopener">{data.job.link}</a>
        </p>
        <p>
            <strong>Status:</strong> {data.job.applied ? 'Applied' : 'Not Applied'} |
            <strong>CV:</strong> {data.job.cvGenerated ? 'Generated' : 'Not Generated'}
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
                <pre>{data.job.cv}</pre>
            {:else}
                <em>CV not generated.</em>
            {/if}
            <button
                on:click={generateCV}
                disabled={loading || (data.job && data.job.cvGenerated)}
                style="margin-top:1rem"
            >
                {data.job && data.job.cvGenerated ? 'CV Generated' : loading ? 'Generating...' : 'Generate CV'}
            </button>
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
</style>