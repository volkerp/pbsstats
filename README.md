# PBS Stats

This program scans a directory tree for Proxmox Backup Server index files (`.fidx` and `.didx`), analyzes chunk deduplication, and reports statistics about chunk usage and file deduplication ratios.

## Features
- Scans all `.fidx` and `.didx` files in a directory recursively
- Counts how often each chunk digest appears across all files
- Calculates deduplication ratio for each file (total chunks / unique chunks)
- Reports the top N most referenced chunks and the top N files with the highest deduplication ratio

## Usage
```
scan_fidx [--top-chunks N] [--top-files N] <directory>
```
- `--top-chunks N`: Show the top N most referenced chunks (default: 50)
- `--top-files N`: Show the top N files with the highest deduplication ratio (default: 50)
- `<directory>`: Root directory to scan for `.fidx` and `.didx` files

## Example
```
scan_fidx --top-chunks 100 --top-files 20 /path/to/datastore
```

This will print the 100 most referenced chunk digests and the 20 files with the highest deduplication ratio in the specified directory tree.
