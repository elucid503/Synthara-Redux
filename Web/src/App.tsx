import { useEffect, useRef, useState } from 'react';
import { Play, Pause, SkipBack, SkipForward } from 'lucide-react';

import { Song, PlayerState, WSEvents, WSMessage, Operation } from './Types';

function App() {

    const [Socket, SetSocket] = useState<WebSocket | null>(null);
    const [CurrentSong, SetCurrentSong] = useState<Song | null>(null);
    const [PlayerStateValue, SetPlayerStateValue] = useState<PlayerState>(PlayerState.Idle);
    const [CurrentTime, SetCurrentTime] = useState(0); // in seconds
    const [ActiveView, SetActiveView] = useState<'NowPlaying' | 'Queue'>('NowPlaying');
    const [BackgroundImage, SetBackgroundImage] = useState<string>('');
    
    const IntervalRef = useRef<number | null>(null);

    // Normalizes Google cover URLs to ensure 512x512 dimensions

    const NormalizeCoverURL = (URL: string): string => {

        return URL.replace(/=w\d+-h\d+(-l\d+)?(-rj)?/g, '=w512-h512-l90-rj');

    };

    // WebSocket connection

    useEffect(() => {

        const Params = window.location.href.split('/');
        const QueueID = Params[Params.length - 1];

        const WS = new WebSocket(`ws://localhost:3000/API/Queue?ID=${QueueID}`);

        WS.onopen = () => {

            console.log('WebSocket connected');

        };

        WS.onmessage = (Event) => {

            const Message: WSMessage<any> = JSON.parse(Event.data);
            
            switch (Message.Event) {

                case WSEvents.Event_Initial:

                    SetCurrentSong(Message.Data.Current);
                    SetPlayerStateValue(Message.Data.State);
                    SetCurrentTime(Message.Data.Progress); // provides initial progress

                break;
                    
                case WSEvents.Event_StateChanged:

                    SetPlayerStateValue(Message.Data.State);

                break;
                    
                case WSEvents.Event_QueueUpdated:

                    SetCurrentSong(Message.Data.Current);
                    SetCurrentTime(0);

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

    // Updates background when song changes

    useEffect(() => {

        if (CurrentSong) {

            SetBackgroundImage(NormalizeCoverURL(CurrentSong.cover));

        }

    }, [CurrentSong]);

    // Progress tracking

    useEffect(() => {

        if (PlayerStateValue == PlayerState.Playing) {

            IntervalRef.current = window.setInterval(() => {

                SetCurrentTime((Prev) => {

                    if (CurrentSong && Prev >= CurrentSong.duration.seconds) {

                        return CurrentSong.duration.seconds;

                    }

                    return Prev + 1;

                });

            }, 1000);

        } else {

            if (IntervalRef.current) {

                clearInterval(IntervalRef.current);
                IntervalRef.current = null;

            }

        }

        return () => {

            if (IntervalRef.current) {

                clearInterval(IntervalRef.current);

            }

        };

    }, [PlayerStateValue, CurrentSong]);

    const SendOperation = (OperationType: Operation, Params: { [key: string]: any } = {}) => {

        if (!Socket || Socket.readyState != WebSocket.OPEN) return;
        
        const Message: any = { Operation: OperationType };

        Object.keys(Params).forEach((Key) => {

            Message[Key] = Params[Key];

        });
        
        Socket.send(JSON.stringify(Message));

    };

    const HandlePlayPause = () => {

        if (PlayerStateValue === PlayerState.Playing) {

            SendOperation(Operation.Pause);

        } else if (PlayerStateValue === PlayerState.Paused) {

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

        const Offset = Value - CurrentTime; // will be positive for seeking forward, negative for backward

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

                {/* Cover Art */}

                <div className="relative aspect-square w-full max-w-md mx-auto mb-8 rounded-lg overflow-hidden shadow-2xl">

                    <img src={NormalizeCoverURL(CurrentSong.cover)} alt={CurrentSong.title} referrerPolicy="no-referrer" className="w-full h-full object-cover" />

                </div>

                {/* Song Info */}

                <div className="text-center mb-8">

                    <h1 className="text-3xl font-bold mb-2 truncate">{CurrentSong.title}</h1>
                    <p className="text-xl text-zinc-400 truncate"> {CurrentSong.artists.join(', ')} </p>

                </div>

                {/* Progress Bar */}

                <div className="mb-8">

                    <div className="relative w-full h-1 bg-zinc-700 rounded-full cursor-pointer overflow-hidden"
                        
                        onClick={(E) => {
                            
                            const Rect = E.currentTarget.getBoundingClientRect();
                            const X = E.clientX - Rect.left;
                            const Percentage = X / Rect.width;
                            const NewTime = Math.max(0, Math.min(CurrentSong.duration.seconds, Percentage * CurrentSong.duration.seconds));
                            
                            HandleSeek(Math.floor(NewTime));

                        }}

                        onMouseDown={(MouseEvent) => {

                            const HandleMouseMove = (MoveEvent: MouseEvent) => {

                                const Rect = MouseEvent.currentTarget.getBoundingClientRect();
                                const X = MoveEvent.clientX - Rect.left;
                                const Percentage = Math.max(0, Math.min(1, X / Rect.width));
                                const NewTime = Percentage * CurrentSong.duration.seconds;
                                
                                SetCurrentTime(Math.floor(NewTime));

                            };

                            const HandleMouseUp = (UpEvent: MouseEvent) => {

                                const Rect = MouseEvent.currentTarget.getBoundingClientRect();
                                const X = UpEvent.clientX - Rect.left;
                                const Percentage = Math.max(0, Math.min(1, X / Rect.width));
                                const NewTime = Percentage * CurrentSong.duration.seconds;

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
                                const NewTime = Percentage * CurrentSong.duration.seconds;

                                SetCurrentTime(Math.floor(NewTime));

                            };

                            const HandleTouchEnd = (EndEvent: TouchEvent) => {

                                const Rect = E.currentTarget.getBoundingClientRect();
                                const X = EndEvent.changedTouches[0].clientX - Rect.left;
                                const Percentage = Math.max(0, Math.min(1, X / Rect.width));
                                const NewTime = Percentage * CurrentSong.duration.seconds;

                                HandleSeek(Math.floor(NewTime));

                                document.removeEventListener('touchmove', HandleTouchMove);
                                document.removeEventListener('touchend', HandleTouchEnd);

                            };

                            document.addEventListener('touchmove', HandleTouchMove);
                            document.addEventListener('touchend', HandleTouchEnd);

                        }}

                    >
                        <div className="absolute top-0 left-0 h-full bg-white rounded-full transition-all duration-100" style={{ width: `${(CurrentTime / CurrentSong.duration.seconds) * 100}%` }} />
                    
                    </div>

                    <div className="flex justify-between text-sm text-zinc-500 mt-2">

                        <span>{FormatTime(CurrentTime)}</span>
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

                    <button onClick={() => SetActiveView('NowPlaying')} className={`px-6 py-2 rounded-md border transition-colors ${ActiveView == 'NowPlaying' ? 'bg-white text-zinc-950 border-white' : 'bg-transparent text-white border-zinc-600 hover:border-white' }`} >
                        Now Playing
                    </button>

                    <button onClick={() => SetActiveView('Queue')} className={`px-6 py-2 rounded-md border transition-colors ${ActiveView == 'Queue' ? 'bg-white text-zinc-950 border-white' : 'bg-transparent text-white border-zinc-600 hover:border-white' }`} >
                        Queue
                    </button>

                </div>

            </div>

        </div>

    );

}

export default App;