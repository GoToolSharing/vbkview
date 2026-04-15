# vbkview

`vbkview` is a CLI to inspect and extract data from Veeam `.vbk` backup files.

It is built on top of [`vbktoolkit`](https://github.com/GoToolSharing/vbktoolkit) and supports both:

- interactive shell mode
- non-interactive commands (automation/CI-friendly)

## Features

- Browse VBK internal filesystem (`ls`, `tree`, `stat`)
- Read file content (`cat`)
- Extract files (`get`)
- Find files by name (`find`)
- Search text in files (`grep`)
- List embedded virtual disk files (`disks`)
- JSON output mode for automation (`--json`)
- Stable exit codes for common error categories

## Installation

### Install latest

```bash
go install github.com/GoToolSharing/vbkview/cmd/vbkview@latest
```

### Build locally

```bash
git clone https://github.com/GoToolSharing/vbkview
cd vbkview
go build -o vbkview ./cmd/vbkview
```

## Quick Start

```bash
vbkview --help

# Interactive shell
vbkview shell --vbk /path/to/backup.vbk

# List root directory
vbkview ls --vbk /path/to/backup.vbk /
```

## Global Flags

- `-f, --vbk` path to `.vbk` file (required for data commands)
- `--verify` verify VBK metadata CRCs (default `true`)
- `--cwd` working directory inside VBK for relative paths
- `--json` JSON output (when supported)

## Commands

- `shell` start interactive mode
- `pwd` print current directory
- `ls [path]` list directory content (`-l` for long format)
- `cat <path>` print file content (`--limit`, `--base64`)
- `get <src> [dst]` extract file (`--resume`, `--sha256`)
- `find <name> [start]` find files by name
- `stat [path]` show metadata (`--props`)
- `tree [path]` print directory tree (`--depth`)
- `grep <pattern> [start]` search text (`-i`, `--max-bytes`)
- `disks` list `.vhd`/`.vhdx` entries
- `volumes` list available volumes abstraction

## Examples

### Non-interactive usage

```bash
# Long listing
vbkview ls -l --vbk /path/to/backup.vbk /some/dir

# Metadata as JSON
vbkview stat --vbk /path/to/backup.vbk --json /summary.xml

# Tree up to depth 2
vbkview tree --vbk /path/to/backup.vbk --depth 2 /

# Find files (JSON)
vbkview find --vbk /path/to/backup.vbk --json summary /

# Grep case-insensitive in first 1 MiB per file
vbkview grep --vbk /path/to/backup.vbk -i --max-bytes 1048576 "error|warning" /
```

### Extraction workflows

```bash
# Simple extraction
vbkview get --vbk /path/to/backup.vbk /path/in/vbk/file.bin ./file.bin

# Resume extraction if output already exists
vbkview get --vbk /path/to/backup.vbk --resume /path/in/vbk/file.bin ./file.bin

# Extract + integrity verification
vbkview get --vbk /path/to/backup.vbk --sha256 <expected_sha256> /path/in/vbk/file.bin ./file.bin

# JSON extraction result
vbkview get --vbk /path/to/backup.vbk --json /path/in/vbk/file.bin ./file.bin
```

### Interactive shell

```text
vbk[/]> ls
vbk[/]> cd some/folder
vbk[/some/folder]> ll
vbk[/some/folder]> stat file.txt
vbk[/some/folder]> grep "password" /
vbk[/some/folder]> get file.txt ./file.txt
vbk[/some/folder]> quit
```

## Exit Codes

- `0` success
- `1` generic error
- `2` usage/argument error
- `3` path not found
- `4` not a directory
- `5` is a directory (when file expected)
- `6` unsupported data/feature
- `7` invalid VBK input/metadata

## Development

```bash
go test ./...
go build ./cmd/vbkview
```

If you work locally with `vbktoolkit`, use a `replace` directive in `go.mod`:

```go
replace github.com/GoToolSharing/vbktoolkit => ../vbktoolkit
```
