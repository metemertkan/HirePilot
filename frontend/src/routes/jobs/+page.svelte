
<script lang="ts">
    import { goto } from '$app/navigation';
    import { onMount } from 'svelte';

    // Job state
    let jobs: { id: number; title: string; company: string; link: string; status: string; cvGenerated: boolean; cv: string; description: string; score: number}[] = [];
    let title = '';
    let company = '';
    let link = '';
    let status = 'open';
    let cvGenerated = false;
    let cv = '';
    let loading = false;
    let error = '';
    let description = '';
    // For each job, track selected prompt id (default: none)
    let selectedPromptIds: Record<number, number | null> = {};
    let prompts: { id: number; name: string; prompt: string, cvGenerationDefault: boolean, scoreGenerationDefault:boolean }[] = [];
    let loadingPrompts = false;
    let errorPrompts = '';

    const JOB_API_URL = 'http://localhost:8080/api/jobs';
    const PROMPT_API_URL = 'http://localhost:8080/api/prompts';

    let statusFilter = 'all';
    const statusOptions = [
        { value: 'all', label: 'All' },
        { value: 'open', label: 'Open' },
        { value: 'applied', label: 'Applied' },
        { value: 'closed', label: 'Closed' }
    ];

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

    async function fetchJobs(filterStatus = statusFilter) {
        loading = true;
        error = '';
        try {
            let url = JOB_API_URL;
            if (filterStatus && filterStatus !== 'all') {
                url += `?status=${encodeURIComponent(filterStatus)}`;
            }
            const res = await fetch(url);
            if (!res.ok) throw new Error('Failed to fetch jobs');
            const data = await res.json();
            jobs = Array.isArray(data) ? data : [];
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



    async function addJob() {
        error = '';
        try {
            const res = await fetch(JOB_API_URL, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ title, company, link, status, cvGenerated, cv , description })
            });
            if (!res.ok) throw new Error('Failed to add job');
            title = company = link = cv = description = '';
            status = 'open';
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

        async function generateScore(jobId: number) {
        try {
            loading = true;
            error = '';
            // Find selected prompt id for this job
            const promptId = selectedPromptIds[jobId];
            const res = await fetch(`${JOB_API_URL}/${jobId}/generate-score`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ promptId })
            });
            if (!res.ok) throw new Error('Failed to generate Score');
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
        await fetchJobs();
        await fetchPrompts();
    });
</script>
<link href="https://cdn.datatables.net/2.3.2/css/dataTables.bootstrap4.min.css" rel="stylesheet">

<h2>Jobs</h2>

<div class="row justify-content-center">

    <div class="col-xl-10 col-lg-12 col-md-9">

        <div class="card o-hidden border-0 shadow-lg my-5">
            <div class="card-body p-0">
                <!-- Nested Row within Card Body -->
                <div class="row">
                    <div class="col-lg-12">
                        <div class="p-5">
                            <div class="text-center">
                                <h1 class="h4 text-gray-900 mb-4">Add A Job</h1>
                            </div>
                            <form class="user" on:submit|preventDefault={addJob}>
                                <div class="form-group">
                                    <input type="text" class="form-control form-control-user"
                                    bind:value={title} required placeholder="Job Title">
                                </div>
                                <div class="form-group">
                                    <input type="text" class="form-control form-control-user" placeholder="Company" 
                                    bind:value={company} required>
                                </div>
                                <div class="form-group">
                                    <input type="text" class="form-control form-control-user" placeholder="Link" 
                                    bind:value={link} required>
                                </div>
                                <div class="form-group">
                                    <textarea class="form-control form-control-user" placeholder="Description" 
                                    bind:value={description} required rows="4"></textarea>
                                </div>
                                <button class="btn btn-primary btn-user btn-block" type="submit">Add Job</button>
                            </form>
                        </div>
                    </div>
                </div>
            </div>
        </div>

    </div>

