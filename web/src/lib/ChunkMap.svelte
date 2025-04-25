<script>
  import { onMount } from 'svelte';
  import { hoveredFile } from './hoveredFile.js';
  import { filesStore } from './filesStore.js';
  let canvas;
  let ctx;
  let digests = [];
  let minCount = 0;
  let maxCount = 1;
  let scale = 1;
  let numCols = 100;
  let offsetX = 0;   // viewport offset
  let offsetY = 0;
  let viewp_ofs = { x: 0, y: 0 }; // viewport offset
  let mipMap = [];
  let isPanning = false;
  let startPan = { x: 0, y: 0 };
  let panOrigin = { x: 0, y: 0 };
  let currentHoveredFile = null;
    
  let latestFetchId = 0;

  hoveredFile.subscribe(async value => {
    currentHoveredFile = value;
    if (currentHoveredFile?.filename) {
      const fetchId = ++latestFetchId;
      const res = await fetch(`http://localhost:8080/api/refchunks?filename=${encodeURIComponent(currentHoveredFile.filename)}`);
      const json = await res.json();

      // Only update if this is the latest fetch
      if (fetchId === latestFetchId && currentHoveredFile && Array.isArray(json.ref_chunks)) {
        currentHoveredFile.ref_chunks = new Set(json.ref_chunks);
        draw();
      }
    } else {
      draw();
    }
  });

  // Fetch digest data from API
  async function fetchDigests() {
    const res = await fetch('http://localhost:8080/api/digests');
    digests = await res.json();
    if (digests.length > 0) {
        minCount = Infinity;
        for (const d of digests) {
        if (d.count < minCount) minCount = d.count;
        }
        maxCount = 0;
        for (const d of digests) {
        if (d.count > maxCount) maxCount = d.count;
        }
        numCols = Math.ceil(Math.sqrt(digests.length))
        mipMap = calcMipmap(digests);        
    }
    
    draw();
  }

function calcMipmap(digests) {
    const mipMap = [];
    const blockSize = 4;
    const rows = Math.ceil(digests.length / numCols);
    const mipCols = Math.ceil(numCols / blockSize);
    const mipRows = Math.ceil(rows / blockSize);

    for (let by = 0; by < mipRows; by++) {
      for (let bx = 0; bx < mipCols; bx++) {
        let sum = 0;
        let count = 0;
        for (let dy = 0; dy < blockSize; dy++) {
          for (let dx = 0; dx < blockSize; dx++) {
            const x = bx * blockSize + dx;
            const y = by * blockSize + dy;
            const idx = y * numCols + x;
            if (idx < digests.length) {
              sum += digests[idx].count;
              count++;
            }
          }
        }
        mipMap.push({ count: count > 0 ? sum / count : 0 });
      }
    }
    return mipMap;
}

  // Map count to color (green to red)
let logBase = 10; // You can change this base as needed

function countToColor(count) {
    // Avoid log(0) and negative values
    const safeMin = Math.max(minCount, 1);
    const safeMax = Math.max(maxCount, safeMin + 1);
    const safeCount = Math.max(count, 1);

    const logMin = Math.log(safeMin) / Math.log(logBase);
    const logMax = Math.log(safeMax) / Math.log(logBase);
    const logCount = Math.log(safeCount) / Math.log(logBase);

    const t = (logCount - logMin) / (logMax - logMin || 1);
    // t=0: red, t=1: green
    const g = Math.round(255 * t);
    const r = Math.round(255 * (1 - t));
    return `rgb(${r},${g},0)`;
}

  // Draw the digest squares
