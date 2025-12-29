import { useEffect, useState } from 'react';
import { Play, Pause, SkipBack, SkipForward, Trash2, CornerDownRight, RefreshCw } from 'lucide-react';

import { Song, PlayerState, WSEvents, WSMessage, Operation, LyricsResponse } from './Types';
import { NormalizeCoverURL, FormatTime, SendOperation, FetchLyrics } from './Utils/Misc';
import { HandleProgressBarClick, HandleProgressBarMouseDown, HandleProgressBarTouchStart } from './Utils/Inputs';

import DetailsView from './Views/Details';
import LyricsView from './Views/Lyrics';
import QueueView from './Views/Queue';

function App() {

    const [Socket, SetSocket] = useState<WebSocket | null>(null);
   
    const [CurrentSong, SetCurrentSong] = useState<Song | null>(null);
    const [PreviousSongs, SetPreviousSongs] = useState<Song[]>([]);
    const [UpcomingSongs, SetUpcomingSongs] = useState<Song[]>([]);

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
    
    const FetchLyricsAndSetState = async (Song: Song) => {

        SetLyricsLoading(true);
        SetLyricsError(false);

        const { data, error } = await FetchLyrics(Song);
        
        SetLyrics(data);
        SetLyricsError(error);
        SetLyricsLoading(false);

    };

    // WebSocket connection

    useEffect(() => {

        const Params = window.location.href.split('/');
        const QueueID = Params[Params.length - 1].split('?')[0]; // ignores the query params

        const WS = new WebSocket(`${import.meta.env.VITE_SERVER_URL}/API/Queue?ID=${QueueID}`);

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
                    
                    const InitialProgress = Message.Data.Progress;
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
                    
                    SetCurrentTime(Math.max(0, (Message.Data.Progress - 100))); // slight buffer to account for latency

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
            
            ActiveView == 'Lyrics' && FetchLyricsAndSetState(CurrentSong);

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

            FetchLyricsAndSetState(CurrentSong);

        }
        
    }, [ActiveView, CurrentSong]);

    const HandlePlayPause = () => {

        if (PlayerStateValue == PlayerState.Playing) {

            SendOperation(Socket, Operation.Pause);

        } else if (PlayerStateValue == PlayerState.Paused) {

            SendOperation(Socket, Operation.Resume);

        }

    };

    const HandlePrevious = () => {

        SendOperation(Socket, Operation.Last);

    };

    const HandleNext = () => {

        SendOperation(Socket, Operation.Next);

    };

    const HandleSeek = (Value: number) => {

        const Offset = (Value - CurrentTime) / 1000; // convert ms to seconds for offset

        SendOperation(Socket, Operation.Seek, { Offset });
        SetCurrentTime(Value);

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

                    {ActiveView == 'Details' && (<DetailsView CurrentSong={CurrentSong} />)}

                    {/* Lyrics View */}

                    {ActiveView == 'Lyrics' && (

                        <div className="min-h-[200px] flex items-center justify-center">

                            <LyricsView Lyrics={Lyrics} LyricsError={LyricsError} CurrentTime={CurrentTime} />
                            
                        </div>

                    )}

                    {/* Queue View */}

                    {ActiveView == 'Queue' && (

                        <div className="min-h-[200px] max-h-[500px] overflow-y-scroll">

                            <QueueView Current={CurrentSong} PreviousSongs={PreviousSongs} UpcomingSongs={UpcomingSongs} ActiveContextMenu={ActiveContextMenu} SetActiveContextMenu={SetActiveContextMenu} />
                            
                        </div>

                    )}

                </div>

                {/* Progress Bar */}

                <div className="mb-8">

                    {/* Bar Track */}

                    <div className="relative w-full h-1 bg-zinc-700 rounded-full cursor-pointer overflow-hidden"
                        
                        onClick={(E) => HandleProgressBarClick(E, CurrentSong.duration.seconds, HandleSeek)}

                        onMouseDown={(E) => HandleProgressBarMouseDown(E, CurrentSong.duration.seconds, SetCurrentTime, HandleSeek)}

                        onTouchStart={(E) => HandleProgressBarTouchStart(E, CurrentSong.duration.seconds, SetCurrentTime, HandleSeek)}

                    >
                    
                    {/* Bar Fill */}
                    
                    <div className="absolute top-0 left-0 h-full bg-white rounded-full transition-all duration-100" style={{ width: `${(CurrentTime / (CurrentSong.duration.seconds * 1000)) * 100}%` }}/></div>

                    {/* Time Labels */}
                    
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

                            {IsPrevious && (

                                <button className="w-full flex items-center gap-2 px-3 py-2 text-sm text-left hover:bg-white/10 rounded-lg transition-colors" >
                                   
                                    <RefreshCw size={14} />
                                    Replay

                                </button>

                            )}

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