// Server-Defined Types

export interface Song {

    youtube_id: string;

    title: string;
    artists: string[];
    album: string;

    duration: {

        seconds: number;
        formatted: string;
        
    };

    cover: string;

}

export enum PlayerState {

    Idle = 0,
    Playing = 1,
    Paused = 2,

}

export enum WSEvents {

    Event_Initial = "INITIAL_STATE",

    Event_StateChanged = "STATE_CHANGED",
    Event_QueueUpdated = "QUEUE_UPDATED",
    Event_ProgressUpdate = "PROGRESS_UPDATE",

}

export enum Operation {

    Pause = "Pause",
    Resume = "Resume",

    Next = "Next",
    Last = "Last",

    Seek = "Seek", // Takes in a number (Offset)

}

export interface WSMessage<T> {

    Event: WSEvents;
    Data: T;

}

// Lyrics Types

export interface LyricsSyllabus {

    time: number;
    duration: number;

    text: string;

}

export interface LyricsLine {

    time: number;
    duration: number;

    text: string;
    syllabus?: LyricsSyllabus[];

    element: {

        key?: string;
        songPart?: string;
        singer?: string;

    };

}

export interface LyricsMetadata {

    source: string;

    songWriters?: string[];
    title?: string;

    language?: string;

    totalDuration?: string;
    leadingSilence?: string;

}

export interface LyricsResponse {

    type: "Word" | "Line";

    metadata: LyricsMetadata;

    lyrics: LyricsLine[];

    cached?: string;

    processingTime?: {

        timeElapsed: number;
        lastProcessed: number;

    };

}