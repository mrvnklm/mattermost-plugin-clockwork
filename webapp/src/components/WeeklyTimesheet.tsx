// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import type {TimeEntry} from 'client/Client';
import {t} from 'i18n';
import React, {useMemo} from 'react';
import {decimalHours, localClock, localDayKey, netSeconds} from 'utils/time';

import {LockIcon, PencilIcon} from 'components/icons';
import StatusBadge from 'components/StatusBadge';

type Props = {
    entries: TimeEntry[];
    now: number;
    onSelect: (entry: TimeEntry) => void;

    // When approval is enabled, show a status badge column.
    approvalEnabled?: boolean;
};

export default function WeeklyTimesheet({entries, now, onSelect, approvalEnabled = false}: Props): JSX.Element {
    // Newest first; the server already returns start_at DESC.
    const sorted = useMemo(
        () => [...entries].sort((a, b) => b.start_at - a.start_at),
        [entries],
    );
    const weekNet = useMemo(
        () => sorted.reduce((sum, e) => sum + netSeconds(e, now), 0),
        [sorted, now],
    );

    if (sorted.length === 0) {
        return <div className='tt-empty'>{t('noEntries')}</div>;
    }

    // Total columns before the trailing action cell: date, start, end, break,
    // hours (5), plus an optional status column.
    const footSpan = approvalEnabled ? 5 : 4;

    let lastDay = '';
    return (
        <div className='tt-tablewrap'>
            <table className='tt-table tt-table--compact'>
                <thead>
                    <tr>
                        <th>{t('date')}</th>
                        <th>{t('startTime')}</th>
                        <th>{t('endTime')}</th>
                        <th>{t('breakMinutes')}</th>
                        {approvalEnabled && <th>{t('status')}</th>}
                        <th className='tt-num'>{t('hours')}</th>
                        <th className='tt-act'/>
                    </tr>
                </thead>
                <tbody>
                    {sorted.map((e) => {
                        const day = localDayKey(e.start_at);
                        const showDay = day !== lastDay;
                        lastDay = day;
                        const running = e.end_at === 0;
                        const breakMin = Math.round((e.break_seconds + (e.break_started_at ? Math.max(0, (now - e.break_started_at) / 1000) : 0)) / 60);
                        return (
                            <tr key={e.id}>
                                <td className='tt-date'>{showDay ? day.slice(5) : ''}</td>
                                <td>{localClock(e.start_at)}</td>
                                <td>{running ? '–' : localClock(e.end_at)}</td>
                                <td>{breakMin}</td>
                                {approvalEnabled && (
                                    <td><StatusBadge status={e.status}/></td>
                                )}
                                <td className='tt-num'>{decimalHours(netSeconds(e, now))}</td>
                                <td className='tt-act'>
                                    {running ? (
                                        <span className='tt-live-tag'>{t('running')}</span>
                                    ) : (
                                        <button
                                            className='tt-iconbtn'
                                            title={e.locked ? t('locked') : t('edit')}
                                            aria-label={e.locked ? t('locked') : t('edit')}
                                            onClick={() => onSelect(e)}
                                        >
                                            {e.locked ? <LockIcon/> : <PencilIcon/>}
                                        </button>
                                    )}
                                </td>
                            </tr>
                        );
                    })}
                </tbody>
                <tfoot>
                    <tr>
                        <td colSpan={footSpan}>{t('total')}</td>
                        <td className='tt-num'>{decimalHours(weekNet)}</td>
                        <td className='tt-act'/>
                    </tr>
                </tfoot>
            </table>
        </div>
    );
}
