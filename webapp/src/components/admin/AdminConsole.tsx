// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import Client from 'client/Client';
import type {TimeEntry} from 'client/Client';
import {t} from 'i18n';
import React, {useEffect, useMemo, useState} from 'react';
import {ensureStyles} from 'styles';
import {decimalHours, netSeconds} from 'utils/time';

import type {AdminRow} from 'components/admin/AdminTable';
import AdminTable from 'components/admin/AdminTable';

ensureStyles();

// pad2 zero-pads a number for YYYY-MM-DD date-input strings.
function pad2(n: number): string {
    return String(n).padStart(2, '0');
}

// dateInput formats a Date as the YYYY-MM-DD string an <input type='date'>
// expects (local calendar day).
function dateInput(d: Date): string {
    return `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())}`;
}

// AdminConsole renders the System Console "Team report". Mattermost gates the
// custom setting to system admins; the server re-checks on every request. Any
// custom-setting props Mattermost passes are intentionally ignored.
export default function AdminConsole(): JSX.Element {
    // Defaults computed in the initializer so module-load time never matters:
    // from = first day of the current month, to = today.
    const [from, setFrom] = useState(() => {
        const d = new Date();
        return dateInput(new Date(d.getFullYear(), d.getMonth(), 1));
    });
    const [to, setTo] = useState(() => dateInput(new Date()));
    const [userFilter, setUserFilter] = useState('');
    const [rows, setRows] = useState<AdminRow[]>([]);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');

    // Date-input strings → epoch ms covering the full local days [from, to].
    const fromMs = new Date(from + 'T00:00:00').getTime();
    const toMs = new Date(to + 'T23:59:59.999').getTime();

    // Reload whenever the range changes (and on mount).
    useEffect(() => {
        let cancelled = false;
        const load = async () => {
            setLoading(true);
            setError('');
            try {
                const {entries, usernames} = await Client.adminEntries(fromMs, toMs);
                if (cancelled) {
                    return;
                }
                setRows(entries.map((e: TimeEntry) => ({
                    entry: e,
                    username: usernames[e.user_id] ?? e.user_id,
                })));
            } catch (e) {
                if (!cancelled) {
                    setError(e instanceof Error ? e.message : String(e));
                }
            } finally {
                if (!cancelled) {
                    setLoading(false);
                }
            }
        };
        load();
        return () => {
            cancelled = true;
        };

        // fromMs/toMs derive directly from from/to.
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [from, to]);

    // One clock read per render keeps still-running entries consistent across
    // the totals and the table.
    const now = Date.now();

    // Client-side username substring filter (case-insensitive).
    const filtered = useMemo(() => {
        const q = userFilter.trim().toLowerCase();
        if (!q) {
            return rows;
        }
        return rows.filter((r) => r.username.toLowerCase().includes(q));
    }, [rows, userFilter]);

    const totalHours = useMemo(
        () => decimalHours(filtered.reduce((sum, r) => sum + netSeconds(r.entry, now), 0)),
        [filtered, now],
    );
    const peopleCount = useMemo(
        () => new Set(filtered.map((r) => r.username)).size,
        [filtered],
    );
    const entriesCount = filtered.length;

    return (
        <div className='tt-admin'>
            <div className='tt-admin__inner'>
                <h2 className='tt-admin__title'>{t('adminTitle')}</h2>
                <p className='tt-admin__sub'>{t('adminDesc')}</p>

                <div className='tt-admin__bar'>
                    <div className='tt-admin__field'>
                        <label>{t('from')}</label>
                        <input
                            type='date'
                            className='tt-field'
                            value={from}
                            onChange={(e) => setFrom(e.target.value)}
                        />
                    </div>
                    <div className='tt-admin__field'>
                        <label>{t('to')}</label>
                        <input
                            type='date'
                            className='tt-field'
                            value={to}
                            onChange={(e) => setTo(e.target.value)}
                        />
                    </div>
                    <div className='tt-admin__field'>
                        <label>{t('filterUser')}</label>
                        <input
                            type='text'
                            className='tt-field'
                            placeholder={t('allUsers')}
                            value={userFilter}
                            onChange={(e) => setUserFilter(e.target.value)}
                        />
                    </div>
                    <div className='tt-admin__spacer'/>
                    <button
                        className='btn btn-primary'
                        onClick={() => window.open(Client.adminExportUrl(fromMs, toMs), '_blank', 'noopener')}
                    >
                        {t('exportCsv')}
                    </button>
                </div>

                <div className='tt-admin__totals'>
                    <div className='tt-admin__stat'>
                        <b>{totalHours}</b>
                        <span>{t('totalHours')}</span>
                    </div>
                    <div className='tt-admin__stat'>
                        <b>{peopleCount}</b>
                        <span>{t('people')}</span>
                    </div>
                    <div className='tt-admin__stat'>
                        <b>{entriesCount}</b>
                        <span>{t('entriesCount')}</span>
                    </div>
                </div>

                {error && <div className='tt-banner tt-banner--err'>{error}</div>}

                {loading && rows.length === 0 ? (
                    <div className='tt-admin__empty'>{t('loading')}</div>
                ) : (
                    <AdminTable
                        rows={filtered}
                        now={now}
                    />
                )}
            </div>
        </div>
    );
}
