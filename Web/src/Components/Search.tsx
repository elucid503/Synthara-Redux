import { useState, useRef } from 'react';
import { Search, Music } from 'lucide-react';

import { SuggestionItem } from '../Types';

import { HttpURL } from '../Utils/Misc';

interface SearchBarProps {

    GuildID: string;

    OnSearch: (Query: string) => void;
    OnEnqueue: (TidalID: number) => void;

}

function SearchBar({ GuildID, OnSearch, OnEnqueue }: SearchBarProps) {

    const [Query, SetQuery] = useState('');
    const [Suggestions, SetSuggestions] = useState<SuggestionItem[]>([]);
    const [ShowDropdown, SetShowDropdown] = useState(false);

    const DebounceRef = useRef<any>(null);

    const HandleChange = (E: React.ChangeEvent<HTMLInputElement>) => {

        const Val = E.target.value;
        SetQuery(Val);

        clearTimeout(DebounceRef.current);

        if (Val.trim().length < 2) {

            SetSuggestions([]);
            SetShowDropdown(false);

            return;

        }

        DebounceRef.current = setTimeout(async () => {

            try {

                const Res = await fetch(HttpURL(`/API/Suggestions?ID=${GuildID}&q=${encodeURIComponent(Val)}`));
                const Items: SuggestionItem[] = Res.ok ? await Res.json() : [];

                SetSuggestions(Items);
                SetShowDropdown(Items.length > 0);

            } catch {

                SetSuggestions([]);

            }

        }, 300);

    };

    const DoSearch = (Q: string) => {

        if (!Q.trim()) return;

        SetShowDropdown(false);

        SetQuery(Q);
        OnSearch(Q);

    };

    return (

        <div className="relative">

            <div className="flex items-center gap-2 px-4 py-2.5 bg-zinc-600/35 backdrop-blur-md border border-white/10 rounded-xl">

                <Search size={15} className="text-zinc-400 shrink-0" />

                <input value={Query} className="bg-transparent text-white text-sm outline-none w-full placeholder:text-zinc-500"

                    onChange={HandleChange}
                    onKeyDown={(E) => E.key === 'Enter' && DoSearch(Query)}
                    onFocus={() => Suggestions.length > 0 && SetShowDropdown(true)}
                    onBlur={() => setTimeout(() => SetShowDropdown(false), 150)}

                    placeholder="Search for a song..."

                />

            </div>

            {ShowDropdown && Suggestions.length > 0 && (

                <div className="absolute top-full mt-1 left-0 right-0 bg-zinc-600/35 backdrop-blur-md border border-white/10 rounded-xl overflow-hidden z-50">

                    {Suggestions.slice(0, 8).map((S, I) =>

                        S.type === 'Track' ? (

                            <button key={`track-${S.tidal_id}`} onMouseDown={() => { SetShowDropdown(false); OnEnqueue(S.tidal_id!); }} className="w-full text-left px-4 py-2.5 text-sm hover:bg-white/10 transition-colors flex items-center gap-2" >

                              <Music size={12} className="text-zinc-500 shrink-0" />

                                <span className="text-white truncate">{S.title}</span>
                                <span className="text-zinc-400 text-xs shrink-0">{S.subtitle}</span>

                            </button>

                        ) : (

                            <button key={`text-${I}`} onMouseDown={() => DoSearch(S.text!)} className="w-full text-left px-4 py-2.5 text-sm hover:bg-white/10 transition-colors flex items-center gap-2" >

                                <Search size={12} className="text-zinc-500 shrink-0" />
                                <span className="text-white truncate">{S.text}</span>

                            </button>

                        )

                    )}

                </div>

            )}

        </div>

    );

}

export default SearchBar;
