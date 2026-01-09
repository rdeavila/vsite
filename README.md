# vsite

Static HTML video gallery generator.

## Description

`vsite` is a command-line tool written in Go that scans a directory
containing video files and generates static HTML pages for browsing and
playing videos in the browser.

## Features

- Recursive scanning of directories and subdirectories
- Generation of listing pages with folder navigation
- Integrated video player (Plyr) with advanced controls
- Automatic conversion of incompatible formats to MP4
- NVIDIA GPU acceleration support (NVENC)
- Responsive design with dark theme
- Video navigation (previous/next)
- Keyboard shortcuts in player
- Auto-play next video

## Installation

### Building from source

```bash
git clone https://github.com/your-user/vsite.git
cd vsite
make build
```

### Build targets

```bash
make              # Build optimized binary (default)
make build        # Build optimized binary
make build-debug  # Build with debug symbols
make build-all    # Build for all platforms
make build-linux  # Build for Linux amd64
make build-darwin # Build for macOS arm64
make build-windows# Build for Windows amd64
make clean        # Remove build artifacts
make test         # Run tests
make deps         # Download dependencies
make install      # Install to /usr/local/bin
make install-user # Install to ~/.local/bin
make uninstall    # Uninstall
make info         # Show binary information
make compress     # Compress with UPX
make serve        # Start HTTP server with video seeking support
make help         # Show help
```

### Optional dependencies

To use video conversion (`--convert`):

```bash
# Debian/Ubuntu
sudo apt install ffmpeg

# Fedora/RHEL
sudo dnf install ffmpeg
```

## Usage

### Basic syntax

```bash
vsite [options] <directory>
```

### Options

| Option | Description |
|--------|-------------|
| `-t, --title <text>` | Sets the title of the main page (default: "Videos") |
| `--convert` | Converts incompatible videos (avi, mkv, mov) to MP4 |
| `--gpu` | Uses NVIDIA GPU (NVENC) for faster conversion |
| `-c, --clean` | Removes all generated HTML files from the directory |
| `--clean-converted` | Removes converted MP4 files (keeps originals) |
| `-h, --help` | Shows help |
| `-v, --version` | Shows version |

### Examples

Generate HTML gallery:

```bash
vsite /path/to/videos
```

Generate with custom title:

```bash
vsite --title "My Collection" /path/to/videos
```

Convert incompatible videos and generate HTML:

```bash
vsite --convert /path/to/videos
```

Convert using NVIDIA GPU:

```bash
vsite --convert --gpu /path/to/videos
```

Clean generated HTML files:

```bash
vsite --clean /path/to/videos
```

Remove converted MP4 files (keep originals):

```bash
vsite --clean-converted /path/to/videos
```

## Serving videos

To play videos with seeking support (clicking on the progress bar), you need
an HTTP server that supports range requests.

### Using make serve

```bash
cd /path/to/videos
make -C /path/to/vsite serve
```

This starts `http-server` on port 8000 with range request support.

### Alternative servers

```bash
# Node.js http-server
npx http-server -p 8000

# Python with range support
pip install rangehttpserver
python -m RangeHTTPServer 8000
```

Note: Python's built-in `http.server` does NOT support range requests,
which means video seeking will not work properly.

## Video formats

### Natively supported by browsers

- `.mp4` (H.264)
- `.webm` (VP8/VP9)
- `.ogv` (Theora)

### Require conversion

- `.avi`
- `.mkv`
- `.mov`
- `.wmv`
- `.flv`

Use the `--convert` option to automatically convert these formats to MP4.

## Generated files

HTML files are created directly in the video directory:

```text
/your/directory/
├── video1.mp4
├── video2.mp4
├── subfolder/
│   └── video3.mp4
├── index.html              # Main page
├── style.css               # CSS styles
├── subfolder_index.html    # Subfolder index
├── player_video1.html      # video1 player
├── player_video2.html      # video2 player
└── player_subfolder_video3.html
```

## GPU conversion

The `--gpu` option uses NVIDIA's NVENC encoder to accelerate video
conversion up to 10x compared to CPU.

### Requirements

1. NVENC-compatible NVIDIA GPU
2. NVIDIA driver installed (`nvidia-smi` must work)
3. ffmpeg compiled with NVENC support

### NVIDIA driver installation

```bash
# Debian/Ubuntu
sudo apt install nvidia-driver-535

# Fedora/RHEL
sudo dnf install akmod-nvidia
```

### Conversion parameters

#### CPU (libx264)

| Parameter | Value |
|-----------|-------|
| Codec | H.264 (libx264) |
| Preset | fast |
| CRF | 22 |
| Audio | AAC 128kbps |

#### GPU (NVENC)

| Parameter | Value |
|-----------|-------|
| Codec | H.264 (h264_nvenc) |
| Preset | p4 |
| CQ | 23 |
| Audio | AAC 128kbps |

## Player

The player uses the [Plyr](https://plyr.io/) library and offers:

- Playback controls (play, pause, volume, fullscreen)
- Progress bar with preview
- Speed control (0.5x to 2x)
- Picture-in-Picture
- Navigation between videos in the same directory

### Keyboard shortcuts

| Key | Action |
|-----|--------|
| Space | Play/Pause |
| Left/Right arrows | Seek 5 seconds |
| Up/Down arrows | Volume |
| M | Mute/Unmute |
| F | Fullscreen |
| Esc | Back to listing |

## Project structure

```text
vsite/
├── main.go                 # CLI entry point
├── go.mod                  # Go module
├── Makefile                # Build automation
├── README.md               # Documentation
├── LICENSE                 # MIT License
└── generator/
    ├── generator.go        # HTML generation logic
    └── templates/
        ├── index.html      # Listing template
        └── player.html     # Player template
```

## License

MIT License - see [LICENSE](LICENSE) file for details.
