<script>
  import svelteLogo from './assets/svelte.svg'
  import viteLogo from '/vite.svg'
  import Counter from './lib/Counter.svelte'
  import ChunkMap from './lib/ChunkMap.svelte'
  import FileList from './lib/FileList.svelte'

  let progressCnt = $state(0)
  let progressMsg = $state('Loading...')

  const eventSource = new EventSource('http://localhost:8080/api/stream');
  eventSource.onmessage = (event) => {
    const data = JSON.parse(event.data);
    progressCnt = data.count;
    progressMsg = data.file;
  };
  eventSource.addEventListener('close', (event) => {
    progressMsg = 'event: close';
    eventSource.close();
  });

</script>

<main class="main-layout">
  <!-- <div class="sidebar">
    <FileList />
  </div> -->
  <div class="mainbar">
    <h1>Proxmox Backup Server Chunk usage</h1>
    {#if progressMsg.startsWith('event: close')}
      <h2>Files scanned: {progressCnt}</h2>
    {:else}
      <h2>Scanning: {progressCnt} {progressMsg}</h2> 
    {/if}
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
    padding: 0;
    margin: 0 2px;
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
    font-size: 1.3em;
    margin: 0.5em 1em;
    text-align: left;
  }

  .chunkmap {
    flex: 1;
    display: flex;
    flex-direction: column;
  }
</style>
