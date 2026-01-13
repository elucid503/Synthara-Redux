import { Song, LyricsResponse, Operation } from '../Types';

// Normalizes Google cover URLs to ensure 512x512 dimensions
export const NormalizeCoverURL = (URL: string): string => {

    return URL.replace(/=w\d+-h\d+(-l\d+)?(-rj)?/g, '=w512-h512-l90-rj');

};

// Formats time from seconds to mm:ss format
export const FormatTime = (Seconds: number): string => {

    const Mins = Math.floor(Seconds / 60);
    const Secs = Math.floor(Seconds % 60);

    return `${Mins}:${Secs.toString().padStart(2, '0')}`;

};

// Sends WebSocket operations to the server
export const SendOperation = (Socket: WebSocket | null, OperationType: Operation, Params: { [key: string]: any } = {}) => {

    if (!Socket || Socket.readyState != WebSocket.OPEN) return;
    
    const Message: any = { Operation: OperationType };

    Object.keys(Params).forEach((Key) => {

        Message[Key] = Params[Key];

    });
    
    Socket.send(JSON.stringify(Message));

};


// Fetches lyrics from the API we've decided to use
export const FetchLyrics = async (Song: Song): Promise<{ data: LyricsResponse | null, error: boolean }> => {

    const CleanedTitle = Song.title.replace(/\s*\(.*?\)/g, '').trim();
    const OriginalTitle = Song.title;

    const Result = await LyricsFetcher(CleanedTitle, Song.artists[0]);

    if (Result.error && CleanedTitle != OriginalTitle) {

        return await LyricsFetcher(OriginalTitle, Song.artists[0]);

    }

    return Result;

};

// Helper function to fetch lyrics with specific title/artist/album strings
const LyricsFetcher = async (Title: string, Artist: string): Promise<{ data: LyricsResponse | null, error: boolean }> => {

    try {

        const Params = new URLSearchParams({

            title: Title,
            artist: Artist,

            source: 'apple,lyricsplus,musixmatch,spotify,musixmatch-word'

        });
        
        const Response = await fetch(`https://lyricsplus.prjktla.workers.dev/v2/lyrics/get?${Params}`);
        
        if (Response.ok) {

            const Data: LyricsResponse = await Response.json();
            return { data: Data, error: false };

        } else {

            return { data: null, error: true };

        }

    } catch (Error) {

        console.error('Error fetching lyrics:', Error);

        return { data: null, error: true };

    }

};