function draw() {
    if (!ctx) return;
    ctx.save();
    ctx.setTransform(1, 0, 0, 1, 0, 0); // Reset
    ctx.clearRect(0, 0, canvas.width, canvas.height);
    ctx.translate(-offsetX, -offsetY);
    ctx.scale(scale, scale);
    const size = 16;

    // Calculate visible area in "world" coordinates
    const viewLeft = offsetX / scale;
    const viewTop = offsetY / scale;
    const viewRight = viewLeft + canvas.width / scale;
    const viewBottom = viewTop + canvas.height / scale;

    if (scale > 0.3) {
        digests.forEach((d, i) => {
            const x = (i % numCols) * size;
            const y = Math.floor(i / numCols) * size;

            // Check if the square is in the viewport
            if (
                x + size < viewLeft ||
                x > viewRight ||
                y + size < viewTop ||
                y > viewBottom
            ) {
                return; // Not visible, skip drawing
            }

            ctx.fillStyle = countToColor(d.count);
            ctx.fillRect(x, y, size - 2, size - 2);
            if (currentHoveredFile?.ref_chunks?.has(d.digest_index)) {
              // draw yeellow border
              ctx.strokeStyle = 'yellow';
              ctx.lineWidth = 2;
              ctx.strokeRect(x, y, size - 2, size - 2);
            }

        });
    } else {
        // Draw mipmap
        const mipSize = size * 4;
        const mipCols = Math.ceil(numCols / 4);
        const mipRows = Math.ceil(mipMap.length / mipCols);
        for (let i = 0; i < mipMap.length; i++) {
            const x = (i % mipCols) * mipSize;
            const y = Math.floor(i / mipCols) * mipSize;

            // Check if the mipmap square is in the viewport
            if (
                x + mipSize < viewLeft ||
                x > viewRight ||
                y + mipSize < viewTop ||
                y > viewBottom
            ) {
                continue; // Not visible, skip drawing
            }

            ctx.fillStyle = countToColor(mipMap[i].count);
            ctx.fillRect(x, y, mipSize, mipSize);
        }
    }
    ctx.restore();
}

  function handleWheel(e) {
    e.preventDefault();
    console.log('wheel', e.offsetX, e.offsetY);
    const mouseX = e.offsetX + offsetX
    const mouseY = e.offsetY + offsetY
    const delta = e.deltaY < 0 ? 1.1 : 0.9;
    scale *= delta;
    // Zoom to mouse position
    offsetX += (mouseX * (delta - 1)) * scale;
    offsetY += (mouseY * (delta - 1)) * scale;
    draw();
  }

  function handleMouseDown(e) {
    isPanning = true;
    canvas.style.cursor = 'grabbing';
    startPan = { x: e.clientX, y: e.clientY };
    panOrigin = { x: offsetX, y: offsetY };
  }
  function handleMouseMove(e) {
    if (!isPanning) return;
    offsetX = panOrigin.x - (e.clientX - startPan.x);
    offsetY = panOrigin.y - (e.clientY - startPan.y);
    // Clamp offsets to prevent scrolling too far
    offsetX = Math.max(-16, offsetX);
    offsetY = Math.max(-16, offsetY);

    draw();
  }
  function handleMouseUp() {
    isPanning = false;
    canvas.style.cursor = 'default';
    offsetX = Math.max(0, offsetX);
    offsetY = Math.max(0, offsetY);
    draw();
  }

  onMount(() => {
    ctx = canvas.getContext('2d');
    fetchDigests();
    window.addEventListener('mouseup', handleMouseUp);
    window.addEventListener('mousemove', handleMouseMove);
    // Redraw on resize
    const resize = () => {
      canvas.width = canvas.clientWidth;
      canvas.height = canvas.clientHeight;
      draw();
    };
    resize();
    window.addEventListener('resize', resize);
    return () => {
      window.removeEventListener('mouseup', handleMouseUp);
      window.removeEventListener('mousemove', handleMouseMove);
      window.removeEventListener('resize', resize);
    };
  });
</script>

<style>
.canvas-container {
  width: 100%;
  height: 800px;
  border: 1px solid #ccc;
  position: relative;
}
canvas {
  width: 100%;
  height: 100%;
  display: block;
}
.file-tooltip {
  position: absolute;
  top: 10px;
  left: 10px;
  background: #fffbe6;
  border: 1px solid #ccc;
  padding: 4px 8px;
  border-radius: 4px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.08);
  z-index: 10;
  pointer-events: none;
}
</style>

<div class="canvas-container">
  <canvas
    bind:this={canvas}
    width="800"
    height="400"
    on:wheel={handleWheel}
    on:mousedown={handleMouseDown}
  ></canvas>
  <div>offsetX:{offsetX} offsetY:{offsetY} scale:{scale}</div>
  {#if currentHoveredFile}
    <div>Chunks: {currentHoveredFile.ref_chunks}</div>
    <div class="file-tooltip">{currentHoveredFile.filename}</div>
  {/if}
</div>
