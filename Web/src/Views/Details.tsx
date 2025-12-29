import { Song } from '../Types';

interface DetailsProps {

    CurrentSong: Song;

}

function Details({ CurrentSong }: DetailsProps) {

    const NormalizeCoverURL = (URL: string): string => {

        return URL.replace(/=w\d+-h\d+(-l\d+)?(-rj)?/g, '=w512-h512-l90-rj');

    };

    return (

        <>
            
            {/* Cover Art */}

            <div className="relative aspect-square w-full max-w-md mx-auto mb-8 rounded-lg overflow-hidden shadow-2xl">

                <img src={NormalizeCoverURL(CurrentSong.cover)} alt={CurrentSong.title} referrerPolicy="no-referrer" className="w-full h-full object-cover" />

            </div>

            {/* Song Info */}

            <div className="text-center">

                <h1 className="text-3xl font-bold mb-2 truncate">{CurrentSong.title}</h1>
                <p className="text-xl text-zinc-400 truncate"> {CurrentSong.artists.join(', ')} </p>

            </div>

        </>

    );

}

export default Details;