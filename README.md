# psp-metadata

[![CI](https://github.com/ghostdiverr/psp-metadata/actions/workflows/test.yml/badge.svg)](https://github.com/ghostdiverr/psp-metadata/actions/workflows/test.yml)
[![Release](https://img.shields.io/github/v/release/ghostdiverr/psp-metadata?logo=github)](https://github.com/ghostdiverr/psp-metadata/releases/latest)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A command-line tool for extracting and displaying metadata from PlayStation Portable (PSP) game ROMs.

Supports **ISO** and **CSO** (Compressed ISO) formats.

## Features

- Extract metadata from PSP ROMs (title, region, disc ID, firmware required, etc.)
- Extract embedded artwork (ICON0, PIC0, PIC1)
- Rename ROM files to a clean, predictable format (Title [DISC_ID].ext)
- Display metadata in human-readable text or structured JSON
- Verbose/diagnostic mode for troubleshooting

## Formats Supportés

| Format | Description |
|--------|-------------|
| ISO | Standard PSP ROM image |
| CSO | Compressed ISO (smaller file size) |

## Installation

### Binaires Pré-compilés

Download the latest release for your platform from the [Releases](https://github.com/ghostdiverr/psp-metadata/releases/latest) page.

### Build from Source

```bash
git clone https://github.com/ghostdiverr/psp-metadata.git
cd psp-metadata
go build .
```

Or install directly with Go:

```bash
go install github.com/ghostdiverr/psp-metadata@latest
```

## Usage

### Display Metadata

```bash
psp-metadata info <file.iso|file.cso>
```

**Text output (default):**

```
$ psp-metadata info "My Game.iso"
Field            Value
────────────────── ──────────────────────────────────
Title            Crisis Core: Final Fantasy VII
Game ID          ULUS-10336
Region           North America
Category         UMD Game
Disc Version     1.00
Min. Firmware    6.60
Parental Level   12
File Size        920.31 MB (966272000 bytes)

Artwork          Status
────────────────── ──────
ICON0.PNG (144x80)  [YES]
PIC0.PNG  (480x272) [YES]
PIC1.PNG  (480x272) [NO]
```

### JSON Output

```bash
psp-metadata info --json <file.iso|file.cso>
```

**JSON output:**

```json
{
  "title": "Crisis Core: Final Fantasy VII",
  "disc_id": "ULUS-10336",
  "region": "North America",
  "category": "UMD Game",
  "disc_version": "1.00",
  "system_version": "6.60",
  "parental_level": 12,
  "file_size_bytes": 966272000,
  "artworks": {
    "icon0": true,
    "pic0": true,
    "pic1": false
  }
}
```

### Extract Artwork

```bash
# Extract to current directory
psp-metadata extract game.iso

# Extract to a specific directory
psp-metadata extract -o ./assets game.iso
```

### Rename ROM Files

```bash
# Preview what would be renamed (dry run)
psp-metadata rename -n game.iso
#> Would rename: "game.iso" -> "Crisis Core - Final Fantasy VII [ULUS10336].iso"

# Perform the rename
psp-metadata rename game.iso
```

## Options Globales

| Option | Description |
|--------|-------------|
| `--verbose` / `-v` | Enable diagnostic output with timing information |
| `--help` / `-h` | Show help for any command |

## Tableau Récapitulatif des Formats de Sortie

| Format | Commande | Description |
|--------|----------|-------------|
| **Texte** | `info <fichier>` | Tableau lisible avec métadonnées et artworks |
| **JSON** | `info --json <fichier>` | Structure JSON exploitable par machine |
| **PNG** | `extract <fichier>` | Artworks extraits (ICON0, PIC0, PIC1) |
| **Texte Preview** | `rename -n <fichier>` | Prévisualisation du renommage avant action |

## Diagnostic Mode

Use the `--verbose` flag to see detailed timing and diagnostic information:

```bash
$ psp-metadata -v info game.iso
[verbose] file size on disk: 966272000 bytes
[verbose] iso.Open done in 2ms
[verbose] reading PARAM.SFO...
[verbose] ReadSFO done in 15ms
[verbose] parsing SFO...
[verbose] Parse done in 0ms
[verbose] detecting artworks...
[verbose] DetectArtworks done in 3ms
# ...metadata output follows...
```

## License

This project is licensed under the [MIT License](LICENSE).

Copyright (c) 2024 ghostdiverr
