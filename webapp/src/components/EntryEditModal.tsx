// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import Client from 'client/Client';
import type {TimeEntry} from 'client/Client';
import {t} from 'i18n';
import React, {useCallback, useEffect, useRef, useState} from 'react';
import {fromDatetimeInput, toDatetimeInput} from 'utils/time';
import {useStableId} from 'utils/useStableId';

type Props = {

    // When editing, the existing entry; when adding, null.
    entry: TimeEntry | null;
    onClose: () => void;
    onSaved: () => void;
};

// Selector for the focusable elements inside the dialog (used by the focus trap).
const FOCUSABLE = 'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])';

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
    const [confirmingDelete, setConfirmingDelete] = useState(false);

    const disabled = locked || busy;

    // Unique ids tie each <label htmlFor> to its control and the dialog title to
    // aria-labelledby. useId keeps multiple instances distinct.
    const baseId = useStableId('cw-entry');
    const titleId = `${baseId}-title`;
    const ids = {
        start: `${baseId}-start`,
        end: `${baseId}-end`,
        break: `${baseId}-break`,
        project: `${baseId}-project`,
        note: `${baseId}-note`,
    };

    const dialogRef = useRef<HTMLDivElement>(null);
    const firstFieldRef = useRef<HTMLInputElement>(null);

    // Move focus into the dialog on open so screen-reader/keyboard users land
    // inside it (WCAG 2.4.3). The first field when editable; otherwise the first
    // focusable control (e.g. the Cancel button) for a locked/read-only entry,
    // whose inputs are disabled and thus not focusable.
    useEffect(() => {
        if (firstFieldRef.current && !disabled) {
            firstFieldRef.current.focus();
            return;
        }
        const first = dialogRef.current?.querySelector<HTMLElement>(FOCUSABLE);
        first?.focus();

        // Only on mount.
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    // Escape closes; Tab is trapped within the dialog.
    const onKeyDown = useCallback((e: React.KeyboardEvent) => {
        if (e.key === 'Escape') {
            e.stopPropagation();
            onClose();
            return;
        }
        if (e.key !== 'Tab' || !dialogRef.current) {
            return;
        }
        const nodes = dialogRef.current.querySelectorAll<HTMLElement>(FOCUSABLE);
        if (nodes.length === 0) {
            return;
        }
        const first = nodes[0];
        const last = nodes[nodes.length - 1];
        const active = document.activeElement;
        if (e.shiftKey && active === first) {
            e.preventDefault();
            last.focus();
        } else if (!e.shiftKey && active === last) {
            e.preventDefault();
            first.focus();
        }
    }, [onClose]);

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
            setConfirmingDelete(false);
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
                ref={dialogRef}
                className='tt-modal'
                role='dialog'
                aria-modal={true}
                aria-labelledby={titleId}
                onClick={(e) => e.stopPropagation()}
                onKeyDown={onKeyDown}
            >
                <h3
                    className='tt-modal__title'
                    id={titleId}
                >
                    {isEdit ? t('editEntry') : t('newEntry')}
                </h3>

                {locked && <div className='tt-banner tt-banner--warn'>{t('locked')}</div>}

                <div className='tt-modal__row'>
                    <div>
                        <label
                            className='tt-modal__label'
                            htmlFor={ids.start}
                        >
                            {t('startTime')}
                        </label>
                        <input
                            ref={firstFieldRef}
                            id={ids.start}
                            type='datetime-local'
                            className='tt-field'
                            value={start}
                            disabled={disabled}
                            onChange={(e) => setStart(e.target.value)}
                        />
                    </div>
                    <div>
                        <label
                            className='tt-modal__label'
                            htmlFor={ids.end}
                        >
                            {t('endTime')}
                        </label>
                        <input
                            id={ids.end}
                            type='datetime-local'
                            className='tt-field'
                            value={end}
                            disabled={disabled}
                            onChange={(e) => setEnd(e.target.value)}
                        />
                    </div>
                </div>

                <label
                    className='tt-modal__label'
                    htmlFor={ids.break}
                >
                    {t('breakMinutes')}
                </label>
                <input
                    id={ids.break}
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

                <label
                    className='tt-modal__label'
                    htmlFor={ids.project}
                >
                    {t('project')}
                </label>
                <input
                    id={ids.project}
                    type='text'
                    className='tt-field'
                    list='clockwork-projects'
                    value={project}
                    disabled={disabled}
                    onChange={(e) => setProject(e.target.value)}
                />

                <label
                    className='tt-modal__label'
                    htmlFor={ids.note}
                >
                    {t('note')}
                </label>
                <input
                    id={ids.note}
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
                            confirmingDelete ? (
                                <span className='tt-actbtns'>
                                    <span
                                        className='tt-modal__label'
                                        style={{margin: 0}}
                                    >{t('confirmDelete')}</span>
                                    <button
                                        className='tt-btn-sm'
                                        style={{color: 'var(--error-text, #d24b4e)'}}
                                        disabled={busy}
                                        onClick={remove}
                                    >
                                        {t('delete')}
                                    </button>
                                    <button
                                        className='tt-btn-sm'
                                        disabled={busy}
                                        onClick={() => setConfirmingDelete(false)}
                                    >
                                        {t('cancel')}
                                    </button>
                                </span>
                            ) : (
                                <button
                                    className='btn btn-tertiary'
                                    style={{color: 'var(--error-text, #d24b4e)'}}
                                    disabled={busy}
                                    onClick={() => setConfirmingDelete(true)}
                                >
                                    {t('delete')}
                                </button>
                            )
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
