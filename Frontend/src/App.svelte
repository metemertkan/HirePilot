<script>
  import JobInput from './components/JobInput.svelte';
  import JobList from './components/JobList.svelte';
  let jobs = [];

  function handleAddJob(event) {
    jobs = [...jobs, event.detail];
  }
</script>

<main>
  <h1>HirePilot Job Tracker</h1>
  <JobInput on:add={handleAddJob} />
  <JobList {jobs} />
</main>

<style>
main {
  font-family: system-ui, sans-serif;
  padding: 2rem;
}
</style>
