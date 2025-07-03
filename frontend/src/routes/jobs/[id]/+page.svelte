<script lang="ts">
    export let data: {
        job: { id: number; title: string; company: string; link: string; applied: boolean; cvGenerated: boolean; cv: string } | null;
    };
</script>

{#if data.job}
    <h1>{data.job.title}</h1>
    <p><strong>Company:</strong> {data.job.company}</p>
    <p><strong>Link:</strong> <a href={data.job.link} target="_blank" rel="noopener">{data.job.link}</a></p>
    <p><strong>Status:</strong> {data.job.applied ? 'Applied' : 'Not Applied'}</p>
    <p><strong>CV:</strong> {data.job.cvGenerated ? 'Generated' : 'Not Generated'}</p>
    {#if data.job.cvGenerated}
        <pre>{data.job.cv}</pre>
    {/if}
{:else}
    <p>Job not found.</p>
{/if}