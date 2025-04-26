<script>
  import svelteLogo from './assets/svelte.svg'
  import viteLogo from '/vite.svg'
  import Counter from './lib/Counter.svelte'
  import ChunkMap from './lib/ChunkMap.svelte'
  import FileList from './lib/FileList.svelte'
</script>

<main class="main-layout">
  <div class="sidebar">
    <FileList />
  </div>
  <div class="content">
    <h1>Proxmox Backup Server Chunk usage</h1>
    <div>
      <ChunkMap />
    </div>
  </div>
</main>

<style>
  .main-layout {
    display: flex;
    height: 100vh;
    margin: 0;
    padding: 0;
  }
  .sidebar {
    height: 100vh;
    overflow-y: auto;
    background: #fafbfc;
    border-right: 1px solid #ccc;
    flex-shrink: 0;
  }
  .content {
    flex: 1 1 0%;
    padding: 1.5em;
    overflow: auto;
  }
</style>
