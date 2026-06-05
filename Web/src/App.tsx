import { useEffect, useState, useRef } from 'react';
import { Play, Pause, SkipBack, SkipForward, Trash2, CornerDownRight, RefreshCw } from 'lucide-react';

import { Song, PlayerState, WSEvents, WSMessage, Operation, LyricsResponse, InitialStateData, AuthState } from './Types';
import { NormalizeCoverURL, FormatTime, SendOperation, FetchLyrics, FormatWS, FetchAPI } from './Utils/Misc';

import DetailsView from './Views/Details';
import LyricsView from './Views/Lyrics';
import QueueView from './Views/Queue';
import SearchBar from './Components/Search';

function App() {

    const [Socket, SetSocket] = useState<WebSocket | null>(null);

    const [CurrentSong, SetCurrentSong] = useState<Song | null>(null);
    const [PreviousSongs, SetPreviousSongs] = useState<Song[]>([]);
    const [UpcomingSongs, SetUpcomingSongs] = useState<Song[]>([]);

    const [ActiveContextMenu, SetActiveContextMenu] = useState<{ type: 'Previous' | 'Upcoming', index: number, x: number, y: number } | null>(null);

    const [QueueEnded, SetQueueEnded] = useState(false);
    const [HasEverConnected, SetHasEverConnected] = useState(false);

    const [Toast, SetToast] = useState<string | null>(null);
    const ToastTimeoutRef = useRef<any>(null);
    const [Auth, SetAuth] = useState<AuthState>({ OAuthEnabled: false, Authenticated: false });
    const [ControlsLocked, SetControlsLocked] = useState(false);
    const [GuildLocked, SetGuildLocked] = useState(false);

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
    const [CurrentTime, SetCurrentTime] = useState(0); // in milliseconds

    const GuildID = window.location.href.split('/').pop()?.split('?')[0] ?? '';

    useEffect(() => {

        FetchAPI('/API/Auth/Me')
            .then((Res) => Res.json())
            .then((Data: AuthState & { Username?: string }) => {

                SetAuth({

                    OAuthEnabled: !!Data.OAuthEnabled,
                    Authenticated: !!Data.Authenticated,
                    Username: Data.Username,

                });

                SetControlsLocked(!Data.OAuthEnabled || !Data.Authenticated);

            })
            .catch(() => {});

    }, []);

    const [ActiveView, SetActiveView] = useState<'Details' | 'Queue' | 'Lyrics'>(() => {

        const Params = new URLSearchParams(window.location.search);
        const View = Params.get('View');

        if (View == 'Lyrics') return 'Lyrics';
        if (View == 'Queue') return 'Queue';

        return 'Details';

    });

    const [BackgroundImage, SetBackgroundImage] = useState<string>('');

    const CurrentSongIdRef = useRef<number | null>(null);
    const ActiveViewRef = useRef<'Details' | 'Queue' | 'Lyrics'>(ActiveView);
    const UpcomingSongsLengthRef = useRef<number>(0);

    const [Lyrics, SetLyrics] = useState<LyricsResponse | null>(null);
    const [LyricsError, SetLyricsError] = useState(false);

    const CurrentTimeBuffer = 2000; // 2000ms buffer for progress updates

    const FetchLyricsAndSetState = async (Song: Song) => {

        SetLyricsError(false);

        const { data, error } = await FetchLyrics(Song);

        SetLyrics(data);
        SetLyricsError(error);

    };

    const ShowToast = (Message: string) => {

        if (ToastTimeoutRef.current) {

            clearTimeout(ToastTimeoutRef.current);

        }

        SetToast(Message);
        ToastTimeoutRef.current = setTimeout(() => SetToast(null), 3000);

    };

    // WebSocket connection with reconnection

    useEffect(() => {

        const Params = window.location.href.split('/');
        const QueueID = Params[Params.length - 1].split('?')[0];

        let WS: WebSocket | null = null;

        let ReconnectTimeout: any = null;
        let ShouldReconnect = true;
        let ReconnectAttempts = 0;

        const Connect = () => {

            WS = new WebSocket(FormatWS(`/API/Queue?ID=${QueueID}`));

            WS.onopen = () => {

                console.log('WebSocket connected');
                ReconnectAttempts = 0;
                SetQueueEnded(false);
                SetHasEverConnected(true);

            };

            WS.onmessage = (Event) => {

                const Message: WSMessage<any> = JSON.parse(Event.data);

                switch (Message.Event) {

                    case WSEvents.Event_Initial:

                        const Initial = Message.Data as InitialStateData;

                        SetCurrentSong(Initial.Current);
                        SetPreviousSongs(Initial.Previous || []);
                        SetUpcomingSongs(Initial.Upcoming || []);
                        SetPlayerStateValue(Initial.State);

                        const InitialProgress = Initial.Progress;
                        SetCurrentTime(Math.max(0, (InitialProgress - CurrentTimeBuffer)));

                        if (Initial.OAuthEnabled != null) {

                            SetAuth((Prev) => ({

                                OAuthEnabled: !!Initial.OAuthEnabled,
                                Authenticated: !!Initial.Authenticated,
                                Username: Initial.Authenticated ? (Prev.Username || undefined) : undefined,

                            }));

                        }

                        SetGuildLocked(!!Initial.GuildLocked);
                        SetControlsLocked(!!Initial.ControlsLocked);

                    break;

                    case WSEvents.Event_StateChanged:

                        SetPlayerStateValue(Message.Data.State);

                    break;

                    case WSEvents.Event_ProgressUpdate:

                        SetCurrentTime(Math.max(0, (Message.Data.Progress - CurrentTimeBuffer)));

                    break;

                    case WSEvents.Event_QueueUpdated:

                        SetPreviousSongs(Message.Data.Previous || []);
                        SetUpcomingSongs(Message.Data.Upcoming || []);

                        const NewSong = Message.Data.Current as Song | null;
                        const SongChanged = NewSong && NewSong.tidal_id != CurrentSongIdRef.current;

                        if (SongChanged) {

                            SetCurrentSong(NewSong);
                            SetCurrentTime(0);
                            SetLyrics(null);

                            if (ActiveViewRef.current != 'Queue') {

                                ShowToast(`Now Playing ${NewSong.title}`);

                            }

                        } else if (ActiveViewRef.current != 'Queue' && Message.Data.Upcoming.length > UpcomingSongsLengthRef.current) {

                            ShowToast('Queue Updated');

                        }

                        break;

                    case WSEvents.Event_Error:

                        const ErrorData = Message.Data as { Message: string };

                        if (ErrorData.Message) {

                            ShowToast(ErrorData.Message);

                        }

                    break;

                }

            };

            WS.onerror = (Error) => {

                console.error('WebSocket error:', Error);

            };

            WS.onclose = () => {

                console.log('WebSocket disconnected');
                SetSocket(null);

                if (ShouldReconnect) {

                    ReconnectAttempts++;

                    if (ReconnectAttempts > 3) {

                        SetQueueEnded(true);
                        ShouldReconnect = false;

                    } else {

                        ReconnectTimeout = setTimeout(() => Connect(), 2000);

                    }

                }

            };

            SetSocket(WS);

        };

        Connect();

        return () => {

            ShouldReconnect = false;

            clearTimeout(ReconnectTimeout);
            WS?.close();

        };

    }, []);

    // Updates background when song changes

    useEffect(() => {

        if (CurrentSong) {

            CurrentSongIdRef.current = CurrentSong.tidal_id;
            SetBackgroundImage(NormalizeCoverURL(CurrentSong.cover));

        } else {

            CurrentSongIdRef.current = null;

        }

    }, [CurrentSong]);

    useEffect(() => {

        ActiveViewRef.current = ActiveView;

    }, [ActiveView]);

    useEffect(() => {

        UpcomingSongsLengthRef.current = UpcomingSongs.length;

    }, [UpcomingSongs]);

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

    // Fetch lyrics when switching to Lyrics view or when song changes while on Lyrics view

    useEffect(() => {

        if (ActiveView == 'Lyrics' && CurrentSong) {

            FetchLyricsAndSetState(CurrentSong);

        }

    }, [ActiveView, CurrentSong]);

    const ShowLockBanner = !Auth.OAuthEnabled || GuildLocked;
    const LockBannerMessage = !Auth.OAuthEnabled
        ? 'Discord OAuth is not configured. Web controls are unavailable.'
        : 'Web controls are locked. Use /unlock in Discord to re-enable.';

    const HandlePlayPause = () => {

        if (PlayerStateValue == PlayerState.Playing) {

            SendOperation(Socket, Operation.Pause, {}, ControlsLocked);

        } else if (PlayerStateValue == PlayerState.Paused) {

            SendOperation(Socket, Operation.Resume, {}, ControlsLocked);

        }

    };

    const HandlePrevious = () => {

        SendOperation(Socket, Operation.Last, {}, ControlsLocked);

    };

    const HandleNext = () => {

        SendOperation(Socket, Operation.Next, {}, ControlsLocked);

    };

    const HandleJump = (Index: number) => {

        SendOperation(Socket, Operation.Jump, { Index: Index + 1 }, ControlsLocked);
        SetActiveContextMenu(null);

    };

    const HandleRemove = (Index: number) => {

        SendOperation(Socket, Operation.Remove, { Index }, ControlsLocked);
        SetActiveContextMenu(null);

    };

    const HandleMove = (FromIndex: number, ToIndex: number) => {

        SendOperation(Socket, Operation.Move, { FromIndex, ToIndex }, ControlsLocked);

    };

    const HandleReplay = (Index: number) => {

        SendOperation(Socket, Operation.Replay, { Index }, ControlsLocked);
        SetActiveContextMenu(null);

    };

    const HandleEnqueue = (TidalID: number) => {

        SendOperation(Socket, Operation.Enqueue, { TidalID }, ControlsLocked);
        SetActiveView('Queue');

    };

    if (QueueEnded) {

        return (

            <div className="min-h-screen bg-zinc-950 flex items-center justify-center">

                <div className="text-zinc-500 text-lg">This Queue has Ended</div>

            </div>

        );

    }

    if (!CurrentSong) {

        return (

            <div className="min-h-screen bg-zinc-950 flex items-center justify-center">

                <div className="text-zinc-500 text-lg">{HasEverConnected ? 'No Songs are currently playing.' : 'No Queue Found'}</div>

            </div>

        );

    }

    return (

        <div className="min-h-screen relative text-white flex items-center justify-center px-8 pt-24 pb-8">

            {/* Blurred background */}

            <div className="absolute inset-0 overflow-hidden">

                <div className="absolute inset-0 bg-cover bg-center blur-3xl scale-110 opacity-40" style={{ backgroundImage: `url(${BackgroundImage})` }} />
                <div className="absolute inset-0 bg-zinc-950/50" />

            </div>

            <div className="w-full max-w-2xl relative z-10">

                {/* Content Area */}

                <div className="mb-8">

                    {ActiveView == 'Details' && (<DetailsView key={CurrentSong ? CurrentSong.tidal_id.toString() : 'none'} CurrentSong={CurrentSong} />)}

                    {/* Lyrics View */}

                    {ActiveView == 'Lyrics' && (

                        <div className="min-h-[200px] flex items-center justify-center">

                            <LyricsView Lyrics={Lyrics} LyricsError={LyricsError} CurrentTime={CurrentTime} />

                        </div>

                    )}

                    {/* Queue View */}

                    {ActiveView == 'Queue' && (

                        <div className="min-h-[200px] max-h-[500px] overflow-y-auto">

                            <QueueView key={CurrentSong ? CurrentSong.tidal_id.toString() : 'none'} Current={CurrentSong} PreviousSongs={PreviousSongs} UpcomingSongs={UpcomingSongs} ActiveContextMenu={ActiveContextMenu} SetActiveContextMenu={SetActiveContextMenu} OnMove={HandleMove} ControlsLocked={ControlsLocked} />

                        </div>

                    )}

                </div>

                {/* Progress Bar */}

                <div className="mb-8">

                    {/* Bar Track */}

                    <div className="relative w-full h-1 bg-zinc-700 rounded-full overflow-hidden">

                    {/* Bar Fill */}

                    <div className="absolute top-0 left-0 h-full bg-white rounded-full transition-all duration-100" style={{ width: `${(CurrentTime / (CurrentSong.duration.seconds * 1000)) * 100}%` }}/></div>

                    {/* Time Labels */}

                    <div className="flex justify-between text-sm text-zinc-500 mt-2">

                        <span>{FormatTime(CurrentTime / 1000)}</span>
                        <span>{CurrentSong.duration.formatted}</span>

                    </div>

                </div>

                {ShowLockBanner && (

                    <div className="mb-6 rounded-xl border border-amber-500/30 bg-amber-500/10 px-4 py-3 text-center text-sm text-amber-100">

                        {LockBannerMessage}

                    </div>

                )}

                {/* Controls */}

                <div className={`mb-12 flex items-center justify-center gap-8 ${ControlsLocked ? 'pointer-events-none opacity-40' : ''}`}>

                    <button onClick={HandlePrevious} disabled={ControlsLocked} className="text-white transition-colors hover:text-zinc-400 disabled:cursor-not-allowed" aria-label="Previous" >
                        <SkipBack size={40} fill="currentColor"/>
                    </button>

                    <button onClick={HandlePlayPause} disabled={ControlsLocked} className="flex h-20 w-20 items-center justify-center rounded-full bg-white text-zinc-950 transition-colors hover:bg-zinc-200 disabled:cursor-not-allowed" aria-label={PlayerStateValue === PlayerState.Playing ? 'Pause' : 'Play'} >
                        {PlayerStateValue == PlayerState.Playing ? (<Pause size={32} fill="currentColor" /> ) : (<Play size={32} fill="currentColor" className="ml-1" /> )}
                    </button>

                    <button onClick={HandleNext} disabled={ControlsLocked} className="text-white transition-colors hover:text-zinc-400 disabled:cursor-not-allowed" aria-label="Next" >
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

            {/* Fixed top search bar */}

            <div className="fixed top-10 left-0 right-0 z-40 px-6 pt-4">

                <div className="max-w-2xl mx-auto">

                    <SearchBar GuildID={GuildID} OnEnqueue={HandleEnqueue} Auth={Auth} ControlsLocked={ControlsLocked} />

                </div>

            </div>

            {ActiveContextMenu && !ControlsLocked && (() => {

                const IsPrevious = ActiveContextMenu.type == 'Previous';

                return (

                    <div className="fixed w-48 bg-zinc-600/35 backdrop-blur-md border border-white/10 rounded-xl shadow-xl z-50 overflow-hidden context-menu-content" style={{ top: ActiveContextMenu.y + 4, left: ActiveContextMenu.x - 192 }} >

                        <div className="p-1">

                            {IsPrevious && (

                                <button onClick={() => HandleReplay(ActiveContextMenu.index)} className="w-full flex items-center gap-2 px-3 py-2 text-sm text-left hover:bg-white/10 rounded-lg transition-colors" >

                                    <RefreshCw size={14} />
                                    Replay

                                </button>

                            )}

                            <button onClick={() => !IsPrevious && HandleJump(ActiveContextMenu.index)} className={`w-full flex items-center gap-2 px-3 py-2 text-sm text-left hover:bg-white/10 rounded-lg ${IsPrevious ? 'text-zinc-400 cursor-not-allowed' : 'transition-colors'}`}>

                                <CornerDownRight size={14} />
                                Jump To

                            </button>

                            {!IsPrevious && (

                                <button onClick={() => HandleRemove(ActiveContextMenu.index)} className="w-full flex items-center gap-2 px-3 py-2 text-sm text-left hover:bg-white/10 rounded-lg text-red-400 hover:text-red-300 transition-colors">

                                    <Trash2 size={14} />
                                    Remove

                                </button>

                            )}

                        </div>

                    </div>

                );

            })()}

            {Toast && (

                <div className="fixed bottom-8 left-1/2 -translate-x-1/2 px-6 py-3 bg-white text-zinc-950 rounded-lg shadow-lg z-50 animate-fade-in">

                    {Toast}

                </div>

            )}

        </div>

    );

}

export default App;
