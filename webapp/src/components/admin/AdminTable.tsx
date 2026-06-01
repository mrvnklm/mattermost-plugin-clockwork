// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import type {TimeEntry} from 'client/Client';
import {t} from 'i18n';
import React, {useMemo} from 'react';
import {decimalHours, localClock, localDayKey, netSeconds} from 'utils/time';

export type AdminRow = {entry: TimeEntry; username: string};

type Props = {
    rows: AdminRow[];
    now: number;
};

export default function AdminTable({rows, now}: Props): JSX.Element {
    // Newest first; sort a copy so we never mutate the caller's array.
    const sorted = useMemo(
        () => [...rows].sort((a, b) => b.entry.start_at - a.entry.start_at),
        [rows],
    );
    const totalNet = useMemo(
        () => sorted.reduce((sum, r) => sum + netSeconds(r.entry, now), 0),
        [sorted, now],
    );

    if (sorted.length === 0) {
        return <div className='tt-empty'>{t('noEntries')}</div>;
    }

    return (
        <table className='tt-table'>
            <thead>
                <tr>
                    <th>{t('user')}</th>
                    <th>{t('date')}</th>
                    <th>{t('startTime')}</th>
                    <th>{t('endTime')}</th>
                    <th>{t('breakMinutes')}</th>
                    <th className='tt-num'>{t('hours')}</th>
                </tr>
            </thead>
            <tbody>
                {sorted.map((r) => {
                    const e = r.entry;
                    const running = e.end_at === 0;
                    const breakMin = Math.round((e.break_seconds + (e.break_started_at ? Math.max(0, (now - e.break_started_at) / 1000) : 0)) / 60);
                    return (
                        <tr key={e.id}>
                            <td>{r.username}</td>
                            <td className='tt-date'>{localDayKey(e.start_at)}</td>
                            <td>{localClock(e.start_at)}</td>
                            <td>
                                {running ? (
                                    <>
                                        {'–'}
                                        <span className='tt-live-tag'>{t('running')}</span>
                                    </>
                                ) : localClock(e.end_at)}
                            </td>
                            <td>{breakMin}</td>
                            <td className='tt-num'>{decimalHours(netSeconds(e, now))}</td>
                        </tr>
                    );
                })}
            </tbody>
            <tfoot>
                <tr>
                    <td colSpan={5}>{t('total')}</td>
                    <td className='tt-num'>{decimalHours(totalNet)}</td>
                </tr>
            </tfoot>
        </table>
    );
}
