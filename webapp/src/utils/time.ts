// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import type {TimeEntry} from 'client/Client';

// currentTz is the IANA timezone used for ALL day/clock/week math so the panel
// agrees with the server (which groups by the user's Mattermost timezone). Set
// once at init via setTimezone; undefined ⇒ fall back to the browser's local
// timezone.
let currentTz: string | undefined;

// setTimezone configures the timezone. Pass an IANA name (e.g. "Europe/Berlin");
// empty/invalid resets to browser-local.
export function setTimezone(tz?: string): void {
    currentTz = tz || undefined;
}

const pad = (n: number) => String(n).padStart(2, '0');

type Wall = {y: number; mo: number; d: number; h: number; mi: number};

// wallPartsInTz returns the wall-clock components of an instant in tz.
function wallPartsInTz(ms: number, tz: string): Wall {
    const parts = new Intl.DateTimeFormat('en-US', {
        timeZone: tz,
        hourCycle: 'h23',
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
    }).formatToParts(ms);
    const m: Record<string, string> = {};
    for (const p of parts) {
        m[p.type] = p.value;
    }
    return {y: Number(m.year), mo: Number(m.month), d: Number(m.day), h: Number(m.hour), mi: Number(m.minute)};
}

// tzOffsetMs returns the offset (tz wall-clock minus UTC) at the given instant.
function tzOffsetMs(ms: number, tz: string): number {
    const w = wallPartsInTz(ms, tz);
    const asUTC = Date.UTC(w.y, w.mo - 1, w.d, w.h, w.mi);

    // Round ms to the minute we formatted to, so seconds don't skew the offset.
    return asUTC - (Math.floor(ms / 60000) * 60000);
}

// zonedWallToUtc converts a wall-clock time in tz to a UTC-millis instant,
// resolving the offset (incl. DST) at that wall time. Day/month overflow and
// negatives are normalized by Date.UTC, so callers may pass d±n freely.
function zonedWallToUtc(w: Wall, tz: string): number {
    const guess = Date.UTC(w.y, w.mo - 1, w.d, w.h, w.mi);
    const offset = tzOffsetMs(guess, tz);
    const utc = guess - offset;

    // Re-resolve once: near a DST transition the first offset can be off.
    const offset2 = tzOffsetMs(utc, tz);
    return offset2 === offset ? utc : guess - offset2;
}

// netSeconds mirrors server store.TimeEntry.NetSeconds: gross (start→end) minus
// completed breaks minus any active break, clamped at 0. For a running entry,
// `now` is used as the end.
export function netSeconds(e: TimeEntry, now: number): number {
    const end = e.end_at === 0 ? now : e.end_at;
    const gross = Math.floor((end - e.start_at) / 1000);
    let breaks = e.break_seconds;
    if (e.break_started_at !== 0) {
        breaks += Math.floor((now - e.break_started_at) / 1000);
    }
    const net = gross - breaks;
    return net < 0 ? 0 : net;
}

// formatHMS renders seconds as HH:MM:SS (zero-padded).
export function formatHMS(totalSeconds: number): string {
    const s = Math.max(0, Math.floor(totalSeconds));
    const h = Math.floor(s / 3600);
    const m = Math.floor((s % 3600) / 60);
    return [h, m, s % 60].map(pad).join(':');
}

// formatHM renders seconds as HH:MM (zero-padded).
export function formatHM(totalSeconds: number): string {
    const s = Math.max(0, Math.floor(totalSeconds));
    const h = Math.floor(s / 3600);
    const m = Math.floor((s % 3600) / 60);
    return [h, m].map(pad).join(':');
}

// decimalHours renders seconds as decimal hours with 2 dp (matches the CSV /
// summary "net_hours" convention).
export function decimalHours(totalSeconds: number): string {
    return (Math.max(0, totalSeconds) / 3600).toFixed(2);
}

// weekRange returns [fromMs, toMs) covering the Monday→Monday week containing
// `ref`, computed in the active timezone so it lines up with localDayKey and the
// server's day grouping (no browser-vs-MM-timezone drift, DST-safe).
export function weekRange(ref: Date): {from: number; to: number} {
    if (currentTz) {
        const w = wallPartsInTz(ref.getTime(), currentTz);
        const dow = (new Date(Date.UTC(w.y, w.mo - 1, w.d)).getUTCDay() + 6) % 7; // 0 = Monday
        const from = zonedWallToUtc({y: w.y, mo: w.mo, d: w.d - dow, h: 0, mi: 0}, currentTz);
        const to = zonedWallToUtc({y: w.y, mo: w.mo, d: (w.d - dow) + 7, h: 0, mi: 0}, currentTz);
        return {from, to};
    }
    const d = new Date(ref.getFullYear(), ref.getMonth(), ref.getDate());
    d.setDate(d.getDate() - ((d.getDay() + 6) % 7));
    const to = new Date(d);
    to.setDate(to.getDate() + 7);
    return {from: d.getTime(), to: to.getTime()};
}

// localDayKey formats an instant as a YYYY-MM-DD key in the active timezone.
export function localDayKey(ms: number): string {
    if (currentTz) {
        return new Intl.DateTimeFormat('en-CA', {
            timeZone: currentTz,
            year: 'numeric',
            month: '2-digit',
            day: '2-digit',
        }).format(ms);
    }
    const d = new Date(ms);
    return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

// localClock formats an instant as HH:MM in the active timezone.
export function localClock(ms: number): string {
    if (currentTz) {
        return new Intl.DateTimeFormat('en-GB', {
            timeZone: currentTz,
            hour: '2-digit',
            minute: '2-digit',
            hour12: false,
        }).format(ms);
    }
    const d = new Date(ms);
    return `${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

// dateInputValue formats a Date as the "YYYY-MM-DD" string an <input type="date">
// expects, using the local calendar day. Used for the report range pickers.
export function dateInputValue(d: Date): string {
    return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

// toDatetimeInput converts unix millis to an <input type="datetime-local">
// value ("YYYY-MM-DDTHH:mm") in the active timezone, so the field shows the same
// wall-clock time the rest of the panel displays.
export function toDatetimeInput(ms: number): string {
    if (!ms) {
        return '';
    }
    const w = currentTz ? wallPartsInTz(ms, currentTz) : (() => {
        const d = new Date(ms);
        return {y: d.getFullYear(), mo: d.getMonth() + 1, d: d.getDate(), h: d.getHours(), mi: d.getMinutes()};
    })();
    return `${w.y}-${pad(w.mo)}-${pad(w.d)}T${pad(w.h)}:${pad(w.mi)}`;
}

// fromDatetimeInput parses a datetime-local value as wall-clock time in the
// active timezone and returns unix millis (0 if empty/invalid). This is the
// inverse of toDatetimeInput, so an unchanged edit round-trips exactly.
export function fromDatetimeInput(value: string): number {
    if (!value) {
        return 0;
    }
    const m = value.match(/^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})/);
    if (!m) {
        return 0;
    }
    const w: Wall = {y: Number(m[1]), mo: Number(m[2]), d: Number(m[3]), h: Number(m[4]), mi: Number(m[5])};
    if (currentTz) {
        return zonedWallToUtc(w, currentTz);
    }
    return new Date(w.y, w.mo - 1, w.d, w.h, w.mi).getTime();
}
