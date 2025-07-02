<script>
  import JobInput from './components/JobInput.svelte';
  import JobList from './components/JobList.svelte';
  import { onMount } from 'svelte';
  let jobs = [];

  async function fetchJobs() {
    const res = await fetch('http://localhost:8080/api/jobs');
    jobs = await res.json();
  }

  onMount(fetchJobs);

  function handleJobAdded() {
    fetchJobs();
  }
</script>

<main>
  <h1>HirePilot Job Tracker</h1>
  <JobInput on:jobAdded={handleJobAdded} />
  <JobList {jobs} />
</main>

<style>
main {
  font-family: system-ui, sans-serif;
  padding: 2rem;
}
</style>