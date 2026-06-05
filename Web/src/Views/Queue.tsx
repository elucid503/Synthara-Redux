import { useState } from 'react';

import { DndContext, DragOverlay, KeyboardSensor, PointerSensor, closestCenter, useSensor, useSensors, type DragEndEvent, type DragStartEvent, } from '@dnd-kit/core';
import { SortableContext, sortableKeyboardCoordinates, useSortable, verticalListSortingStrategy, } from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';

import { MoreHorizontal, ChevronDown, ChevronUp, GripVertical } from 'lucide-react';

import { Song } from '../Types';

interface QueueProps {

    Current: Song | null;

    PreviousSongs: Song[];
    UpcomingSongs: Song[];

    ActiveContextMenu: { type: 'Previous' | 'Upcoming', index: number, x: number, y: number } | null;
    SetActiveContextMenu: (Menu: { type: 'Previous' | 'Upcoming', index: number, x: number, y: number } | null) => void;

    OnMove: (FromIndex: number, ToIndex: number) => void;

    ControlsLocked?: boolean;

}

const UpcomingID = (Index: number) => `upcoming-${Index}`;

const NormalizeCoverURL = (URL: string): string => {

    return URL.replace(/=w\d+-h\d+(-l\d+)?(-rj)?/g, '=w512-h512-l90-rj');

};

interface SongRowProps {

    Song: Song;
    Index?: number;

    ShowIndex?: boolean;
    ShowMenu?: boolean;

    OnMenuClick?: (E: React.MouseEvent<HTMLButtonElement>) => void;

}

function SongRow({ Song, Index, ShowIndex = false, ShowMenu = false, OnMenuClick }: SongRowProps) {

    return (

        <>

            {ShowIndex && Index != null && (

                <div className="w-5 shrink-0 text-center text-sm font-medium text-zinc-500">{Index + 1}</div>

            )}

            <img src={NormalizeCoverURL(Song.cover)} referrerPolicy="no-referrer" className="h-11 w-11 shrink-0 rounded-lg object-cover" />

            <div className="min-w-0 flex-1">

                <div className="truncate text-sm font-medium">{Song.title}</div>
                <div className="truncate text-xs text-zinc-400">{Song.artists.join(', ')}</div>

            </div>

            {Song.unavailable && (

                <span className="shrink-0 rounded bg-red-400/10 px-1.5 py-0.5 text-xs font-semibold text-red-400">Unavailable</span>

            )}

            <div className="w-fit shrink-0 text-right text-xs font-semibold text-zinc-500">{Song.duration.formatted}</div>

            {ShowMenu && OnMenuClick && (

                <button type="button" onClick={OnMenuClick} className="context-menu-trigger mr-1 shrink-0 text-zinc-400 transition-colors hover:text-white" onPointerDown={(E) => E.stopPropagation()}>

                    <MoreHorizontal size={16} />

                </button>

            )}

        </>

    );

}

interface SortableUpcomingRowProps {

    Song: Song;
    Index: number;
    ControlsLocked: boolean;

    ActiveContextMenu: QueueProps['ActiveContextMenu'];
    SetActiveContextMenu: QueueProps['SetActiveContextMenu'];

}

function SortableUpcomingRow({ Song, Index, ControlsLocked, ActiveContextMenu, SetActiveContextMenu }: SortableUpcomingRowProps) {

    const { attributes, listeners, setNodeRef, transform, transition, isDragging, } = useSortable({

        id: UpcomingID(Index),
        disabled: ControlsLocked,

    });

    const Style = {

        transform: CSS.Transform.toString(transform),
        transition,

    };

    return (

        <div className={`group relative flex items-center gap-2 rounded-lg bg-white/5 p-3 transition-shadow sm:gap-3 ${isDragging ? 'z-10 opacity-40 shadow-lg shadow-black/20' : 'hover:bg-white/10'}`}

            ref={setNodeRef}
            style={Style}

            onContextMenu={(E) => {

                if (ControlsLocked) return;

                E.preventDefault();
                E.stopPropagation();

                SetActiveContextMenu(

                    ActiveContextMenu?.index === Index && ActiveContextMenu?.type === 'Upcoming'
                        ? null
                        : { type: 'Upcoming', index: Index, x: E.clientX, y: E.clientY },

                );

            }}
        >

            {!ControlsLocked && (

                <button className="touch-none shrink-0 cursor-grab text-zinc-500 transition-colors hover:text-white active:cursor-grabbing"

                    type="button"
                    aria-label={`Reorder ${Song.title}`}

                    {...attributes}
                    {...listeners}

                >

                    <GripVertical size={16} />

                </button>

            )}

            <SongRow Song={Song} Index={Index} ShowIndex ShowMenu={!ControlsLocked}

                OnMenuClick={(E) => {

                    E.stopPropagation();
                    const Rect = E.currentTarget.getBoundingClientRect();

                    SetActiveContextMenu(ActiveContextMenu?.index === Index && ActiveContextMenu?.type === 'Upcoming' ? null : { type: 'Upcoming', index: Index, x: Rect.right, y: Rect.bottom });

                }}
            />

        </div>

    );

}

