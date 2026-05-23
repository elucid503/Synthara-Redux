import React, { useEffect, useState } from 'react';
import { Music, Headphones, Guitar, Mic, Disc3, Star, Sparkles, Zap, Flame, Gem, Crown, Heart, Smile, Moon, Sun, Leaf, Flower, Feather, Eye, Coffee, Anchor, Wind, Laugh, X } from 'lucide-react';

import { WebIdentifier } from '../Types';
import { DefaultIdentifier } from '../Utils/Misc';

export const IconMap: Record<string, React.ComponentType<{ size?: number; className?: string }>> = { Music, Headphones, Guitar, Mic, Disc3, Star, Sparkles, Zap, Flame, Gem, Crown, Heart, Smile, Moon, Sun, Leaf, Flower, Feather, Eye, Coffee, Anchor, Wind, Laugh, };

interface IdentifyProps {

    InitialValue: WebIdentifier | null;

    Required: boolean;

    OnClose: () => void;
    OnSubmit: (Identifier: WebIdentifier) => void;

}

function Identify({ InitialValue, Required, OnClose, OnSubmit }: IdentifyProps) {

    const [Name, SetName] = useState(InitialValue?.Name || '');
    const [Icon, SetIcon] = useState(InitialValue?.Icon || DefaultIdentifier.Icon);

    useEffect(() => {

        SetName(InitialValue?.Name || '');
        SetIcon(InitialValue?.Icon || DefaultIdentifier.Icon);

    }, [InitialValue]);

    const TrimmedName = Name.trim();
    const SelectedIcon = IconMap[Icon] || IconMap[DefaultIdentifier.Icon];

    const Submit = (Event: React.FormEvent) => {

        Event.preventDefault();
        if (!TrimmedName) return;

        OnSubmit({ Name: TrimmedName, Icon });

    };

    return (

        <div className="fixed inset-0 z-[70] flex items-center justify-center bg-zinc-950/80 px-4 backdrop-blur-md">

            <form onSubmit={Submit} className="w-full max-w-md rounded-xl border border-white/10 bg-zinc-900/95 p-5 shadow-2xl">

                <div className="mb-4 flex items-center justify-between gap-4">

                    <div className="flex min-w-0 items-center gap-3">

                        <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-white text-zinc-950">

                            <SelectedIcon size={20} />

                        </div>

                        <div className="min-w-0">

                            <div className="truncate text-lg font-semibold text-white">Identification</div>
                            <div className="truncate text-sm text-zinc-400">Shown on web queue changes</div>

                        </div>

                    </div>

                    {!Required && (

                        <button type="button" onClick={OnClose} className="flex h-9 w-9 items-center justify-center rounded-lg text-zinc-400 transition-colors hover:bg-white/10 hover:text-white" aria-label="Close identification modal">

                          <X size={18} />

                        </button>

                    )}

                </div>

                <div className="mb-5 text-sm text-zinc-300">

                    Identify yourself so people in the call can see who made web edits.

                </div>

                <label className="mb-2 block text-sm text-zinc-300" htmlFor="identify-name">Display Name</label>
                <input id="identify-name" value={Name} onChange={(Event) => SetName(Event.target.value)} className="mb-5 w-full rounded-lg border border-white/10 bg-zinc-800 px-3 py-2 text-white outline-none transition-colors placeholder:text-zinc-500 focus:border-white/40" placeholder="Enter your display name..." maxLength={40} autoFocus />

                <div className="mb-2 text-sm text-zinc-300">Icon</div>
                <div className="mb-5 grid grid-cols-8 gap-2">

                    {Object.entries(IconMap).map(([Key, IconComponent]) => (

                        <button key={Key} type="button" onClick={() => SetIcon(Key)} title={Key} className={`flex aspect-square items-center justify-center rounded-lg border transition-colors ${Icon == Key ? 'border-white bg-white text-zinc-950' : 'border-white/10 bg-zinc-800 text-zinc-300 hover:border-white/40 hover:text-white'}`} aria-label={`Use ${Key} icon`}>

                          <IconComponent size={18} />

                        </button>

                    ))}

                </div>

                <button type="submit" disabled={!TrimmedName} className="w-full rounded-lg bg-white px-4 py-2.5 text-sm font-semibold text-zinc-950 transition-colors hover:bg-zinc-200 disabled:cursor-not-allowed disabled:bg-zinc-700 disabled:text-zinc-400">

                    Save

                </button>

            </form>

        </div>

    );

}

export default Identify;
