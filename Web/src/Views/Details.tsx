import { Song } from '../Types';

interface DetailsProps {

    CurrentSong: Song;

}

function Details({ CurrentSong }: DetailsProps) {

    const NormalizeCoverURL = (URL: string): string => {

        return URL.replace(/=w\d+-h\d+(-l\d+)?(-rj)?/g, '=w512-h512-l90-rj');

    };

    return (

        <div className="flex min-h-0 flex-1 flex-col">

            <div className="relative mx-auto mb-5 aspect-square w-full max-w-[min(58vh,30rem)] overflow-hidden rounded-xl shadow-2xl shadow-black/40">

                <img src={NormalizeCoverURL(CurrentSong.cover)} alt={CurrentSong.title} referrerPolicy="no-referrer" className="h-full w-full object-cover" />

            </div>

            <div className="min-w-0 text-center">

                <h1 className="truncate text-3xl mt-3 font-bold">{CurrentSong.title}</h1>
                <p className="mt-2 truncate text-lg text-zinc-400">{CurrentSong.artists.join(', ')}</p>
                {CurrentSong.album && <p className="mb-1 truncate text-sm text-zinc-500">{CurrentSong.album}</p>}

            </div>

        </div>

    );

}

export default Details;
