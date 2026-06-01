// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import Client from 'client/Client';
import type {TimeEntry} from 'client/Client';
import {t} from 'i18n';
import React, {useState} from 'react';
import {fromDatetimeInput, toDatetimeInput} from 'utils/time';

type Props = {

    // When editing, the existing entry; when adding, null.
    entry: TimeEntry | null;
    onClose: () => void;
    onSaved: () => void;
};

export default function EntryEditModal({entry, onClose, onSaved}: Props): JSX.Element {
    const isEdit = Boolean(entry);
    const locked = Boolean(entry?.locked);

    const [start, setStart] = useState(toDatetimeInput(entry ? entry.start_at : Date.now()));
    const [end, setEnd] = useState(toDatetimeInput(entry ? entry.end_at : 0));
    const [breakMin, setBreakMin] = useState(entry ? Math.round(entry.break_seconds / 60) : 0);
    const [project, setProject] = useState(entry?.project ?? '');
    const [description, setDescription] = useState(entry?.description ?? '');
    const [error, setError] = useState('');
    const [busy, setBusy] = useState(false);

    const disabled = locked || busy;

    const submit = async () => {
        setError('');
        const startMs = fromDatetimeInput(start);
        const endMs = fromDatetimeInput(end);
        if (!startMs) {
            setError(`${t('startTime')} ?`);
            return;
        }
        if (!endMs) {
            setError(t('endRequired'));
            return;
        }
        if (endMs < startMs) {
            setError(t('invalidRange'));
            return;
        }
        const mins = Number.isFinite(breakMin) ? Math.max(0, Math.round(breakMin)) : 0;
        const body = {
            start_at: startMs,
            end_at: endMs,
            break_seconds: mins * 60,
            project,
            description,
        };
        setBusy(true);
        try {
            if (isEdit && entry) {
                await Client.updateEntry(entry.id, body);
            } else {
                await Client.createEntry(body);
            }
            onSaved();
            onClose();
        } catch (e) {
            setError(e instanceof Error ? e.message : String(e));
        } finally {
            setBusy(false);
        }
    };

    const remove = async () => {
        if (!entry) {
            return;
        }
        setBusy(true);
        setError('');
        try {
            await Client.deleteEntry(entry.id);
            onSaved();
            onClose();
        } catch (e) {
            setError(e instanceof Error ? e.message : String(e));
        } finally {
            setBusy(false);
        }
    };

    return (
        <div
            className='tt-overlay'
            onClick={onClose}
        >
            <div
                className='tt-modal'
                onClick={(e) => e.stopPropagation()}
            >
                <h3 className='tt-modal__title'>{isEdit ? t('editEntry') : t('newEntry')}</h3>

                {locked && <div className='tt-banner tt-banner--warn'>{t('locked')}</div>}

                <div className='tt-modal__row'>
                    <div>
                        <label className='tt-modal__label'>{t('startTime')}</label>
                        <input
                            type='datetime-local'
                            className='tt-field'
                            value={start}
                            disabled={disabled}
                            onChange={(e) => setStart(e.target.value)}
                        />
                    </div>
                    <div>
                        <label className='tt-modal__label'>{t('endTime')}</label>
                        <input
                            type='datetime-local'
                            className='tt-field'
                            value={end}
                            disabled={disabled}
                            onChange={(e) => setEnd(e.target.value)}
                        />
                    </div>
                </div>

                <label className='tt-modal__label'>{t('breakMinutes')}</label>
                <input
                    type='number'
                    min={0}
                    className='tt-field'
                    value={Number.isFinite(breakMin) ? breakMin : 0}
                    disabled={disabled}
                    onChange={(e) => {
                        const v = e.target.value === '' ? 0 : Number(e.target.value);
                        setBreakMin(Number.isNaN(v) ? 0 : v);
                    }}
                />

                <label className='tt-modal__label'>{t('project')}</label>
                <input
                    type='text'
                    className='tt-field'
                    list='clockwork-projects'
                    value={project}
                    disabled={disabled}
                    onChange={(e) => setProject(e.target.value)}
                />

                <label className='tt-modal__label'>{t('note')}</label>
                <input
                    type='text'
                    className='tt-field'
                    list='clockwork-notes'
                    value={description}
                    disabled={disabled}
                    onChange={(e) => setDescription(e.target.value)}
                />

                {error && <div className='tt-banner tt-banner--err'>{error}</div>}

                <div className='tt-modal__foot'>
                    <div>
                        {isEdit && !locked && (
                            <button
                                className='btn btn-tertiary'
                                style={{color: 'var(--error-text, #d24b4e)'}}
                                disabled={busy}
                                onClick={remove}
                            >
                                {t('delete')}
                            </button>
                        )}
                    </div>
                    <div style={{display: 'flex', gap: 8}}>
                        <button
                            className='btn btn-tertiary'
                            disabled={busy}
                            onClick={onClose}
                        >
                            {t('cancel')}
                        </button>
                        <button
                            className='btn btn-primary'
                            disabled={disabled}
                            onClick={submit}
                        >
                            {t('save')}
                        </button>
                    </div>
                </div>
            </div>
        </div>
    );
}
