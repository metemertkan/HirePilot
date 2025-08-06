
<script lang="ts">
    import { goto } from '$app/navigation';
    import { onMount, onDestroy } from 'svelte';
    import { page } from '$app/stores';
    import { get } from 'svelte/store';
    import { BASE_API_URL } from '../../lib/config';

    // Job state
    let jobs: { id: number; title: string; company: string; link: string; status: string; cvGenerated: boolean; cv: string; description: string; score: number; isPending?: boolean }[] = [];
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

    // WebSocket and polling state
    let pollingInterval: NodeJS.Timeout | null = null;
    let hasPendingItems = false;
    let websocket: WebSocket | null = null;
    let connectionStatus: 'connecting' | 'connected' | 'disconnected' | 'error' = 'disconnected';
    let reconnectAttempts = 0;
    const maxReconnectAttempts = 5;

    const JOB_API_URL = `${BASE_API_URL}/api/jobs`;
    const PROMPT_API_URL = `${BASE_API_URL}/api/prompts`;

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
            
            // Preserve pending items that aren't in the response yet
            const pendingItems = jobs.filter(j => j.isPending);
            const newJobs = Array.isArray(data) ? data : [];
            
            // Remove pending items that now exist in the response
            const filteredPendingItems = pendingItems.filter(pending => 
                !newJobs.some(newJob => newJob.title === pending.title && newJob.company === pending.company)
            );
            
            jobs = [...newJobs, ...filteredPendingItems];
            hasPendingItems = filteredPendingItems.length > 0;
            
            // Initialize selectedPromptIds for each job (default: first prompt if available)
            for (const job of jobs) {
                if (!(job.id in selectedPromptIds)) {
                    selectedPromptIds[job.id] = (Array.isArray(prompts) && prompts.length > 0) ? prompts[0].id : null;
                }
            }
            
            // Start polling if we have pending items
            if (hasPendingItems) {
                startPollingIfNeeded();
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
        
        // Create optimistic job object
        const optimisticJob = {
            id: Date.now(), // Temporary ID
            title: title,
            company: company,
            link: link,
            status: status,
            cvGenerated: cvGenerated,
            cv: cv,
            description: description,
            score: 0,
            isPending: true // Flag to show it's pending
        };
        
        // Optimistically add to UI
        jobs = [...jobs, optimisticJob];
        
        // Initialize prompt selection for optimistic job
        selectedPromptIds[optimisticJob.id] = (Array.isArray(prompts) && prompts.length > 0) ? prompts[0].id : null;
        
        // Clear form immediately
        const originalValues = { title, company, link, status, cvGenerated, cv, description };
        title = company = link = cv = description = '';
        status = 'open';
        cvGenerated = false;
        
        try {
            const res = await fetch(JOB_API_URL, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ 
                    title: originalValues.title, 
                    company: originalValues.company, 
                    link: originalValues.link, 
                    status: originalValues.status, 
                    cvGenerated: originalValues.cvGenerated, 
                    cv: originalValues.cv, 
                    description: originalValues.description 
                })
            });
            
            if (!res.ok) throw new Error('Failed to add job');
            
            // Wait a bit for the backend to process, then refresh
            setTimeout(async () => {
                await fetchJobs(statusFilter);
            }, 1000);
            
        } catch (e) {
            // Remove optimistic job on error
            jobs = jobs.filter(j => j.id !== optimisticJob.id);
            delete selectedPromptIds[optimisticJob.id];
            
            // Restore form values
            title = originalValues.title;
            company = originalValues.company;
            link = originalValues.link;
            status = originalValues.status;
            cvGenerated = originalValues.cvGenerated;
            cv = originalValues.cv;
            description = originalValues.description;
            
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
        // Read status from query string if present
        const url = get(page).url;
        const statusParam = url.searchParams.get('status');
        if (statusParam && statusOptions.some(opt => opt.value === statusParam)) {
            statusFilter = statusParam;
        }
        await fetchJobs(statusFilter);
        await fetchPrompts();
        connectWebSocket();
        startPollingIfNeeded();
    });

    // WebSocket connection for real-time updates
    function connectWebSocket() {
        if (reconnectAttempts >= maxReconnectAttempts) {
            console.log('Max reconnection attempts reached, falling back to polling');
            connectionStatus = 'error';
            startPollingIfNeeded();
            return;
        }

        try {
            connectionStatus = 'connecting';
            websocket = new WebSocket('ws://localhost:8080/ws');
            
            websocket.onopen = () => {
                console.log('WebSocket connected');
                connectionStatus = 'connected';
                reconnectAttempts = 0;
            };
            
            websocket.onmessage = (event) => {
                try {
                    const message = JSON.parse(event.data);
                    handleWebSocketMessage(message);
                } catch (e) {
                    console.error('Error parsing WebSocket message:', e);
                }
            };
            
            websocket.onclose = () => {
                console.log('WebSocket disconnected');
                connectionStatus = 'disconnected';
                
                const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), 30000);
                reconnectAttempts++;
                
                setTimeout(connectWebSocket, delay);
            };
            
            websocket.onerror = (error) => {
                console.error('WebSocket error:', error);
                connectionStatus = 'error';
            };
        } catch (e) {
            console.error('Failed to connect WebSocket:', e);
            connectionStatus = 'error';
            startPollingIfNeeded();
        }
    }

    // Handle incoming WebSocket messages
    function handleWebSocketMessage(message: { type: string; data: any }) {
        switch (message.type) {
            case 'job_created':
                handleJobCreated(message.data);
                break;
            case 'job_updated':
                handleJobUpdated(message.data);
                break;
            default:
                console.log('Unknown WebSocket message type:', message.type);
        }
    }

    // Handle real-time job creation
    function handleJobCreated(newJob: any) {
        // Remove any pending item with the same title and company
        jobs = jobs.filter(j => !(j.isPending && j.title === newJob.title && j.company === newJob.company));
        
        // Add the new job if it doesn't already exist
        if (!jobs.some(j => j.id === newJob.id)) {
            jobs = [...jobs, newJob];
            // Initialize prompt selection for new job
            if (!(newJob.id in selectedPromptIds)) {
                selectedPromptIds[newJob.id] = (Array.isArray(prompts) && prompts.length > 0) ? prompts[0].id : null;
            }
        }
        
        updatePendingStatus();
    }

    // Handle real-time job updates
    function handleJobUpdated(updatedJob: any) {
        jobs = jobs.map(j => 
            j.id === updatedJob.id ? { ...updatedJob, isPending: false } : j
        );
        updatePendingStatus();
    }

    // Update pending status and polling
    function updatePendingStatus() {
        hasPendingItems = jobs.some(j => j.isPending);
        if (!hasPendingItems && pollingInterval) {
            clearInterval(pollingInterval);
            pollingInterval = null;
        }
    }

    // Start polling when there are pending items (fallback)
    function startPollingIfNeeded() {
        if (hasPendingItems && !pollingInterval && (connectionStatus === 'error' || !websocket)) {
            pollingInterval = setInterval(async () => {
                await fetchJobs(statusFilter);
                if (!hasPendingItems && pollingInterval) {
                    clearInterval(pollingInterval);
                    pollingInterval = null;
                }
            }, 2000);
        }
    }

    // Stop polling and close WebSocket when component is destroyed
    onDestroy(() => {
        if (pollingInterval) {
            clearInterval(pollingInterval);
        }
        if (websocket) {
            websocket.close();
        }
    });
