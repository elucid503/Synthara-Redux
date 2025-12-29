import { useEffect, useState, useMemo } from 'react';
import { Play, Pause, SkipBack, SkipForward, Music, MoreHorizontal, Trash2, CornerDownRight, ChevronDown, ChevronUp } from 'lucide-react';

import { Song, PlayerState, WSEvents, WSMessage, Operation, LyricsResponse } from './Types';

function App() {

    const [Socket, SetSocket] = useState<WebSocket | null>(null);
   
    const [CurrentSong, SetCurrentSong] = useState<Song | null>(null);
    const [PreviousSongs, SetPreviousSongs] = useState<Song[]>([]);
    const [UpcomingSongs, SetUpcomingSongs] = useState<Song[]>([]);

    const [ShowPrevious, SetShowPrevious] = useState(false);

    const [ActiveContextMenu, SetActiveContextMenu] = useState<{ type: 'Previous' | 'Upcoming', index: number, x: number, y: number } | null>(null);

    // Close context menu on click outside or scroll

    useEffect(() => {

        const HandleClickOutside = (E: MouseEvent) => {

            if (ActiveContextMenu) {

                const Target = E.target as HTMLElement;

                if (!Target.closest('.context-menu-trigger') && !Target.closest('.context-menu-content')) {
                   
                    SetActiveContextMenu(null);

                }

            }

        };

        const HandleScroll = () => {

            if (ActiveContextMenu) {

                SetActiveContextMenu(null);

            }

        };

        window.addEventListener('click', HandleClickOutside);
        window.addEventListener('scroll', HandleScroll, true);

        return () => {

            window.removeEventListener('click', HandleClickOutside);
            window.removeEventListener('scroll', HandleScroll, true);

        };

    }, [ActiveContextMenu]);

    const [PlayerStateValue, SetPlayerStateValue] = useState<PlayerState>(PlayerState.Idle);
    const [CurrentTime, SetCurrentTime] = useState(0);
    
    const [ActiveView, SetActiveView] = useState<'Details' | 'Queue' | 'Lyrics'>(() => {

        const Params = new URLSearchParams(window.location.search);
        const View = Params.get('View');

        if (View == 'Lyrics') return 'Lyrics';
        if (View == 'Queue') return 'Queue';

        return 'Details';

    });

    const [BackgroundImage, SetBackgroundImage] = useState<string>('');

    const [Lyrics, SetLyrics] = useState<LyricsResponse | null>(null);
    const [LyricsLoading, SetLyricsLoading] = useState(false);
    const [LyricsError, SetLyricsError] = useState(false);
    
    // Normalizes Google cover URLs to ensure 512x512 dimensions

    const NormalizeCoverURL = (URL: string): string => {

        return URL.replace(/=w\d+-h\d+(-l\d+)?(-rj)?/g, '=w512-h512-l90-rj');

    };

    const FetchLyrics = async (Song: Song) => {

        SetLyricsLoading(true);
        SetLyricsError(false);

        try {

            const Params = new URLSearchParams({

                title: Song.title.replace(/\s*\(.*?\)/g, '').trim(), // removes info in parentheses
                artist: Song.artists[0],
                album: Song.album,

                source: 'apple,lyricsplus,musixmatch,spotify,musixmatch-word'

            });
            
            const Response = await fetch(`https://lyricsplus.prjktla.workers.dev/v2/lyrics/get?${Params}`);
            
            if (Response.ok) {

                const Data: LyricsResponse = await Response.json();
                SetLyrics(Data);

            } else {

                SetLyrics(null);
                SetLyricsError(true);

            }

        } catch (Error) {

            console.error('Error fetching lyrics:', Error);
            
            SetLyrics(null);
            SetLyricsError(true);

        } finally {

            SetLyricsLoading(false);

        }

    };

    // WebSocket connection

    useEffect(() => {

        const Params = window.location.href.split('/');
        const QueueID = Params[Params.length - 1].split('?')[0]; // ignores the query params

        const WS = new WebSocket(`ws://localhost:3000/API/Queue?ID=${QueueID}`);

        WS.onopen = () => {

            console.log('WebSocket connected');

        };

        WS.onmessage = (Event) => {

            const Message: WSMessage<any> = JSON.parse(Event.data);
            
            switch (Message.Event) {

                case WSEvents.Event_Initial:

                    SetCurrentSong(Message.Data.Current);
                    SetPreviousSongs(Message.Data.Previous || []);
                    SetUpcomingSongs(Message.Data.Upcoming || []);
                    SetPlayerStateValue(Message.Data.State);
                    
                    const InitialProgress = Message.Data.Progress * 1000; // convert seconds to ms
                    SetCurrentTime(InitialProgress);

                break;
                    
                case WSEvents.Event_StateChanged:

                    SetPlayerStateValue(Message.Data.State);

                break;
                    
                case WSEvents.Event_QueueUpdated:

                    SetCurrentSong(Message.Data.Current);
                    SetPreviousSongs(Message.Data.Previous || []);
                    SetUpcomingSongs(Message.Data.Upcoming || []);
                    SetCurrentTime(0);

                    SetLyrics(null);

                break;

                case WSEvents.Event_ProgressUpdate:
                    
                    SetCurrentTime(Math.max(0, (Message.Data.Progress - 250))); // slight buffer to account for latency

                break;

            }

        };

        WS.onerror = (Error) => {

            console.error('WebSocket error:', Error);

        };

        WS.onclose = () => {

            console.log('WebSocket disconnected.');

        };

        SetSocket(WS);

        return () => {

            WS.close();

        };

    }, []);

    // Updates background & lyrics when song changes

    useEffect(() => {

        if (CurrentSong) {

            SetBackgroundImage(NormalizeCoverURL(CurrentSong.cover));
            
            ActiveView == 'Lyrics' && FetchLyrics(CurrentSong);

        }

    }, [CurrentSong]);

    // Progress tracking

    useEffect(() => {

        let Interval: any;

        if (PlayerStateValue == PlayerState.Playing) {

            Interval = setInterval(() => {

                SetCurrentTime((Prev) => Prev + 50);

            }, 50);

        }

        return () => clearInterval(Interval);

    }, [PlayerStateValue]);

    // Fetch lyrics when switching to Lyrics view

    useEffect(() => {

        if (ActiveView == 'Lyrics' && CurrentSong && !Lyrics && !LyricsLoading) {

            FetchLyrics(CurrentSong);

        }
        
    }, [ActiveView, CurrentSong]);

    // Lyrics Logic

    const CurrentLineIndex = useMemo(() => {

        if (!Lyrics || !Lyrics.lyrics) return -1;

        // Finds the last line that has started. This is ideal, since we only display one line at a time

        for (let i = Lyrics.lyrics.length - 1; i >= 0; i--) {

            if (CurrentTime >= Lyrics.lyrics[i].time) {

                return i;

            }

        }

        return -1;

    }, [Lyrics, CurrentTime]);

    const RenderSong = (Song: Song, Mode: 'Big' | 'Normal' | 'Muted', Index: number = 0, Key?: string) => {

        const IsBig = Mode == 'Big';
        const IsPrevious = Mode == 'Muted';
        const ContextType = IsPrevious ? 'Previous' : 'Upcoming';

        if (IsBig) {

            return (

                <div key={Key} className="flex items-center gap-4 p-4 rounded-xl bg-white/10">
                    
                    <img src={NormalizeCoverURL(Song.cover)} referrerPolicy='no-referrer' className="w-16 h-16 rounded-lg object-cover shadow-lg" />
                        
                    <div className="flex-1 min-w-0">

                        <div className="text-lg font-bold truncate">{Song.title}</div>
                        <div className="text-zinc-400 truncate">{Song.artists.join(', ')}</div>
                    
                    </div>

                    <div className="text-zinc-400 mr-2 font-semibold">{Song.duration.formatted}</div>
                
                </div>

            );

        }

        return (

            <div key={Key} className={`flex items-center gap-4 p-3 rounded-lg bg-white/5 hover:bg-white/10 transition-colors group relative ${IsPrevious ? 'opacity-60' : ''}`}>
                
                {!IsPrevious && (<div className="p-1 text-center text-zinc-500 font-medium text-sm">{Index + 1}</div>)}

                <img src={NormalizeCoverURL(Song.cover)} referrerPolicy='no-referrer' className="w-10 h-10 rounded object-cover" />
                
                <div className="flex-1 min-w-0">

                    <div className="font-medium truncate text-sm">{Song.title}</div>
                    <div className="text-xs text-zinc-400 truncate">{Song.artists.join(', ')}</div>
                
                </div>
                
                <div className="text-xs font-semibold text-zinc-500 w-12 text-right">{Song.duration.formatted}</div>

                <button onClick={(E) => { 
                    
                    E.stopPropagation(); 
                    const Rect = E.currentTarget.getBoundingClientRect();
                    
                    SetActiveContextMenu(ActiveContextMenu?.index === Index && ActiveContextMenu?.type === ContextType ? null : { type: ContextType, index: Index, x: Rect.right, y: Rect.bottom }); 
                
                }} className="mr-2 text-zinc-400 hover:text-white transition-colors context-menu-trigger" >
                    
                    <MoreHorizontal size={16} />

                </button>

            </div>

        );

    };

    const RenderQueue = () => {

        return (

            <div className="w-full h-fit max-w-3xl mx-auto">
                
                {/* Previous Songs Toggle */}

                {PreviousSongs.length > 0 && (

                    <div className="mb-6">

                        <button onClick={() => SetShowPrevious(!ShowPrevious)} className="flex items-center gap-2 text-xs font-bold text-zinc-500 uppercase tracking-wider hover:text-white transition-colors">
                            
                            Previous
                            {ShowPrevious ? <ChevronUp className='mb-0.5' size={16} /> : <ChevronDown className='mb-0.5' size={16} />}

                        </button>

                        {ShowPrevious && (

                            <div className="mt-4 space-y-2">

                                {PreviousSongs.map((Song, Index) => RenderSong(Song, 'Muted', Index, `prev-${Index}`))}
                            
                            </div>

                        )}

                    </div>

                )}

                {/* Current Song */}

                <div className="mb-8">

                    <h2 className="text-xs font-bold text-zinc-500 uppercase tracking-wider mb-4">Now Playing</h2>
                    {CurrentSong && RenderSong(CurrentSong, 'Big')}
                
                </div>

                {/* Upcoming Songs */}

                {UpcomingSongs.length > 0 && (

                    <div>

                        <h2 className="text-xs font-bold text-zinc-500 uppercase tracking-wider mb-4">Next Up</h2>
                        
                        <div className="space-y-2">
                            
                            {UpcomingSongs.map((Song, Index) => RenderSong(Song, 'Normal', Index, `next-${Index}`))}
                    
                        </div>
                            
                    </div>
                        
                )}

            </div>

        );

    };

    const RenderLyrics = () => {

        if (LyricsError) {

            return (

                <div className="min-h-[200px] flex items-center justify-center">

                    <div className="text-zinc-500">No Lyrics available.</div>

                </div>

            );

        }

        if (!Lyrics || CurrentLineIndex == -1) {

            if (Lyrics && Lyrics.lyrics.length > 0 && CurrentTime < Lyrics.lyrics[0].time) {

                return (

                    <div className="min-h-[200px] flex flex-col items-center justify-center animate-pulse">
                        
                        <Music size={64} className="text-zinc-500" />
                    
                    </div>

                );

            }

            return (

                <div className="min-h-[200px] flex items-center justify-center">

                    <div className="text-zinc-500 animate-pulse">Loading Lyrics...</div>

                </div>

            );

        }

        const CurrentLine = Lyrics.lyrics[CurrentLineIndex];
        const NextLine = Lyrics.lyrics[CurrentLineIndex + 1];

        // Check for instrumental

        const LineEnd = CurrentLine.time + CurrentLine.duration;
        const IsInstrumental = NextLine && (NextLine.time - LineEnd > 10_000) && (CurrentTime > LineEnd);

        if (IsInstrumental) {

            return (

                <div className="min-h-[200px] flex flex-col items-center justify-center animate-pulse">
                    
                    <Music size={64} className="text-zinc-500" />
                
                </div>

            );

        }

        const HasSyllables = CurrentLine.syllabus && CurrentLine.syllabus.length > 0;

        return (

            <div key={CurrentLineIndex} className="lyric-line-active text-center max-w-4xl mx-auto px-4">
                
                <div className="text-3xl md:text-4xl font-semibold leading-relaxed tracking-wide">
                    
                    {HasSyllables ? (

                        <div className="flex flex-wrap justify-center gap-x-3 gap-y-1">
                            
                            {(() => {

                                const Words: any[][] = [];
                                let CurrentWord: any[] = [];

                                CurrentLine.syllabus!.forEach((Syllable) => {

                                    CurrentWord.push(Syllable);

                                    if (Syllable.text.endsWith(' ')) {

                                        Words.push(CurrentWord);
                                        CurrentWord = [];

                                    }

                                });

                                if (CurrentWord.length > 0) Words.push(CurrentWord);

                                return Words.map((Word, WordIndex) => (

                                    <span key={WordIndex} className="whitespace-nowrap inline-block">
                                        
                                        {Word.map((Syllable, SyllableIndex) => {

                                            const IsActive = CurrentTime >= Syllable.time;

                                            return (

                                                <span key={SyllableIndex} className={`transition-colors ease-linear ${IsActive ? 'text-white' : 'text-zinc-600'}`} style={{ transitionDuration: `${IsActive && Syllable.duration > 200 ? Syllable.duration : 200}ms` }} >
                                                    
                                                    {Syllable.text}

                                                </span>

                                            );

                                        })}

                                    </span>

                                ));

                            })()}

                        </div>

                    ) : (

                        <span className="text-white transition-colors duration-500">
                            
                            {CurrentLine.text}
                            
                        </span>
                        
                    )}

                </div>

            </div>

        );

    };

    const SendOperation = (OperationType: Operation, Params: { [key: string]: any } = {}) => {

        if (!Socket || Socket.readyState != WebSocket.OPEN) return;
        
        const Message: any = { Operation: OperationType };

        Object.keys(Params).forEach((Key) => {

            Message[Key] = Params[Key];

        });
        
        Socket.send(JSON.stringify(Message));

    };

    const HandlePlayPause = () => {

        if (PlayerStateValue == PlayerState.Playing) {

            SendOperation(Operation.Pause);

        } else if (PlayerStateValue == PlayerState.Paused) {

            SendOperation(Operation.Resume);

        }

    };

    const HandlePrevious = () => {

        SendOperation(Operation.Last);

    };

    const HandleNext = () => {

        SendOperation(Operation.Next);

    };

    const HandleSeek = (Value: number) => {

        const Offset = (Value - CurrentTime) / 1000; // convert ms to seconds for offset

        SendOperation(Operation.Seek, { Offset });
        SetCurrentTime(Value);

    };

    const FormatTime = (Seconds: number): string => {

        const Mins = Math.floor(Seconds / 60);
        const Secs = Math.floor(Seconds % 60);

        return `${Mins}:${Secs.toString().padStart(2, '0')}`;

    };

    if (!CurrentSong) {

        return (

            <div className="min-h-screen bg-zinc-950 flex items-center justify-center">

                <div className="text-zinc-500 text-lg">No Songs are currently playing.</div>

            </div>

        );

    }

    return (

        <div className="min-h-screen relative text-white flex items-center justify-center p-8">
            
            {/* Blurred background */}

            <div className="absolute inset-0 overflow-hidden">

                <div className="absolute inset-0 bg-cover bg-center blur-3xl scale-110 opacity-40" style={{ backgroundImage: `url(${BackgroundImage})` }} />
                <div className="absolute inset-0 bg-zinc-950/50" />
                
            </div>

            <div className="w-full max-w-2xl relative z-10">

                {/* Content Area */}

                <div className="mb-8">

                    {ActiveView == 'Details' && (<>
                        
                        {/* Cover Art */}

                        <div className="relative aspect-square w-full max-w-md mx-auto mb-8 rounded-lg overflow-hidden shadow-2xl">

                            <img src={NormalizeCoverURL(CurrentSong.cover)} alt={CurrentSong.title} referrerPolicy="no-referrer" className="w-full h-full object-cover" />

                        </div>

                        {/* Song Info */}

                        <div className="text-center">

                            <h1 className="text-3xl font-bold mb-2 truncate">{CurrentSong.title}</h1>
                            <p className="text-xl text-zinc-400 truncate"> {CurrentSong.artists.join(', ')} </p>

                        </div>

                    </>)}

                    {/* Lyrics View */}

                    {ActiveView == 'Lyrics' && (

                        <div className="min-h-[200px] flex items-center justify-center">

                            {RenderLyrics()}
                            
                        </div>

                    )}

                    {/* Queue View */}

                    {ActiveView == 'Queue' && (

                        <div className="min-h-[200px] max-h-[500px] overflow-y-scroll">

                            {RenderQueue()}
                            
                        </div>

                    )}

                </div>

                {/* Progress Bar */}

                <div className="mb-8">

                    <div className="relative w-full h-1 bg-zinc-700 rounded-full cursor-pointer overflow-hidden"
                        
                        onClick={(E) => {
                            
                            const Rect = E.currentTarget.getBoundingClientRect();
                            const X = E.clientX - Rect.left;
                            const Percentage = X / Rect.width;
                            const NewTime = Math.max(0, Math.min(CurrentSong.duration.seconds * 1000, Percentage * CurrentSong.duration.seconds * 1000));
                            
                            HandleSeek(Math.floor(NewTime));

                        }}

                        onMouseDown={(MouseEvent) => {

                            const HandleMouseMove = (MoveEvent: MouseEvent) => {

                                const Rect = MouseEvent.currentTarget.getBoundingClientRect();
                                const X = MoveEvent.clientX - Rect.left;
                                const Percentage = Math.max(0, Math.min(1, X / Rect.width));
                                const NewTime = Percentage * CurrentSong.duration.seconds * 1000;
                                
                                SetCurrentTime(Math.floor(NewTime));

                            };

                            const HandleMouseUp = (UpEvent: MouseEvent) => {

                                const Rect = MouseEvent.currentTarget.getBoundingClientRect();
                                const X = UpEvent.clientX - Rect.left;
                                const Percentage = Math.max(0, Math.min(1, X / Rect.width));
                                const NewTime = Percentage * CurrentSong.duration.seconds * 1000;

                                HandleSeek(Math.floor(NewTime));

                                document.removeEventListener('mousemove', HandleMouseMove);
                                document.removeEventListener('mouseup', HandleMouseUp);

                            };

                            document.addEventListener('mousemove', HandleMouseMove);
                            document.addEventListener('mouseup', HandleMouseUp);

                        }}

                        onTouchStart={(E) => {

                            const HandleTouchMove = (MoveEvent: TouchEvent) => {

                                const Rect = E.currentTarget.getBoundingClientRect();
                                const X = MoveEvent.touches[0].clientX - Rect.left;
                                const Percentage = Math.max(0, Math.min(1, X / Rect.width));
                                const NewTime = Percentage * CurrentSong.duration.seconds * 1000;

                                SetCurrentTime(Math.floor(NewTime));

                            };

                            const HandleTouchEnd = (EndEvent: TouchEvent) => {

                                const Rect = E.currentTarget.getBoundingClientRect();
                                const X = EndEvent.changedTouches[0].clientX - Rect.left;
                                const Percentage = Math.max(0, Math.min(1, X / Rect.width));
                                const NewTime = Percentage * CurrentSong.duration.seconds * 1000;

                                HandleSeek(Math.floor(NewTime));

                                document.removeEventListener('touchmove', HandleTouchMove);
                                document.removeEventListener('touchend', HandleTouchEnd);

                            };

                            document.addEventListener('touchmove', HandleTouchMove);
                            document.addEventListener('touchend', HandleTouchEnd);

                        }}

                    >
                        <div className="absolute top-0 left-0 h-full bg-white rounded-full transition-all duration-100" style={{ width: `${(CurrentTime / (CurrentSong.duration.seconds * 1000)) * 100}%` }} />
                    
                    </div>

                    <div className="flex justify-between text-sm text-zinc-500 mt-2">

                        <span>{FormatTime(CurrentTime / 1000)}</span>
                        <span>{CurrentSong.duration.formatted}</span>

                    </div>

                </div>

                {/* Controls */}

                <div className="flex items-center justify-center gap-8 mb-12">

                    <button onClick={HandlePrevious} className="text-white hover:text-zinc-400 transition-colors" aria-label="Previous" >
                        <SkipBack size={40} fill="currentColor"/>
                    </button>
                    
                    <button onClick={HandlePlayPause} className="w-20 h-20 rounded-full bg-white text-zinc-950 hover:bg-zinc-200 transition-colors flex items-center justify-center" aria-label={PlayerStateValue === PlayerState.Playing ? 'Pause' : 'Play'} >
                        {PlayerStateValue == PlayerState.Playing ? (<Pause size={32} fill="currentColor" /> ) : (<Play size={32} fill="currentColor" className="ml-1" /> )}
                    </button>
                    
                    <button onClick={HandleNext} className="text-white hover:text-zinc-400 transition-colors" aria-label="Next" >
                        <SkipForward size={40} fill="currentColor"/>
                    </button>

                </div>

                {/* Bottom Buttons */}

                <div className="flex justify-center gap-4">

                    <button onClick={() => SetActiveView('Details')} className={`px-6 py-2 rounded-md border transition-colors ${ActiveView == 'Details' ? 'bg-white text-zinc-950 border-white' : 'bg-transparent text-white border-zinc-600 hover:border-white' }`} >
                        Details
                    </button>

                    <button onClick={() => SetActiveView('Lyrics')} className={`px-6 py-2 rounded-md border transition-colors ${ActiveView == 'Lyrics' ? 'bg-white text-zinc-950 border-white' : 'bg-transparent text-white border-zinc-600 hover:border-white' }`} >
                        Lyrics
                    </button>

                    <button onClick={() => SetActiveView('Queue')} className={`px-6 py-2 rounded-md border transition-colors ${ActiveView == 'Queue' ? 'bg-white text-zinc-950 border-white' : 'bg-transparent text-white border-zinc-600 hover:border-white' }`} >
                        Queue
                    </button>

                </div>

            </div>

            {ActiveContextMenu && (() => {

                const IsPrevious = ActiveContextMenu.type == 'Previous';

                return (
                    
                    <div className="fixed w-48 bg-zinc-600/35 backdrop-blur-md border border-white/10 rounded-xl shadow-xl z-50 overflow-hidden context-menu-content" style={{ top: ActiveContextMenu.y + 4, left: ActiveContextMenu.x - 192 }} >
                        
                        <div className="p-1">

                            <button className={`w-full flex items-center gap-2 px-3 py-2 text-sm text-left hover:bg-white/10 rounded-lg ${IsPrevious ? 'text-zinc-400 cursor-not-allowed' : 'transition-colors'}`}>
                                
                                <CornerDownRight size={14} />
                                Jump To

                            </button>
                            
                            {!IsPrevious && (

                                <button className="w-full flex items-center gap-2 px-3 py-2 text-sm text-left hover:bg-white/10 rounded-lg text-red-400 hover:text-red-300 transition-colors">
                                   
                                    <Trash2 size={14} />
                                    Remove

                                </button>

                            )}

                        </div>

                    </div>

                );

            })()}

        </div>

    );

}

export default App;