<script>
  import { onMount } from 'svelte';
  import { hoveredFile } from './hoveredFile.js';
  import { filesStore } from './filesStore.js';
  import { get } from 'svelte/store';
  let loading = true;
  let error = null;

  let files = [];

  const unsubscribe = filesStore.subscribe(value => {
    files = value;
  });

  async function fetchFiles() {
    try {
      const res = await fetch('http://localhost:8080/api/files');
      if (!res.ok) throw new Error('Failed to fetch files');
      let fetchedFiles = await res.json();
      fetchedFiles.sort((a, b) => a.filename.localeCompare(b.filename));
      // Convert unique_ref_chunks to Set for better performance
      for (const file of fetchedFiles) {
        if (Array.isArray(file.ref_chunks)) {
          file.ref_chunks = new Set(file.ref_chunks);
        }
      }

      filesStore.set(fetchedFiles);
    } catch (e) {
      error = e.message;
    } finally {
      loading = false;
    }
  }

  function ratioToColor(ratio) {
    ratio = Math.max(0, Math.min(1, ratio));
    const r = Math.round(255 * ratio);
    const g = Math.round(255 * (1 - ratio));
    return `rgb(${r},${g},0)`;
  }

  function handleMouseOver(file) {
    hoveredFile.set(file);

  }

  function handleMouseOut() {
    hoveredFile.set(null);
  }

  onMount(fetchFiles);
</script>

<style>
.file-list {
  width: 100%;
  max-height: 100vh;
  overflow-y: scroll;
  border-right: 1px solid #ccc;
  background: #fafbfc;
  font-size: 0.92em;
  padding: 0.25em 0.5em;
}
.file-entry {
  padding: 2px 4px;
  margin: 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  cursor: pointer;
  border-radius: 2px;
  border-bottom: 2px solid #ccc;
  transition: background 0.3s, border-bottom 0.3s;
}
.file-entry:hover {
  background: #e6f0fa;
}
</style>

<div class="file-list">
  {#if loading}
    <div>Loading...</div>
  {:else if error}
    <div style="color:red">{error}</div>
  {:else}
    {#each files as file}
        <div class="file-entry"
          title="{file.filename} Dedup:{file.unique_chunks / file.num_ref_chunks}"
          style="background-color: {ratioToColor(file.unique_chunks / file.num_ref_chunks)}"
          on:mouseenter={() => handleMouseOver(file)}
          on:mouseleave={handleMouseOut}
          role="button"
          tabindex="0"
          aria-label="{file.filename}, Deduplication ratio: {file.unique_chunks / file.num_ref_chunks}"
        >
          {file.filename}
        </div>
    {/each}
  {/if}
</div>
