<script lang="ts">
    let jobs: { title: string; company: string; link: string }[] = [];
    let title = '';
    let company = '';
    let link = '';
    let loading = false;
    let error = '';

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
                body: JSON.stringify({ title, company, link })
            });
            if (!res.ok) throw new Error('Failed to add job');
            title = company = link = '';
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
                <a href={job.link} target="_blank" rel="noopener">View</a>
            </li>
        {/each}
    </ul>
{/if}