</div>
<!--
        <form on:submit|preventDefault={addJob}>
            <input placeholder="Job Title" bind:value={title} required />
            <input placeholder="Company" bind:value={company} required />
            <input placeholder="Link" bind:value={link} required />
            <textarea placeholder="Description" bind:value={description} required rows="4"></textarea>
            <button type="submit">Add Job</button>
        </form>
-->
        <label for="status-filter">Filter by status: </label>
        <select id="status-filter" class="btn btn-primary dropdown-toggle"  bind:value={statusFilter} on:change={() => fetchJobs(statusFilter)}>
            {#each statusOptions as option}
                <option value={option.value}>{option.label}</option>
            {/each}
        </select>

        <div class="card shadow mb-4">
            <div class="card-body">
                <div class="table-responsive">
                    <table class="table table-bordered" id="dataTable" width="100%" cellspacing="0">
                        <thead>
                            <tr>
                                <th>Title</th>
                                <th>Company</th>
                                <th>Link</th>
                                <th>Status</th>
                                <th>CV Generated</th>
                                <th>Score</th>
                                <th>Generate CV</th>
                                <th>Generate Score</th>
                                <th>View</th>
                            </tr>
                        </thead>
                        <tfoot>
                            <tr>
                                <th>Title</th>
                                <th>Company</th>
                                <th>Link</th>
                                <th>Status</th>
                                <th>CV Generated</th>
                                <th>Score</th>
                                <th>Generate CV</th>
                                <th>Generate Score</th>
                                <th>View</th>
                            </tr>
                        </tfoot>
                        <tbody>
                            {#each jobs as job}
                                <tr>
                                    <td>{job.title}</td>
                                    <td>{job.company}</td>
                                    <td>{job.link}</td>
                                    <td>{job.status === 'applied' ? 'Applied' : job.status === 'closed' ? 'Closed' : 'Open'}</td>
                                    <td>{job.cvGenerated ? 'Yes' : 'No'}</td>
                                    <td>{job.score == null || job.score == 0 ? 'Not scored': job.score}</td>
                                    <td><button on:click={() => generateCV(job.id)} disabled={loading || job.cvGenerated}>Generate CV</button></td>
                                    <td><button on:click={() => generateScore(job.id)} disabled={loading || !job.cv}>Generate Score</button></td>
                                    <td><button on:click={() => goto(`/jobs/${job.id}`)}>View</button></td>
                                </tr>
                            {/each}
                        </tbody>
                    </table>
                </div>
            </div>
        </div>


<!--
        {#if loading}
            <p>Loading...</p>
        {:else if error}
            <p style="color: red">{error}</p>
        {:else}
            <h2>Jobs</h2>
            <label for="status-filter">Filter by status: </label>
            <select id="status-filter" bind:value={statusFilter} on:change={() => fetchJobs(statusFilter)}>
                {#each statusOptions as option}
                    <option value={option.value}>{option.label}</option>
                {/each}
            </select>
            {#if jobs.length === 0}
                <p>No jobs found.</p>
            {:else}
                <ul class="job-list">
                    {#each jobs as job}
                        <li>
                            <strong>{job.title}</strong> at {job.company} — 
                            {job.status === 'applied' ? 'Applied' : job.status === 'closed' ? 'Closed' : 'Open'} —
                            {job.cvGenerated ? 'CV Generated' : 'CV Not Generated'} —
                            {'Score: ' + job.score == null || job.score == 0 ? 'Not scored': job.score} —
                            <button on:click={() => goto(`/jobs/${job.id}`)}>View</button>
            
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
                            <button
                                on:click={() => generateScore(job.id)}
                                disabled={loading || !job.cv}
                            >
                                {job.cv ? (job.score ? 'Score Generated' : 'Generate Score') : 'Generate Score'}
                            </button>
                        </li>
                    {/each}
                </ul>
            {/if}
        {/if}
-->