</script>
<link href="https://cdn.datatables.net/2.3.2/css/dataTables.bootstrap4.min.css" rel="stylesheet">

<div class="d-flex justify-content-between align-items-center mb-3">
    <h2>Jobs</h2>
    <div class="connection-status">
        {#if connectionStatus === 'connected'}
            <span class="badge badge-success">🟢 Real-time</span>
        {:else if connectionStatus === 'connecting'}
            <span class="badge badge-warning">🟡 Connecting...</span>
        {:else if connectionStatus === 'disconnected'}
            <span class="badge badge-warning">🟡 Reconnecting...</span>
        {:else if connectionStatus === 'error'}
            <span class="badge badge-secondary">⚫ Polling mode</span>
        {/if}
    </div>
</div>

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
                                <th></th>
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
                                <th></th>
                            </tr>
                        </tfoot>
                        <tbody>
                            {#each jobs as job}
                                <tr class={job.isPending ? 'table-warning' : ''}>
                                    <td>
                                        <button on:click={() => goto(`/jobs/${job.id}`)} disabled={job.isPending}>
                                            {job.title}
                                        </button>
                                        {#if job.isPending}
                                            <span class="badge badge-warning ml-2">Processing...</span>
                                        {/if}
                                    </td>
                                    <td>{job.company}</td>
                                    <td>{job.link}</td>
                                    <td>{job.status === 'applied' ? 'Applied' : job.status === 'closed' ? 'Closed' : 'Open'}</td>
                                    <td>{job.cvGenerated ? 'Yes' : 'No'}</td>
                                    <td>{job.score == null || job.score == 0 ? 'Not scored': job.score}</td>
                                    <td>
                                        {#if !job.isPending}
                                            <button on:click={() => generateCV(job.id)} disabled={loading || job.cvGenerated}>Generate CV</button>
                                        {:else}
                                            <button disabled class="btn btn-secondary btn-sm">Processing...</button>
                                        {/if}
                                    </td>
                                    <td>
                                        {#if !job.isPending}
                                            <button on:click={() => generateScore(job.id)} disabled={loading || (job.cvGenerated && !(job.score==0 || job.score == null))}>Generate Score</button>
                                        {:else}
                                            <button disabled class="btn btn-secondary btn-sm">Processing...</button>
                                        {/if}
                                    </td>
                                    <td>
                                        {#if !job.isPending}
                                            <button on:click={() => goto(`/jobs/${job.id}`)}>View</button>
                                        {:else}
                                            <button disabled class="btn btn-secondary btn-sm">Processing...</button>
                                        {/if}
                                    </td>
                                </tr>
                            {/each}
                        </tbody>
                    </table>
                </div>
            </div>
        </div>

        {#if error}
            <div class="alert alert-danger">{error}</div>
        {/if}
        {#if loading}
            <div class="alert alert-info">Loading...</div>
        {/if}
