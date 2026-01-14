# Synthara-Redux

A high-performance Discord music bot written in Go, featuring real-time audio transcoding, comprehensive playback controls, multi-language support, and a modern web interface.

## Overview

Synthara-Redux provides seamless music streaming in Discord voice channels with native transcoding. The bot supports YouTube, Spotify, and Apple Music URLs, includes full Queue management, lyrics display, and a React-based web dashboard for remote control.

## Features

### Music Playback
- **Multi-Platform Support**: YouTube, Spotify, and Apple Music URLs
- **Search Integration**: Natural language search via InnerTube (YouTube Music)
- **Queue Management**: Add, move, remove, jump, and shuffle songs
- **Playback Controls**: Play, pause, resume, next, previous, repeat modes, and seek
- **Album Playback**: Queue entire albums with a single command

### User Experience
- **Localization**: Full support for 11 languages (English, Spanish, Chinese, French, Italian, German, Polish, Russian, Japanese)
- **Web Dashboard**: Real-time React interface for queue viewing, lyrics, playback control, and song details
- **Lyrics Integration**: Synchronized (Word-Synced) lyrics fetched from multiple providers
- **User History**: Records listening history and preferences
- **Web Controls Lock**: Optional security to restrict web-based operations

### Audio Quality
- Native AAC decoding via FDK-AAC (no FFMPEG dependency)
- Real-time sample rate conversion (44.1kHz → 48kHz)
- Opus encoding at 128kbps for Discord streaming
- Low-latency HLS segment buffering

## Audio Processing Pipeline

The bot uses a custom audio transcoding pipeline:

1. **Stream Acquisition**: Fetches HLS manifests from YouTube/InnerTube API and downloads MPEG-TS segments
2. **Demuxing**: Extracts AAC frames from transport stream using `astits`
3. **AAC Decoding**: Decodes to PCM via FDK-AAC (CGO bindings) with automatic resampling
4. **Opus Encoding**: Encodes to Opus at 128kbps/48kHz for Discord
5. **Streaming**: Direct packet transmission to Discord voice gateway

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

## Building the Project

### Prerequisites

**Bot**
- Go 1.24.1+
- FDK-AAC library
- C compiler (GCC/Clang)
- pkg-config

**Web**
- Node.js 18+
- Bun or Deno

### System Dependencies

**Linux (Debian/Ubuntu)**
```bash
sudo apt-get install libfdk-aac-dev pkg-config build-essential
```

**macOS**
```bash
brew install fdk-aac pkg-config
```

**Windows**
```bash
Good Luck... use WSL
```

### Build Script

**Bot**
```bash
chmod +x build.sh
./build.sh
```

**Web**
```bash
cd Web
bun install
bun run build
```

## Running Everything

```bash
./synthara-redux
```

The bot will initialize Discord gateway, register commands, and start the web server on port `3000` or whatever is specified in the `PORT` environment variable.

## Available Commands

- `/play <query>` - Search and play a song (supports URLs)
- `/pause` / `/resume` - Control playback
- `/next` / `/last` - Navigate queue
- `/jump <position>` - Jump to specific song
- `/replay <position>` - Restart from specific position
- `/repeat <mode>` - Set repeat mode (Off/One/All)
- `/shuffle <enabled>` - Toggle shuffle
- `/queue` - View current queue
- `/move <song> <position>` - Reorder queue
- `/lyrics` - Display synchronized lyrics
- `/album` - Play entire album
- `/controls` - Get web dashboard link
- `/lock` / `/unlock` - Manage web control permissions
- `/stats` - View listening statistics
- `/forget` - Clear your listening history
- `/leave` - Disconnect from voice channel

## Technical Stack

**Bot**
- Go 1.24.1
- disgoorg/disgo (Discord library)
- FDK-AAC (AAC decoding via CGO)
- gopus (Opus encoding)
- innertube-go + custom scrapers

**Web**
- React 19
- TypeScript
- Vite
- Tailwind CSS
- WebSocket (real-time updates)