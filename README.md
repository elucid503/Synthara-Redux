# Synthara-Redux

A Go rewrite of the Synthara music bot, aiming to provide efficient and convenient music with friends. 

## Overview

Synthara-Redux is a Discord bot that provides music streaming capabilities with real-time audio transcoding. The bot searches YouTube for music, downloads and processes audio streams, and plays them directly in Discord voice channels with minimal latency and high audio quality.

## Audio Processing Pipeline

The bot implements a sophisticated multi-stage audio processing pipeline:

### 1. Stream Acquisition
- Uses InnerTube (YouTube's internal API) and a custom scraper to search for songs and retrieve HLS manifests
- Downloads MPEG-TS segments containing AAC-encoded audio

### 2. Demuxing & Frame Extraction
- Parses MPEG-TS containers using `astits` demuxer
- Identifies and extracts AAC audio frames from transport stream packets
- Handles ADTS (Audio Data Transport Stream) framing

### 3. AAC Decoding
- Decodes AAC frames to raw PCM audio using FDK-AAC library via CGO
- Supports automatic sample rate conversion (typically 44.1kHz → 48kHz)
- Uses linear interpolation for high-quality resampling

### 4. Opus Encoding
- Encodes PCM audio to Opus format at 128kbps bitrate
- Configured for 48kHz sample rate, stereo, 20ms frame size
- Optimized for voice/music streaming (960 samples per frame)

### 5. Discord Streaming
- Transmits Opus packets directly to Discord voice gateway
- Maintains low latency buffering and synchronization

## CGO Integration

This project makes extensive use of CGO to interface with native C libraries for better audio processing performance.

### FDK-AAC Decoder

The bot uses FDK-AAC (Fraunhofer FDK AAC) through CGO bindings for AAC decoding:

```go
#cgo pkg-config: fdk-aac
#include <fdk-aac/aacdecoder_lib.h>
```

**Why FDK-AAC?**
- Industry-standard AAC decoder with exceptional quality
- Superior performance compared to pure Go implementations
- Robust handling of various AAC profiles and formats
- Required for decoding YouTube's audio streams

The decoder handles:
- ADTS frame parsing and validation
- Multi-channel audio decoding
- Stream info extraction (sample rate, channels, frame size)
- Memory-safe buffer management

### Build Considerations

CGO requires a C compiler and introduces platform-specific build requirements:

- **Build tags**: `//go:build linux || darwin || windows` ensures cross-platform compatibility
- **pkg-config**: Used to locate FDK-AAC headers and libraries
- **Compiler flags**: Optimizations enabled via `CGO_CFLAGS`

## Environment Configuration

Create a `.env` file in the project root with the following variables:

```env
# Required: Discord bot token from Discord Developer Portal
DISCORD_TOKEN=your_discord_bot_token_here

# Optional: YouTube cookie string from authenticated YouTube session
YOUTUBE_COOKIE=your_youtube_cookie_here

# Optional: Force application-command registration on startup (set to "true" to refresh)
REFRESH_COMMANDS=false
```

### Obtaining Configuration Values

**Discord Token:**
1. Go to [Discord Developer Portal](https://discord.com/developers/applications)
2. Create a new application or select an existing one
3. Navigate to the "Bot" section
4. Click "Reset Token" or "Copy" to get your bot token
5. Enable the following Privileged Gateway Intents:
   - Server Members Intent (optional, for member info)
   - Message Content Intent (if processing messages)

**YouTube Cookie (Optional):**
- Required only for age-restricted content or member-only videos
- Export cookies from your browser while logged into YouTube
- Use a browser extension like "EditThisCookie" or "Cookie-Editor"
- Copy the entire cookie string and paste it in the `.env` file

**Command Registration:**
- Set `REFRESH_COMMANDS=true` on first run or when commands are modified
- Set to `false` for normal operation to avoid unnecessary API calls

## Building the Project

### Prerequisites

1. **Go 1.24.1 or later**
   ```bash
   go version
   ```

2. **FDK-AAC library** (required for CGO)
   
   **Linux (Debian/Ubuntu):**
   ```bash
   sudo apt-get update
   sudo apt-get install libfdk-aac-dev pkg-config
   ```
   
   **macOS:**
   ```bash
   brew install fdk-aac pkg-config
   ```
   
   **Windows:**
   - Install MSYS2 from https://www.msys2.org/
   - Open MSYS2 MinGW 64-bit terminal:
     ```bash
     pacman -S mingw-w64-x86_64-fdk-aac mingw-w64-x86_64-pkg-config mingw-w64-x86_64-gcc
     ```
   - Add MinGW bin to PATH: `C:\msys64\mingw64\bin`

3. **C Compiler (GCC or Clang)**
   - Linux: Usually pre-installed or via `build-essential`
   - macOS: Install Xcode Command Line Tools
   - Windows: Provided by MSYS2 (see above)

### Build Script

The project includes a build script that sets optimal CGO flags:

```bash
chmod +x build.sh
./build.sh
```

The build script:
- Enables CGO explicitly
- Sets `-O3` optimization for C code
- Suppresses false-positive compiler warnings
- Produces optimized binary: `synthara-redux`

### Manual Build

Alternatively, build manually:

```bash
export CGO_ENABLED=1
export CGO_CFLAGS="-O3"
go build -v -o synthara-redux
```

## Running the Bot

```bash
./synthara-redux
```

The bot will:
1. Load environment variables from `.env`
2. Initialize Discord client with gateway connection
3. Register slash commands (if `REFRESH_COMMANDS=true`)
4. Initialize InnerTube client for YouTube API access
5. Start listening for commands

## Usage

**Play Command:**
```
/play query:<song name or artist>
```

The bot will:
- Search YouTube Music for the query
- Join your voice channel
- Stream the top result with real-time transcoding

## Technical Stack

- **Language**: Go 1.24.1
- **Discord Library**: disgoorg/disgo
- **Audio Decoding**: FDK-AAC (via CGO)
- **Audio Encoding**: gopus (Opus encoder)
- **YouTube API**: innertube-go + Overture-Play
- **Container Parsing**: astits (MPEG-TS), joy4 (AAC parsing)