function Queue({ Current, PreviousSongs, UpcomingSongs, ActiveContextMenu, SetActiveContextMenu, OnMove, ControlsLocked = false }: QueueProps) {

    const [ShowPrevious, SetShowPrevious] = useState(false);
    const [ActiveDragIndex, SetActiveDragIndex] = useState<number | null>(null);

    const Sensors = useSensors(

        useSensor(PointerSensor, { activationConstraint: { distance: 6 } }),
        useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),

    );

    const UpcomingIDs = UpcomingSongs.map((_, Index) => UpcomingID(Index));

    const HandleDragStart = (Event: DragStartEvent) => {

        const Index = UpcomingIDs.indexOf(String(Event.active.id));

        SetActiveDragIndex(Index >= 0 ? Index : null);

    };

    const HandleDragEnd = (Event: DragEndEvent) => {

        SetActiveDragIndex(null);

        const { active, over } = Event;

        if (!over || active.id === over.id) return;

        const FromIndex = UpcomingIDs.indexOf(String(active.id));
        const ToIndex = UpcomingIDs.indexOf(String(over.id));

        if (FromIndex < 0 || ToIndex < 0 || FromIndex === ToIndex) return;

        OnMove(FromIndex, ToIndex);

    };

    const HandleDragCancel = () => {

        SetActiveDragIndex(null);

    };

    const RenderPreviousSong = (Song: Song, Index: number) => (

        <div key={`prev-${Index}`} className="group relative flex items-center gap-3 rounded-lg bg-white/5 p-3 opacity-60 sm:gap-4"

            onContextMenu={(E) => {

                if (ControlsLocked) return;

                E.preventDefault();
                E.stopPropagation();

                SetActiveContextMenu(ActiveContextMenu?.index === Index && ActiveContextMenu?.type === 'Previous' ? null : { type: 'Previous', index: Index, x: E.clientX, y: E.clientY });

            }}
        >

            <SongRow Song={Song} ShowMenu={!ControlsLocked}

                OnMenuClick={(E) => {

                    E.stopPropagation();
                    const Rect = E.currentTarget.getBoundingClientRect();

                    SetActiveContextMenu(ActiveContextMenu?.index === Index && ActiveContextMenu?.type === 'Previous' ? null : { type: 'Previous', index: Index, x: Rect.right, y: Rect.bottom });

                }}

            />

        </div>

    );

    const ActiveDragSong = ActiveDragIndex != null ? UpcomingSongs[ActiveDragIndex] : null;

    return (

        <div className="mx-auto h-fit w-full max-w-4xl">

            {PreviousSongs.length > 0 && (

                <div className="mb-6">

                    <button onClick={() => SetShowPrevious(!ShowPrevious)} className="flex items-center gap-2 text-xs font-bold uppercase tracking-wider text-zinc-500 transition-colors hover:text-white">

                        Previous
                        {ShowPrevious ? <ChevronUp className="mb-0.5" size={16} /> : <ChevronDown className="mb-0.5" size={16} />}

                    </button>

                    {ShowPrevious && (

                        <div className="mt-4 space-y-2">

                            {PreviousSongs.map(RenderPreviousSong)}

                        </div>

                    )}

                </div>

            )}

            <div className="mb-8">

                <h2 className="mb-4 text-xs font-bold uppercase tracking-wider text-zinc-500">Now Playing</h2>

                {Current && (

                    <div className="flex items-center gap-4 rounded-xl border border-white/10 bg-white/[0.08] p-4 shadow-lg shadow-black/10">

                        <img src={NormalizeCoverURL(Current.cover)} referrerPolicy="no-referrer" className="h-16 w-16 rounded-lg object-cover shadow-lg" />

                        <div className="min-w-0 flex-1">

                            <div className="truncate text-lg font-bold">{Current.title}</div>
                            <div className="truncate text-zinc-400">{Current.artists.join(', ')}</div>

                        </div>

                        <div className="mr-2 font-semibold text-zinc-400">{Current.duration.formatted}</div>

                    </div>

                )}

            </div>

            {UpcomingSongs.length > 0 && (

                <div>

                    <h2 className="mb-4 text-xs font-bold uppercase tracking-wider text-zinc-500">Next Up</h2>

                    <DndContext sensors={Sensors} collisionDetection={closestCenter} onDragStart={HandleDragStart} onDragEnd={HandleDragEnd} onDragCancel={HandleDragCancel} >

                        <SortableContext items={UpcomingIDs} strategy={verticalListSortingStrategy}>

                            <div className="space-y-2">

                                {UpcomingSongs.map((Song, Index) => (

                                    <SortableUpcomingRow key={UpcomingID(Index)} Song={Song} Index={Index}

                                        ControlsLocked={ControlsLocked}

                                        ActiveContextMenu={ActiveContextMenu}
                                        SetActiveContextMenu={SetActiveContextMenu}

                                    />

                                ))}

                            </div>

                        </SortableContext>

                        <DragOverlay dropAnimation={{ duration: 180, easing: 'cubic-bezier(0.18, 0.67, 0.6, 1)' }}>

                            {ActiveDragSong && ActiveDragIndex != null ? (

                                <div className="flex items-center gap-2 rounded-lg border border-white/20 bg-zinc-900/95 p-3 shadow-2xl shadow-black/50 sm:gap-3">

                                    <GripVertical size={16} className="shrink-0 text-white" />

                                    <SongRow Song={ActiveDragSong} Index={ActiveDragIndex} ShowIndex />

                                </div>

                            ) : null}

                        </DragOverlay>

                    </DndContext>

                </div>

            )}

        </div>

    );

}

export default Queue;
