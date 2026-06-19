// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {t} from 'i18n';
import React, {useMemo} from 'react';
import {decimalHours, netSeconds} from 'utils/time';

import type {AdminRow} from 'components/admin/AdminTable';

type Props = {
    rows: AdminRow[];
    now: number;
};

type ProjectStat = {project: string; entries: number; net: number};

// AdminProjectSummary aggregates net hours per project across the visible rows
// (a common reporting need: hours per client/project). Entries without a
// project are grouped under the localized "No project" bucket.
export default function AdminProjectSummary({rows, now}: Props): JSX.Element {
    const stats = useMemo(() => {
        const byProject = new Map<string, ProjectStat>();
        for (const r of rows) {
            const project = r.entry.project || t('noProject');
            const stat = byProject.get(project) ?? {project, entries: 0, net: 0};
            stat.entries += 1;
            stat.net += netSeconds(r.entry, now);
            byProject.set(project, stat);
        }
        return [...byProject.values()].sort((a, b) => b.net - a.net);
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
                    <th>{t('project_')}</th>
                    <th>{t('entriesCount')}</th>
                    <th className='tt-num'>{t('hours')}</th>
                </tr>
            </thead>
            <tbody>
                {stats.map((s) => (
                    <tr key={s.project}>
                        <td>{s.project}</td>
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
