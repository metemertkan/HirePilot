<script lang="ts">
    import { goto } from '$app/navigation';
    
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

    const API_URL = 'http://localhost:8080/api/jobs';

    async function fetchJobs() {
        loading = true;
        error = '';
        try {
            const res = await fetch(API_URL);
            if (!res.ok) throw new Error('Failed to fetch jobs');
            jobs = await res.json();
        } catch (e) {
            error = e.message;
        } finally {
            loading = false;
        }
    }

    async function addJob() {
        error = '';
        try {
            const res = await fetch(API_URL, {
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
            error = e.message;
        }
    }

    // Fetch jobs on mount
    import { onMount } from 'svelte';
    onMount(fetchJobs);
</script>

<h1>HirePilot Job Board</h1>

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
                <button
                    on:click={async () => {
                        try {
                            loading = true;
                            error = '';
                            const res = await fetch(`${API_URL}/${job.id}/generate-cv`, {
                                method: 'POST'
                            });
                            if (!res.ok) throw new Error('Failed to generate CV');
                            await fetchJobs();
                        } catch (e) {
                            error = e.message;
                        } finally {
                            loading = false;
                        }
                    }}
                    disabled={loading || job.cvGenerated}
                >
                    {job.cvGenerated ? 'CV Generated' : 'Generate CV'}
                </button>
            </li>
        {/each}
    </ul>
{/if}