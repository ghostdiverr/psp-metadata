# psp-metadata

A command-line tool for working with PlayStation Portable (PSP) game metadata.

## Features

- Extract metadata and artworks from PSP game ROMs (ISO and CSO)
- Rename ROM files based on extracted metadata
- View detailed game information

## Installation

```bash
go install github.com/ghostdiverr/psp-metadata@latest
```

Or build from source:

```bash
git clone <repo>
cd psp-metadata
go build .
```

## Usage

```bash
# Show help
psp-metadata --help

# Extract metadata and artworks from a ROM
psp-metadata extract <rom-file>

# Show game info
psp-metadata info <rom-file>

# Rename a ROM based on its metadata
psp-metadata rename <rom-file>
```

## License

TBD
