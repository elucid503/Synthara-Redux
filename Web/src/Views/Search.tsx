import { Music, Plus } from 'lucide-react';
import { SearchResult } from '../Types';

interface SearchResultsProps {
    Results: SearchResult[];
    OnEnqueue: (TidalID: number) => void;
}

function SearchResults({ Results, OnEnqueue }: SearchResultsProps) {

    if (Results.length === 0) {

        return (

            <div className="flex flex-col items-center justify-center py-16 text-zinc-500">

                <Music size={32} className="mb-3 opacity-40" />
                <span className="text-sm">No results found.</span>

            </div>

        );

    }

    return (

        <div className="flex flex-col gap-1">

            {Results.map((S) => (

                <button key={S.tidal_id} onClick={() => OnEnqueue(S.tidal_id)} className="flex items-center gap-3 px-3 py-2.5 rounded-xl bg-white/5 hover:bg-white/10 transition-colors text-left w-full group" >

                    <div className="flex-1 min-w-0">
                        <div className="text-sm text-white font-medium truncate">{S.title}</div>
                        <div className="text-xs text-zinc-400 truncate">{S.subtitle}</div>
                    </div>

                    <Plus size={16} className="text-zinc-500 group-hover:text-white transition-colors shrink-0" />

                </button>

            ))}

        </div>

    );

}

export default SearchResults;
