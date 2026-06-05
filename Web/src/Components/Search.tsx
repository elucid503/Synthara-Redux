import { useState, useRef } from 'react';
import { Search, Music, LogIn, LogOut } from 'lucide-react';

import { AuthState, SuggestionItem } from '../Types';

import { FetchAPI, FormatURL } from '../Utils/Misc';

interface SearchBarProps {

    GuildID: string;

    OnEnqueue: (TidalID: number) => void;

    Auth: AuthState;
    ControlsLocked: boolean;

}

function SearchBar({ GuildID, OnEnqueue, Auth, ControlsLocked }: SearchBarProps) {

    const [Query, SetQuery] = useState('');
    const [Suggestions, SetSuggestions] = useState<SuggestionItem[]>([]);
    const [ShowDropdown, SetShowDropdown] = useState(false);

    const DebounceRef = useRef<any>(null);

    const FetchSuggestions = (Val: string) => {

        clearTimeout(DebounceRef.current);

        if (ControlsLocked || Val.trim().length < 2) {

            SetSuggestions([]);
            SetShowDropdown(false);

            return;

        }

        DebounceRef.current = setTimeout(async () => {

            try {

                const Res = await FetchAPI(`/API/Suggestions?ID=${GuildID}&q=${encodeURIComponent(Val)}`);
                const Items: SuggestionItem[] = Res.ok ? await Res.json() : [];

                SetSuggestions(Items);
                SetShowDropdown(Items.length > 0);

            } catch {

                SetSuggestions([]);

            }

        }, 300);

    };

    const HandleChange = (E: React.ChangeEvent<HTMLInputElement>) => {

        const Val = E.target.value;
        SetQuery(Val);
        FetchSuggestions(Val);

    };

    const SelectSuggestionText = (Text: string) => {

        SetQuery(Text);
        FetchSuggestions(Text);

    };

    const EnqueueTrack = (TidalID: number) => {

        SetQuery('');
        SetSuggestions([]);
        SetShowDropdown(false);
        OnEnqueue(TidalID);

    };

    const Login = () => {

        const ReturnTo = `${window.location.pathname}${window.location.search}`;
        window.location.href = FormatURL(`/API/Auth/Login?returnTo=${encodeURIComponent(ReturnTo)}`);

    };

    const Logout = async () => {

        await FetchAPI('/API/Auth/Logout', { method: 'POST' });
        window.location.reload();

    };

    const Placeholder = !Auth.OAuthEnabled || ControlsLocked
        ? 'Controls locked'
        : 'Search for a song...';

    return (

        <div className="relative flex gap-2">

            <div className={`flex flex-1 items-center gap-2 rounded-xl border border-white/10 bg-zinc-600/35 px-4 py-2.5 backdrop-blur-md ${ControlsLocked ? 'opacity-50' : ''}`}>

                <Search size={15} className="shrink-0 text-zinc-400" />

                <input value={Query} disabled={ControlsLocked} className="w-full bg-transparent text-sm text-white outline-none placeholder:text-zinc-500 disabled:cursor-not-allowed"

                    onChange={HandleChange}
                    onFocus={() => Suggestions.length > 0 && SetShowDropdown(true)}
                    onBlur={() => setTimeout(() => SetShowDropdown(false), 150)}

                    placeholder={Placeholder}

                />

            </div>

            {Auth.OAuthEnabled && (

                Auth.Authenticated ? (

                    <button onClick={Logout} title={`Signed in as ${Auth.Username}`} className="flex h-[42px] shrink-0 items-center gap-2 rounded-xl border border-white/10 bg-zinc-600/35 px-3 text-sm text-zinc-300 backdrop-blur-md transition-colors hover:border-white/30 hover:text-white" aria-label="Sign out">

                        <LogOut size={16} />
                        <span className="hidden max-w-[8rem] truncate sm:inline">{Auth.Username}</span>

                    </button>

                ) : (

                    <button onClick={Login} className="flex h-[42px] shrink-0 items-center gap-2 rounded-xl border border-[#5865F2]/40 bg-[#5865F2]/20 px-3 text-sm font-medium text-white backdrop-blur-md transition-colors hover:bg-[#5865F2]/35" aria-label="Sign in with Discord">

                        <LogIn size={16} />
                        <span className="hidden sm:inline">Sign in</span>

                    </button>

                )

            )}

            {ShowDropdown && Suggestions.length > 0 && !ControlsLocked && (

                <div className={`absolute left-0 top-full z-50 mt-2 overflow-hidden rounded-xl border border-white/10 bg-zinc-600/35 backdrop-blur-md ${Auth.OAuthEnabled ? 'right-[50px]' : 'right-0'}`}>

                    {Suggestions.slice(0, 8).map((S, I) =>

                        S.type === 'Track' ? (

                            <button key={`track-${S.tidal_id}`} onMouseDown={() => EnqueueTrack(S.tidal_id!)} className="flex w-full items-center gap-2 px-4 py-2.5 text-left text-sm transition-colors hover:bg-white/10" >

                              <Music size={12} className="shrink-0 text-zinc-500" />

                                <span className="truncate text-white">{S.title}</span>
                                <span className="shrink-0 text-xs text-zinc-400">{S.subtitle}</span>

                            </button>

                        ) : (

                            <button key={`text-${I}`} onMouseDown={() => SelectSuggestionText(S.text!)} className="flex w-full items-center gap-2 px-4 py-2.5 text-left text-sm transition-colors hover:bg-white/10" >

                                <Search size={12} className="shrink-0 text-zinc-500" />
                                <span className="truncate text-white">{S.text}</span>

                            </button>

                        )

                    )}

                </div>

            )}

        </div>

    );

}

export default SearchBar;