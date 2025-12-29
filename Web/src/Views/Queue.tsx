import { useState } from 'react';
import { MoreHorizontal, ChevronDown, ChevronUp } from 'lucide-react';

import { Song } from '../Types';

interface QueueProps {

    Current: Song | null;

    PreviousSongs: Song[];
    UpcomingSongs: Song[];

    ActiveContextMenu: { type: 'Previous' | 'Upcoming', index: number, x: number, y: number } | null;
    SetActiveContextMenu: (Menu: { type: 'Previous' | 'Upcoming', index: number, x: number, y: number } | null) => void;

}

function Queue({ Current, PreviousSongs, UpcomingSongs, ActiveContextMenu, SetActiveContextMenu }: QueueProps) {

    const [ShowPrevious, SetShowPrevious] = useState(false);

    const NormalizeCoverURL = (URL: string): string => {

        return URL.replace(/=w\d+-h\d+(-l\d+)?(-rj)?/g, '=w512-h512-l90-rj');

    };

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
                
                <div className="flex-1 min-w-0 -ml-1">

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
                {Current && RenderSong(Current, 'Big')}
            
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

}

export default Queue;
