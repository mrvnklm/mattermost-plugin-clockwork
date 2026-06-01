// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import Client from 'client/Client';
import type {TimeEntry} from 'client/Client';
import {t} from 'i18n';
import React, {useCallback, useEffect, useMemo, useState} from 'react';
import {ensureStyles} from 'styles';
import {formatHMS, localClock, localDayKey, netSeconds, weekRange} from 'utils/time';

import EntryEditModal from 'components/EntryEditModal';
import {PauseIcon, PlayIcon, PlusIcon, StopIcon} from 'components/icons';
import WeeklyTimesheet from 'components/WeeklyTimesheet';

ensureStyles();

export default function RHSView(): JSX.Element {
    const [current, setCurrent] = useState<TimeEntry | null>(null);
    const [entries, setEntries] = useState<TimeEntry[]>([]);
    const [now, setNow] = useState(Date.now());
    const [project, setProject] = useState('');
    const [description, setDescription] = useState('');
    const [error, setError] = useState('');
    const [busy, setBusy] = useState(false);
    const [editing, setEditing] = useState<{entry: TimeEntry | null} | null>(null);
    const [suggest, setSuggest] = useState<{projects: string[]; notes: string[]}>({projects: [], notes: []});

    const refetch = useCallback(async () => {
        setError('');
        try {
            const {from, to} = weekRange(new Date());
            const [cur, list, sug] = await Promise.all([
                Client.getCurrent(),
                Client.listEntries(from, to),
                Client.suggestions(),
            ]);
            setCurrent(cur.entry);
            setEntries(list.entries);
            setSuggest(sug);
        } catch (e) {
            setError(e instanceof Error ? e.message : String(e));
        }
    }, []);

    useEffect(() => {
        refetch();
    }, [refetch]);

    // Tick the on-screen clock once per second while a timer is running.
    useEffect(() => {
        if (!current || current.end_at !== 0) {
            return undefined;
        }
        setNow(Date.now());
        const id = setInterval(() => setNow(Date.now()), 1000);
        return () => clearInterval(id);
    }, [current]);

    const run = useCallback(async (action: () => Promise<unknown>) => {
        setBusy(true);
        setError('');
        try {
            await action();
            await refetch();
        } catch (e) {
            setError(e instanceof Error ? e.message : String(e));
        } finally {
            setBusy(false);
        }
    }, [refetch]);

    const onStart = () => run(async () => {
        await Client.start(project.trim(), description.trim());
        setProject('');
        setDescription('');
    });
    const onStop = () => run(() => Client.stop());
    const onBreakToggle = () => run(() =>
        (current && current.break_started_at ? Client.breakStop() : Client.breakStart()));

    const todayKey = localDayKey(now);

    // Today's closed entries (the running one is shown in the hero above).
    const todayEntries = useMemo(
        () => entries.filter((e) => e.end_at !== 0 && localDayKey(e.start_at) === todayKey),
        [entries, todayKey],
    );

    const onBreak = Boolean(current && current.break_started_at);
    const elapsed = current ? netSeconds(current, now) : 0;

    return (
        <div className='tt'>
            {/* Autocomplete sources shared by the start form and the edit modal. */}
            <datalist id='clockwork-projects'>
                {suggest.projects.map((v) => (
                    <option
                        key={v}
                        value={v}
                    />
                ))}
            </datalist>
            <datalist id='clockwork-notes'>
                {suggest.notes.map((v) => (
                    <option
                        key={v}
                        value={v}
                    />
                ))}
            </datalist>

            {error && <div className='tt__err'>{error}</div>}

            {/* Hero: live timer or start controls */}
            <div className='tt__section'>
                {current ? (
                    <div className={`tt-hero ${onBreak ? '' : 'tt-hero--run'}`}>
                        <div className='tt-hero__state'>
                            <span className={`tt-dot ${onBreak ? '' : 'tt-dot--live'}`}/>
                            {onBreak ? t('onBreak') : t('running')}
                        </div>
                        <div className='tt-time'>{formatHMS(elapsed)}</div>
                        <div className='tt-hero__meta'>
                            {current.project && <span className='tt-hero__proj'>{current.project}</span>}
                            {current.project && current.description ? ' · ' : ''}
                            {current.description}
                            {!current.project && !current.description ? t('noProject') : ''}
                        </div>
                        <div className='tt-actions'>
                            <button
                                className='btn btn-primary'
                                disabled={busy}
                                onClick={onStop}
                            >
                                <StopIcon/>{' '}{t('stop')}
                            </button>
                            <button
                                className='btn btn-tertiary'
                                disabled={busy}
                                onClick={onBreakToggle}
                            >
                                {onBreak ? <PlayIcon/> : <PauseIcon/>}{' '}
                                {onBreak ? t('breakStop') : t('breakStart')}
                            </button>
                        </div>
                    </div>
                ) : (
                    <div>
                        <div className='tt__label'>{t('idle')}</div>
                        <input
                            type='text'
                            className='tt-field'
                            list='clockwork-projects'
                            placeholder={t('project')}
                            value={project}
                            disabled={busy}
                            onChange={(e) => setProject(e.target.value)}
                        />
                        <input
                            type='text'
                            className='tt-field'
                            list='clockwork-notes'
                            placeholder={t('note')}
                            value={description}
                            disabled={busy}
                            onChange={(e) => setDescription(e.target.value)}
                            onKeyDown={(e) => e.key === 'Enter' && !busy && onStart()}
                        />
                        <button
                            className='btn btn-primary tt-start'
                            disabled={busy}
                            onClick={onStart}
                        >
                            <PlayIcon/>{' '}{t('start')}
                        </button>
                    </div>
                )}
            </div>

            {/* Today */}
            <div className='tt__section'>
                <div className='tt__label'>{t('today')}</div>
                {todayEntries.length === 0 ? (
                    <div className='tt-empty'>{t('noEntries')}</div>
                ) : (
                    <div className='tt-list'>
                        {todayEntries.map((e) => (
                            <button
                                key={e.id}
                                className='tt-row tt-row--btn'
                                onClick={() => setEditing({entry: e})}
                                title={t('edit')}
                            >
                                <div className='tt-row__main'>
                                    <div className='tt-row__top'>
                                        <span>{localClock(e.start_at)}{' – '}{localClock(e.end_at)}</span>
                                        {e.project && <span className='tt-chip'>{e.project}</span>}
                                    </div>
                                    {e.description && <div className='tt-note'>{e.description}</div>}
                                </div>
                                <span className='tt-dur'>{formatHMS(netSeconds(e, now))}</span>
                            </button>
                        ))}
                    </div>
                )}
            </div>

            {/* Week */}
            <div className='tt__section'>
                <div className='tt__label'>
                    {t('week')}
                    <button
                        className='tt-link'
                        onClick={() => setEditing({entry: null})}
                    >
                        <PlusIcon/>{' '}{t('add')}
                    </button>
                </div>
                <WeeklyTimesheet
                    entries={entries}
                    now={now}
                    onSelect={(entry) => setEditing({entry})}
                />
            </div>

            {editing && (
                <EntryEditModal
                    entry={editing.entry}
                    onClose={() => setEditing(null)}
                    onSaved={refetch}
                />
            )}
        </div>
    );
}
