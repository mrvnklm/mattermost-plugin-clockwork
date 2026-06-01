// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {t} from 'i18n';
import React, {useMemo} from 'react';
import {decimalHours, netSeconds} from 'utils/time';

import type {AdminRow} from 'components/admin/AdminTable';

type Props = {
    rows: AdminRow[];
    now: number;
    onSelectUser?: (username: string) => void;
};

type UserStat = {username: string; entries: number; net: number};

// AdminUserSummary is the primary "Team report" view: one row per distinct
// user with their entry count and total net hours, sorted by hours desc. The
// user name is a button that drives the parent's client-side user filter.
export default function AdminUserSummary({rows, now, onSelectUser}: Props): JSX.Element {
    // Aggregate per username, then sort by net seconds desc. Sorting derived
    // data avoids mutating the caller's array.
    const stats = useMemo(() => {
        const byUser = new Map<string, UserStat>();
        for (const r of rows) {
            const stat = byUser.get(r.username) ?? {username: r.username, entries: 0, net: 0};
            stat.entries += 1;
            stat.net += netSeconds(r.entry, now);
            byUser.set(r.username, stat);
        }
        return [...byUser.values()].sort((a, b) => b.net - a.net);
    }, [rows, now]);

    const totalNet = useMemo(
        () => stats.reduce((sum, s) => sum + s.net, 0),
        [stats],
    );
    const totalEntries = useMemo(
        () => stats.reduce((sum, s) => sum + s.entries, 0),
        [stats],
    );

    if (stats.length === 0) {
        return <div className='tt-empty'>{t('noEntries')}</div>;
    }

    return (
        <table className='tt-table'>
            <thead>
                <tr>
                    <th>{t('user')}</th>
                    <th>{t('entriesCount')}</th>
                    <th className='tt-num'>{t('hours')}</th>
                </tr>
            </thead>
            <tbody>
                {stats.map((s) => (
                    <tr key={s.username}>
                        <td>
                            {onSelectUser ? (
                                <button
                                    className='tt-link'
                                    onClick={() => onSelectUser(s.username)}
                                >
                                    {s.username}
                                </button>
                            ) : s.username}
                        </td>
                        <td>{s.entries}</td>
                        <td className='tt-num'>{decimalHours(s.net)}</td>
                    </tr>
                ))}
            </tbody>
            <tfoot>
                <tr>
                    <td>{t('total')}</td>
                    <td>{totalEntries}</td>
                    <td className='tt-num'>{decimalHours(totalNet)}</td>
                </tr>
            </tfoot>
        </table>
    );
}
