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