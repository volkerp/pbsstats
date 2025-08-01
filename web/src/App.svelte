<script>
  import svelteLogo from './assets/svelte.svg'
  import viteLogo from '/vite.svg'
  import Counter from './lib/Counter.svelte'
  import ChunkMap from './lib/ChunkMap.svelte'
  import FileList from './lib/FileList.svelte'

  let progressMsg = $state('Loading...')

  const eventSource = new EventSource('http://localhost:8080/api/stream');
  eventSource.onmessage = (event) => {
    progressMsg = event.data; 
  };
  eventSource.addEventListener('close', (event) => {
    progressMsg = 'event: close';
    eventSource.close();
  });

</script>

<main class="main-layout">
  <div class="sidebar">
    <FileList />
  </div>
  <div class="mainbar">
    <h1>Proxmox Backup Server Chunk usage</h1>
    <h2>Scanning: {progressMsg}</h2>
    <div class="chunkmap">
      <ChunkMap progressMsg={progressMsg} />
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

  h1 {
    font-size: 1.8em;
    margin: 0.5em 0;
  }

  h2 {
    font-size: 1.5em;
    margin: 0.5em 0;
  }

  .chunkmap {
    flex: 1;
    display: flex;
    flex-direction: column;
  }
</style>
