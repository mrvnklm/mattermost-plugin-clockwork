// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import Client from 'client/Client';
import type {EntryStatus, TimeEntry} from 'client/Client';
import {t} from 'i18n';
import React, {useCallback, useEffect, useMemo, useState} from 'react';
import {ensureStyles} from 'styles';
import {dateInputValue, decimalHours, fromDatetimeInput, netSeconds} from 'utils/time';
import {useStableId} from 'utils/useStableId';

import AdminProjectSummary from 'components/admin/AdminProjectSummary';
import type {AdminRow} from 'components/admin/AdminTable';
import AdminTable from 'components/admin/AdminTable';

ensureStyles();

type StatusFilter = 'all' | EntryStatus;

// PersonalReport is the full-page time report scoped to the current user. It is
// the default view of the Clockwork product and is available to every user — it
// reuses the same presentation as the admin team report (totals, per-project
// breakdown, entries table) but without any cross-user or approval-admin
// controls. All data comes from the caller's own endpoints (/entries,
// /reports/export, /timesheet/submit|withdraw).
export default function PersonalReport(): JSX.Element {
    // Default range: first day of the current month → today.
    const [from, setFrom] = useState(() => {
        const d = new Date();
        return dateInputValue(new Date(d.getFullYear(), d.getMonth(), 1));
    });
    const [to, setTo] = useState(() => dateInputValue(new Date()));

    const [statusFilter, setStatusFilter] = useState<StatusFilter>('all');
    const [rows, setRows] = useState<AdminRow[]>([]);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');
    const [actionBusy, setActionBusy] = useState(false);
    const [approvalEnabled, setApprovalEnabled] = useState(false);

    const ids = useStableId('cw-me');
    const fromId = `${ids}-from`;
    const toId = `${ids}-to`;
    const statusId = `${ids}-status`;

    const fromMs = fromDatetimeInput(from + 'T00:00');
    const toMs = fromDatetimeInput(to + 'T23:59') + 59_999;

    // Config is immutable for the session; fetch once.
    useEffect(() => {
        let cancelled = false;
        Client.config().
            then((c) => {
                if (!cancelled) {
                    setApprovalEnabled(Boolean(c.approval_enabled));
                }
            }).
            catch(() => {
                // approval defaults to off; ignore
            });
        return () => {
            cancelled = true;
        };
    }, []);

    const reload = useCallback(async (signal?: {cancelled: boolean}) => {
        setLoading(true);
        setError('');
        try {
            const {entries} = await Client.listEntries(fromMs, toMs);
            if (signal?.cancelled) {
                return;
            }
            setRows(entries.map((e: TimeEntry) => ({entry: e, username: ''})));
        } catch (e) {
            if (!signal?.cancelled) {
                setError(e instanceof Error ? e.message : String(e));
            }
        } finally {
            if (!signal?.cancelled) {
                setLoading(false);
            }
        }

        // fromMs/toMs derive from from/to.
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [from, to]);

    useEffect(() => {
        const signal = {cancelled: false};
        reload(signal);
        return () => {
            signal.cancelled = true;
        };
    }, [reload]);

    const now = Date.now();

    const filtered = useMemo(() => {
        if (statusFilter === 'all') {
            return rows;
        }
        return rows.filter((r) => r.entry.status === statusFilter);
    }, [rows, statusFilter]);

    const totalHours = useMemo(
        () => decimalHours(filtered.reduce((sum, r) => sum + netSeconds(r.entry, now), 0)),
        [filtered, now],
    );
    const entriesCount = filtered.length;

    // Counts that drive which workflow action is offered for the selected range.
    const openCount = useMemo(() => rows.filter((r) => r.entry.status === 'open' && r.entry.end_at !== 0).length, [rows]);
    const submittedCount = useMemo(() => rows.filter((r) => r.entry.status === 'submitted').length, [rows]);

    const onExport = () => {
        const win = window.open(Client.exportUrl(fromMs, toMs), '_blank', 'noopener');
        if (!win) {
            setError(t('exportFailed'));
        }
    };

    const runAction = async (fn: () => Promise<{updated: number}>) => {
        setActionBusy(true);
        setError('');
        try {
            await fn();
            await reload();
        } catch (e) {
            setError(e instanceof Error ? e.message : String(e));
        } finally {
            setActionBusy(false);
        }
    };
    const onSubmit = () => runAction(() => Client.submitTimesheet(fromMs, toMs));
    const onWithdraw = () => runAction(() => Client.withdrawTimesheet(fromMs, toMs));

    return (
        <div className='tt-admin'>
            <div className='tt-admin__inner'>
                <h2 className='tt-admin__title'>{t('myReportTitle')}</h2>
                <p className='tt-admin__sub'>{t('myReportDesc')}</p>

                <div className='tt-admin__bar'>
                    <div className='tt-admin__field'>
                        <label htmlFor={fromId}>{t('from')}</label>
                        <input
                            id={fromId}
                            type='date'
                            className='tt-field'
                            value={from}
                            onChange={(e) => setFrom(e.target.value)}
                        />
                    </div>
                    <div className='tt-admin__field'>
                        <label htmlFor={toId}>{t('to')}</label>
                        <input
                            id={toId}
                            type='date'
                            className='tt-field'
                            value={to}
                            onChange={(e) => setTo(e.target.value)}
                        />
                    </div>
                    {approvalEnabled && (
                        <div className='tt-admin__field'>
                            <label htmlFor={statusId}>{t('filterStatus')}</label>
                            <select
                                id={statusId}
                                className='tt-field'
                                value={statusFilter}
                                onChange={(e) => setStatusFilter(e.target.value as StatusFilter)}
                            >
                                <option value='all'>{t('allStatuses')}</option>
                                <option value='open'>{t('statusOpen')}</option>
                                <option value='submitted'>{t('statusSubmitted')}</option>
                                <option value='approved'>{t('statusApproved')}</option>
                            </select>
                        </div>
                    )}
                    <div className='tt-admin__spacer'/>
                    {approvalEnabled && openCount > 0 && (
                        <button
                            className='tt-btn-sm'
                            disabled={actionBusy}
                            onClick={onSubmit}
                        >
                            {t('submitRange')}
                        </button>
                    )}
                    {approvalEnabled && submittedCount > 0 && (
                        <button
                            className='tt-btn-sm'
                            disabled={actionBusy}
                            onClick={onWithdraw}
                        >
                            {t('withdraw')}
                        </button>
                    )}
                    <button
                        className='btn btn-primary'
                        onClick={onExport}
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
                        <b>{entriesCount}</b>
                        <span>{t('entriesCount')}</span>
                    </div>
                </div>

                {error && <div className='tt-banner tt-banner--err'>{error}</div>}

                {loading && rows.length === 0 ? (
                    <div className='tt-admin__empty'>{t('loading')}</div>
                ) : (
                    <>
                        <h3 className='tt-admin__h3'>{t('byProject')}</h3>
                        <AdminProjectSummary
                            rows={filtered}
                            now={now}
                        />
                        <AdminTable
                            rows={filtered}
                            now={now}
                            approvalEnabled={approvalEnabled}
                            showUser={false}
                        />
                    </>
                )}
            </div>
        </div>
    );
}
