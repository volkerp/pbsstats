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
  <div class="mainbar">
    <h1>Proxmox Backup Server Chunk usage</h1>
    <div class="chunkmap">
      <ChunkMap />
    </div>
  </div>
</main>

<style>
  .main-layout {
    display: flex;
    flex-direction: row;
    height: 100vh;
    margin: 0;
    padding: 0;
  }
  .sidebar {
    overflow-y: auto;
    background: #fafbfc;
    border-right: 1px solid #ccc;
    flex: 1;
  }
  .mainbar {
    display: flex;
    flex-direction: column;
    flex: 5;
  }
  .content {
    flex: 5;
  }
  .chunkmap {
    flex: 1;
    display: flex;
    flex-direction: column;
  }
</style>
