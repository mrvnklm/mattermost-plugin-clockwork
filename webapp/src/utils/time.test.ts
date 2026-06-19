// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import type {TimeEntry} from 'client/Client';

import {
    decimalHours,
    formatHM,
    formatHMS,
    fromDatetimeInput,
    localClock,
    localDayKey,
    netSeconds,
    setTimezone,
    toDatetimeInput,
    weekRange,
} from './time';

// Build a minimal TimeEntry; only the fields netSeconds reads matter.
function entry(over: Partial<TimeEntry>): TimeEntry {
    return {
        id: 'x',
        user_id: 'u',
        start_at: 0,
        end_at: 0,
        break_seconds: 0,
        break_started_at: 0,
        project: '',
        description: '',
        status: 'open',
        locked: false,
        created_at: 0,
        updated_at: 0,
        ...over,
    };
}

// A fixed reference instant: 2024-03-13T12:00:00Z (a Wednesday).
const REF = Date.UTC(2024, 2, 13, 12, 0, 0);
const HOUR = 3600 * 1000;
const DAY = 24 * HOUR;

afterEach(() => {
    setTimezone(undefined);
});

describe('netSeconds', () => {
    it('computes gross minus breaks for a closed entry', () => {
        const e = entry({start_at: 0, end_at: 2 * HOUR, break_seconds: 600});
        expect(netSeconds(e, REF)).toBe((2 * 3600) - 600);
    });

    it('uses now as the end for a running entry', () => {
        const e = entry({start_at: REF - HOUR, end_at: 0});
        expect(netSeconds(e, REF)).toBe(3600);
    });

    it('adds the active (unfinished) break', () => {
        const e = entry({start_at: REF - (2 * HOUR), end_at: 0, break_started_at: REF - (30 * 60 * 1000)});

        // gross 2h, active break 30m => 1h30m net
        expect(netSeconds(e, REF)).toBe(90 * 60);
    });

    it('clamps negative net to 0', () => {
        const e = entry({start_at: 0, end_at: HOUR, break_seconds: 99999});
        expect(netSeconds(e, REF)).toBe(0);
    });
});

describe('formatting', () => {
    it('formatHMS zero-pads h:m:s', () => {
        expect(formatHMS(3661)).toBe('01:01:01');
        expect(formatHMS(0)).toBe('00:00:00');
        expect(formatHMS(-5)).toBe('00:00:00');
    });

    it('formatHM zero-pads h:m', () => {
        expect(formatHM(3661)).toBe('01:01');
        expect(formatHM(59)).toBe('00:00');
    });

    it('decimalHours renders 2dp', () => {
        expect(decimalHours(3600)).toBe('1.00');
        expect(decimalHours(5400)).toBe('1.50');
        expect(decimalHours(-100)).toBe('0.00');
    });
});

describe('datetime-local round-trip', () => {
    it('round-trips an unchanged edit exactly (browser-local)', () => {
        setTimezone(undefined);
        const ms = Date.UTC(2024, 5, 1, 10, 30); // arbitrary
        const s = toDatetimeInput(ms);
        expect(fromDatetimeInput(s)).toBe(fromDatetimeInput(toDatetimeInput(ms)));
        expect(s).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/);
    });

    it('round-trips in a fixed IANA timezone', () => {
        setTimezone('Europe/Berlin');
        const ms = Date.UTC(2024, 5, 1, 8, 15); // 10:15 Berlin (CEST)
        const s = toDatetimeInput(ms);
        expect(s).toBe('2024-06-01T10:15');
        expect(fromDatetimeInput(s)).toBe(ms);
    });

    it('handles a winter (CET) instant in Berlin', () => {
        setTimezone('Europe/Berlin');
        const ms = Date.UTC(2024, 0, 15, 9, 0); // 10:00 Berlin (CET, +1)
        expect(toDatetimeInput(ms)).toBe('2024-01-15T10:00');
        expect(fromDatetimeInput('2024-01-15T10:00')).toBe(ms);
    });

    it('empty values map to empty/0', () => {
        expect(toDatetimeInput(0)).toBe('');
        expect(fromDatetimeInput('')).toBe(0);
        expect(fromDatetimeInput('garbage')).toBe(0);
    });
});

describe('DST transitions (Europe/Berlin)', () => {
    it('round-trips a wall time on the spring-forward day after the gap', () => {
        setTimezone('Europe/Berlin');

        // 2024-03-31 02:00 CET jumps to 03:00 CEST. 10:00 wall is well clear of
        // the gap and must round-trip cleanly.
        const s = '2024-03-31T10:00';
        const ms = fromDatetimeInput(s);
        expect(toDatetimeInput(ms)).toBe(s);
    });

    it('round-trips a wall time on the autumn fall-back day', () => {
        setTimezone('Europe/Berlin');

        // 2024-10-27 03:00 CEST falls back to 02:00 CET.
        const s = '2024-10-27T10:00';
        const ms = fromDatetimeInput(s);
        expect(toDatetimeInput(ms)).toBe(s);
    });
});

describe('weekRange', () => {
    it('returns a 7-day Monday→Monday window (browser-local)', () => {
        setTimezone(undefined);
        const {from, to} = weekRange(new Date(REF));
        expect(to - from).toBe(7 * DAY);

        // Monday is day 1 in the local calendar of the from instant.
        const fromDow = new Date(from).getDay();
        expect(fromDow).toBe(1);
    });

    it('returns a DST-correct week in Berlin spanning the spring transition', () => {
        setTimezone('Europe/Berlin');

        // Week containing 2024-03-31 (spring forward) is Mon 25th → Mon Apr 1st.
        const ref = new Date(Date.UTC(2024, 2, 31, 12, 0));
        const {from, to} = weekRange(ref);

        // The week starts on Monday 2024-03-25 00:00 Berlin.
        expect(localDayKey(from)).toBe('2024-03-25');

        // Spans one fewer hour than a flat 7 days due to the lost hour.
        expect(to - from).toBe((7 * DAY) - HOUR);
    });

    it('returns a DST-correct week spanning the autumn transition', () => {
        setTimezone('Europe/Berlin');

        // Week containing 2024-10-27 (fall back) is Mon 21st → Mon 28th.
        const ref = new Date(Date.UTC(2024, 9, 27, 12, 0));
        const {from, to} = weekRange(ref);
        expect(localDayKey(from)).toBe('2024-10-21');

        // Gains one extra hour from the repeated hour.
        expect(to - from).toBe((7 * DAY) + HOUR);
    });
});

describe('localDayKey / localClock', () => {
    it('groups by the configured timezone, not the host', () => {
        setTimezone('Asia/Tokyo');

        // 2024-03-13T22:00Z is 2024-03-14 07:00 in Tokyo (+9).
        const ms = Date.UTC(2024, 2, 13, 22, 0);
        expect(localDayKey(ms)).toBe('2024-03-14');
        expect(localClock(ms)).toBe('07:00');
    });
});
