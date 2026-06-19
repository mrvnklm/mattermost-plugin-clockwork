// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

// Crisp inline-SVG icons (currentColor) — replaces unreliable icon-font glyphs.

type Props = {size?: number};

const base = (size: number): React.SVGProps<SVGSVGElement> => ({
    width: size,
    height: size,
    viewBox: '0 0 24 24',
    fill: 'none',
    stroke: 'currentColor',
    strokeWidth: 2,
    strokeLinecap: 'round',
    strokeLinejoin: 'round',
    'aria-hidden': true,
});

export const ClockIcon = ({size = 18}: Props): JSX.Element => (
    <svg {...base(size)}>
        <circle
            cx='12'
            cy='12'
            r='9'
        />
        <polyline points='12 7 12 12 15 14'/>
    </svg>
);

export const PlayIcon = ({size = 16}: Props): JSX.Element => (
    <svg
        {...base(size)}
        fill='currentColor'
        stroke='none'
    >
        <path d='M8 5.5v13l11-6.5z'/>
    </svg>
);

export const StopIcon = ({size = 16}: Props): JSX.Element => (
    <svg
        {...base(size)}
        fill='currentColor'
        stroke='none'
    >
        <rect
            x='6'
            y='6'
            width='12'
            height='12'
            rx='2'
        />
    </svg>
);

export const PauseIcon = ({size = 16}: Props): JSX.Element => (
    <svg
        {...base(size)}
        fill='currentColor'
        stroke='none'
    >
        <rect
            x='6.5'
            y='5'
            width='4'
            height='14'
            rx='1'
        />
        <rect
            x='13.5'
            y='5'
            width='4'
            height='14'
            rx='1'
        />
    </svg>
);

export const PencilIcon = ({size = 16}: Props): JSX.Element => (
    <svg {...base(size)}>
        <path d='M12 20h9'/>
        <path d='M16.5 3.5a2.12 2.12 0 0 1 3 3L7 19l-4 1 1-4z'/>
    </svg>
);

export const PlusIcon = ({size = 14}: Props): JSX.Element => (
    <svg {...base(size)}>
        <line
            x1='12'
            y1='5'
            x2='12'
            y2='19'
        />
        <line
            x1='5'
            y1='12'
            x2='19'
            y2='12'
        />
    </svg>
);

export const ChevronLeftIcon = ({size = 16}: Props): JSX.Element => (
    <svg {...base(size)}>
        <polyline points='15 18 9 12 15 6'/>
    </svg>
);

export const ChevronRightIcon = ({size = 16}: Props): JSX.Element => (
    <svg {...base(size)}>
        <polyline points='9 18 15 12 9 6'/>
    </svg>
);

export const LockIcon = ({size = 13}: Props): JSX.Element => (
    <svg {...base(size)}>
        <rect
            x='4'
            y='11'
            width='16'
            height='10'
            rx='2'
        />
        <path d='M8 11V7a4 4 0 0 1 8 0v4'/>
    </svg>
);
