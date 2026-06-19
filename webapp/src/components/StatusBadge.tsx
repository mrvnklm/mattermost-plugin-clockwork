// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import type {EntryStatus} from 'client/Client';
import {t} from 'i18n';
import type {StringKey} from 'i18n';
import React from 'react';

const LABEL_KEY: Record<EntryStatus, StringKey> = {
    open: 'statusOpen',
    submitted: 'statusSubmitted',
    approved: 'statusApproved',
};

// statusLabel returns the localized label for a status value (falls back to the
// raw value for forward-compatibility with unknown server values).
export function statusLabel(status: EntryStatus): string {
    const key = LABEL_KEY[status];
    return key ? t(key) : status;
}

// StatusBadge renders a colored pill for an entry's approval status.
export default function StatusBadge({status}: {status: EntryStatus}): JSX.Element {
    return (
        <span className={`tt-badge tt-badge--${status}`}>
            {statusLabel(status)}
        </span>
    );
}
