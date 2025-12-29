import React from 'react';

// Calculates percentage and time from mouse/touch position
export const CalculateTimeFromPosition = (ClientX: number, Rect: DOMRect, DurationSeconds: number): number => {

    const X = ClientX - Rect.left;

    const Percentage = Math.max(0, Math.min(1, X / Rect.width));
    return Percentage * DurationSeconds * 1000;

};

// Handles click events on progress bar
export const HandleProgressBarClick = (Event: React.MouseEvent<HTMLDivElement>, DurationSeconds: number, OnSeek: (time: number) => void ) => {
   
    const Rect = Event.currentTarget.getBoundingClientRect();
    const NewTime = CalculateTimeFromPosition(Event.clientX, Rect, DurationSeconds);

    OnSeek(Math.floor(NewTime));

};

// Handles mouse down events for dragging
export const HandleProgressBarMouseDown = (Event: React.MouseEvent<HTMLDivElement>, DurationSeconds: number, OnTimeUpdate: (time: number) => void, OnSeek: (time: number) => void ) => {

    const HandleMouseMove = (MoveEvent: MouseEvent) => {

        const Rect = Event.currentTarget.getBoundingClientRect();
        const NewTime = CalculateTimeFromPosition(MoveEvent.clientX, Rect, DurationSeconds);
        
        OnTimeUpdate(Math.floor(NewTime));

    };

    const HandleMouseUp = (UpEvent: MouseEvent) => {

        const Rect = Event.currentTarget.getBoundingClientRect();
        const NewTime = CalculateTimeFromPosition(UpEvent.clientX, Rect, DurationSeconds);
        
        OnSeek(Math.floor(NewTime));

        document.removeEventListener('mousemove', HandleMouseMove);
        document.removeEventListener('mouseup', HandleMouseUp);

    };

    document.addEventListener('mousemove', HandleMouseMove);
    document.addEventListener('mouseup', HandleMouseUp);

};

// Handles touch start events for dragging
export const HandleProgressBarTouchStart = (Event: React.TouchEvent<HTMLDivElement>, DurationSeconds: number, OnTimeUpdate: (time: number) => void, OnSeek: (time: number) => void ) => {
    
    const HandleTouchMove = (MoveEvent: TouchEvent) => {

        const Rect = Event.currentTarget.getBoundingClientRect();
        const NewTime = CalculateTimeFromPosition(MoveEvent.touches[0].clientX, Rect, DurationSeconds);
        
        OnTimeUpdate(Math.floor(NewTime));

    };

    const HandleTouchEnd = (EndEvent: TouchEvent) => {

        const Rect = Event.currentTarget.getBoundingClientRect();
        const NewTime = CalculateTimeFromPosition(EndEvent.changedTouches[0].clientX, Rect, DurationSeconds);
        
        OnSeek(Math.floor(NewTime));

        document.removeEventListener('touchmove', HandleTouchMove);
        document.removeEventListener('touchend', HandleTouchEnd);

    };

    document.addEventListener('touchmove', HandleTouchMove);
    document.addEventListener('touchend', HandleTouchEnd);

};