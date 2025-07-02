<script>
  export let jobs = [];
</script>

<ul class="job-list">
  {#each jobs as job, i}
    <li>
      <strong>{job.title}</strong> at {job.company}
    <div>
      <a href="{job.link}" target="_blank" rel="noopener noreferrer">{job.link}</a>
    </div>
    </li>
  {/each}
</ul>

<style>
.job-list {
  list-style: none;
  padding: 0;
  max-width: 600px;
}
.job-list li {
  margin-bottom: 1rem;
  padding: 0.5rem;
  border: 1px solid #eee;
  border-radius: 4px;
}
</style>
