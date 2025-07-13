<script lang="ts">
    import { onMount } from 'svelte';
    const JOB_API_URL = 'http://localhost:8080/api/jobs/today';
    let loading = false;
    let error = '';
    let todayJobsCount = 0;
    let openJobs = 0;
    let totalAppliedJobs = 0;
    let promptsCount = 0;
    let loadingPrompts = false;
    let errorPrompts = '';
    
    const PROMPT_API_URL = 'http://localhost:8080/api/prompts';

    async function fetchOpenJobs() {
        loading = true;
        error = '';
        try {
            let url = JOB_API_URL;
            url += `?status=${encodeURIComponent('open')}`;
            const res = await fetch(url);
            if (!res.ok) throw new Error('Failed to fetch jobs');
            const data = await res.json();
            openJobs = Array.isArray(data) ? data.length : 0;
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

    async function fetchTotalAppliedJobs() {
        loading = true;
        error = '';
        try {
            let url = JOB_API_URL;
            url += `?status=${encodeURIComponent('applied')}`;
            const res = await fetch(url);
            if (!res.ok) throw new Error('Failed to fetch jobs');
            const data = await res.json();
            totalAppliedJobs = Array.isArray(data) ? data.length : 0;
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

    async function fetchTodayJobsCount() {
    loading = true;
    error = '';
    try {
        const res = await fetch(JOB_API_URL);
        if (!res.ok) throw new Error('Failed to fetch jobs count');
        const data = await res.json();
        todayJobsCount = data.count;
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

    async function fetchPromptsCount() {
        loadingPrompts = true;
        errorPrompts = '';
        try {
            const res = await fetch(PROMPT_API_URL);
            if (!res.ok) throw new Error('Failed to fetch prompts');
            const data = await res.json();
            promptsCount = data.count;
        } catch (e) {
            promptsCount = 0;
            if (e instanceof Error) {
                errorPrompts = e.message;
            } else {
                errorPrompts = String(e);
            }            
        } finally {
            loadingPrompts = false;
        }
    }
onMount(() => {
    fetchTodayJobsCount();
    fetchOpenJobs();
    fetchTotalAppliedJobs();
    fetchPromptsCount();
});
</script>


<h1>HirePilot Job Board</h1>

<div class="row">

    <!-- Earnings (Monthly) Card Example -->
    <div class="col-xl-3 col-md-6 mb-4">
        <div class="card border-left-primary shadow h-100 py-2">
            <div class="card-body">
                <div class="row no-gutters align-items-center">
                    <div class="col mr-2">
                        <div class="text-xs font-weight-bold text-primary text-uppercase mb-1">
                            Applied Today</div>
                        <div class="h4 mb-0 font-weight-bold text-gray-800">{todayJobsCount}</div>
                    </div>
                </div>
            </div>
        </div>
    </div>

    <!-- Earnings (Annual) Card Example -->
    <div class="col-xl-3 col-md-6 mb-4">
        <div class="card border-left-success shadow h-100 py-2">
            <div class="card-body">
                <div class="row no-gutters align-items-center">
                    <div class="col mr-2">
                        <div class="text-xs font-weight-bold text-success text-uppercase mb-1">
                            Open Positions</div>
                        <div class="h4 mb-0 font-weight-bold text-gray-800">{openJobs}</div>
                    </div>
                </div>
            </div>
        </div>
    </div>

    <!-- Tasks Card Example -->
    <div class="col-xl-3 col-md-6 mb-4">
        <div class="card border-left-info shadow h-100 py-2">
            <div class="card-body">
                <div class="row no-gutters align-items-center">
                    <div class="col mr-2">
                        <div class="text-xs font-weight-bold text-info text-uppercase mb-1">Total Applications
                        </div>
                        <div class="row no-gutters align-items-center">
                            <div class="col-auto">
                                <div class="h5 mb-0 mr-3 font-weight-bold text-gray-800">{totalAppliedJobs}</div>
                            </div>                      
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </div>

    <!-- Pending Requests Card Example -->
    <div class="col-xl-3 col-md-6 mb-4">
        <div class="card border-left-warning shadow h-100 py-2">
            <div class="card-body">
                <div class="row no-gutters align-items-center">
                    <div class="col mr-2">
                        <div class="text-xs font-weight-bold text-warning text-uppercase mb-1">
                            Number of Prompts</div>
                        <div class="h5 mb-0 font-weight-bold text-gray-800">{promptsCount}</div>
                    </div>
                </div>
            </div>
        </div>
    </div>
</div>