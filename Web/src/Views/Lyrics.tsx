import { useMemo } from 'react';
import { Music } from 'lucide-react';

import { LyricsResponse } from '../Types';

interface LyricsProps {

    Lyrics: LyricsResponse | null;

    LyricsError: boolean;

    CurrentTime: number;

}

function Lyrics({ Lyrics, LyricsError, CurrentTime }: LyricsProps) {

    const CurrentLineIndex = useMemo(() => {

        if (!Lyrics || !Lyrics.lyrics) return -1;

        // Finds the last line that has started. This is ideal, since we only display one line at a time

        for (let i = Lyrics.lyrics.length - 1; i >= 0; i--) {

            if (CurrentTime >= Lyrics.lyrics[i].time) {

                return i;

            }

        }

        return -1;

    }, [Lyrics, CurrentTime]);

    if (LyricsError) {

        return (

            <div className="min-h-[200px] flex items-center justify-center">

                <div className="text-zinc-500">No Lyrics available.</div>

            </div>

        );

    }

    if (!Lyrics || CurrentLineIndex == -1) {

        if (Lyrics && Lyrics.lyrics.length > 0 && CurrentTime < Lyrics.lyrics[0].time) {

            return (

                <div className="min-h-[200px] flex flex-col items-center justify-center animate-pulse">
                    
                    <Music size={64} className="text-zinc-500" />
                
                </div>

            );

        }

        return (

            <div className="min-h-[200px] flex items-center justify-center">

                <div className="text-zinc-500 animate-pulse">Loading Lyrics...</div>

            </div>

        );

    }

    const CurrentLine = Lyrics.lyrics[CurrentLineIndex];
    const NextLine = Lyrics.lyrics[CurrentLineIndex + 1];

    // Check for instrumental

    const LineEnd = CurrentLine.time + CurrentLine.duration;
    const IsInstrumental = NextLine && (NextLine.time - LineEnd > 10_000) && (CurrentTime > LineEnd);

    if (IsInstrumental) {

        return (

            <div className="min-h-[200px] flex flex-col items-center justify-center animate-pulse">
                
                <Music size={64} className="text-zinc-500" />
            
            </div>

        );

    }

    const HasSyllables = CurrentLine.syllabus && CurrentLine.syllabus.length > 0;

    return (

        <div key={CurrentLineIndex} className="lyric-line-active text-center max-w-4xl mx-auto px-4">
            
            <div className="text-3xl md:text-4xl font-semibold leading-relaxed tracking-wide">
                
                {HasSyllables ? (

                    <div className="flex flex-wrap justify-center gap-x-3 gap-y-1">
                        
                        {(() => {

                            const Words: any[][] = [];
                            let CurrentWord: any[] = [];

                            CurrentLine.syllabus!.forEach((Syllable) => {

                                CurrentWord.push(Syllable);

                                if (Syllable.text.endsWith(' ')) {

                                    Words.push(CurrentWord);
                                    CurrentWord = [];

                                }

                            });

                            if (CurrentWord.length > 0) Words.push(CurrentWord);

                            return Words.map((Word, WordIndex) => (

                                <span key={WordIndex} className="whitespace-nowrap inline-block">
                                    
                                    {Word.map((Syllable, SyllableIndex) => {

                                        const IsActive = CurrentTime >= Syllable.time;

                                        return (

                                            <span key={SyllableIndex} className={`transition-colors ease-linear ${IsActive ? 'text-white' : 'text-zinc-600'}`} style={{ transitionDuration: `${IsActive && Syllable.duration > 200 ? Syllable.duration : 200}ms` }} >
                                                
                                                {Syllable.text}

                                            </span>

                                        );

                                    })}

                                </span>

                            ));

                        })()}

                    </div>

                ) : (

                    <span className="text-white transition-colors duration-500">
                        
                        {CurrentLine.text}
                        
                    </span>
                    
                )}

            </div>

        </div>

    );

}

export default Lyrics;