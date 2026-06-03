import { LyricsLine, LyricsResponse, LyricsSyllabus } from '../Types';

const BINIMUM_API = 'https://lyrics-api.binimum.org/';
const ITUNES_NS = 'http://music.apple.com/lyric-ttml-internal';

interface BinimumSearchResult {

    lyricsUrl: string;
    timing_type: string;

}

interface BinimumSearchResponse {

    total: number;
    results: BinimumSearchResult[];

}

const attr = (el: Element, name: string): string | null => {

    return el.getAttribute(name) ?? el.getAttributeNS(ITUNES_NS, name.replace(/^itunes:/, ''));

};

export const parseTTMLTime = (value: string | null): number => {

    if (!value) return 0;

    const Parts = value.split(':').map(parseFloat);

    let Seconds = 0;

    if (Parts.length === 3) {

        Seconds = Parts[0] * 3600 + Parts[1] * 60 + Parts[2];

    } else if (Parts.length === 2) {

        Seconds = Parts[0] * 60 + Parts[1];

    } else {

        Seconds = Parts[0];

    }

    return Math.round(Seconds * 1000);

};

const getSongPart = (el: Element): string | undefined => {

    let Node: Element | null = el;

    while (Node) {

        const Part = attr(Node, 'itunes:songPart');

        if (Part) return Part;

        Node = Node.parentElement;

    }

};

const parseParagraph = (p: Element): LyricsLine | null => {

    const Begin = parseTTMLTime(p.getAttribute('begin'));
    const End = parseTTMLTime(p.getAttribute('end'));

    if (!p.getAttribute('begin')) return null;

    const Spans = p.querySelectorAll('span');

    if (Spans.length === 0) {

        const Text = p.textContent?.trim();

        if (!Text) return null;

        return {

            time: Begin,
            duration: Math.max(0, End - Begin),
            text: Text,
            element: {

                key: attr(p, 'itunes:key') ?? undefined,
                songPart: getSongPart(p),

            },

        };

    }

    const Syllabus: LyricsSyllabus[] = [];
    let WordSpans: LyricsSyllabus[] = [];

    const FlushWord = (TrailingSpace: boolean) => {

        if (WordSpans.length === 0) return;

        if (TrailingSpace) {

            const Last = WordSpans[WordSpans.length - 1];
            Last.text += ' ';

        }

        Syllabus.push(...WordSpans);
        WordSpans = [];

    };

    for (const Child of p.childNodes) {

        if (Child.nodeType === Node.ELEMENT_NODE && (Child as Element).localName === 'span') {

            const Span = Child as Element;
            const Text = Span.textContent ?? '';

            if (!Text) continue;

            const SpanBegin = parseTTMLTime(Span.getAttribute('begin'));
            const SpanEnd = parseTTMLTime(Span.getAttribute('end'));

            WordSpans.push({

                time: SpanBegin,
                duration: Math.max(0, SpanEnd - SpanBegin),
                text: Text,

            });

        } else if (Child.nodeType === Node.TEXT_NODE && /\s/.test(Child.textContent ?? '')) {

            FlushWord(true);

        }

    }

    FlushWord(false);

    if (Syllabus.length === 0) return null;

    return {

        time: Begin,
        duration: Math.max(0, End - Begin),
        text: Syllabus.map((s) => s.text.trimEnd()).join(''),
        syllabus: Syllabus,
        element: {

            key: attr(p, 'itunes:key') ?? undefined,
            songPart: getSongPart(p),

        },

    };

};

export const parseTTML = (xml: string, timingType?: string): LyricsResponse | null => {

    const Doc = new DOMParser().parseFromString(xml, 'application/xml');

    if (Doc.querySelector('parsererror')) return null;

    const Root = Doc.documentElement;

    const Timing = attr(Root, 'itunes:timing') ?? timingType ?? '';
    const Type = Timing.toLowerCase() === 'word' ? 'Word' : 'Line';

    const SongWriters = [...Doc.querySelectorAll('songwriter')].map((n) => n.textContent?.trim()).filter((s): s is string => !!s);

    const Lyrics: LyricsLine[] = [];

    for (const p of Doc.querySelectorAll('p')) {

        const Line = parseParagraph(p);

        if (Line) Lyrics.push(Line);

    }

    if (Lyrics.length === 0) return null;

    const LeadingSilence = Doc.querySelector('iTunesMetadata')?.getAttribute('leadingSilence');

    return {

        type: Type,

        metadata: {

            source: 'binimum',
            songWriters: SongWriters.length > 0 ? SongWriters : undefined,
            language: Root.getAttribute('xml:lang') ?? undefined,
            leadingSilence: LeadingSilence ?? undefined,

        },

        lyrics: Lyrics,

    };

};

export const fetchBinimumLyrics = async (Title: string, Artist: string, Album: string, DurationSeconds: number): Promise<LyricsResponse | null> => {

    const Params = new URLSearchParams({

        track: Title,

        artist: Artist,
        album: Album,

        duration: String(Math.round(DurationSeconds)),

    });

    const Search = await fetch(`${BINIMUM_API}?${Params}`);

    if (!Search.ok) return null;

    const Data: BinimumSearchResponse = await Search.json();

    if (!Data.total || !Data.results?.[0]?.lyricsUrl) return null;

    const { lyricsUrl, timing_type } = Data.results[0];

    const Ttml = await fetch(lyricsUrl);
    if (!Ttml.ok) return null;

    return parseTTML(await Ttml.text(), timing_type);

};
