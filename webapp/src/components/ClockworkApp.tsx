// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {t} from 'i18n';
import React, {useState} from 'react';
import {getIsAdmin} from 'session';
import {ensureStyles} from 'styles';

import AdminConsole from 'components/admin/AdminConsole';
import PersonalReport from 'components/PersonalReport';

ensureStyles();

type Mode = 'personal' | 'team';

// ClockworkApp is the full-page component behind the Clockwork product (the
// switcher entry). Every user gets the personal report; system admins get a
// toggle to the cross-user team report. The server re-authorizes the admin
// endpoints regardless, so the toggle is a convenience, not a security boundary.
export default function ClockworkApp(): JSX.Element {
    const admin = getIsAdmin();
    const [mode, setMode] = useState<Mode>('personal');

    if (!admin) {
        return <PersonalReport/>;
    }

    return (
        <div className='tt-product'>
            <div className='tt-product__tabs'>
                <button
                    className={'tt-tab' + (mode === 'personal' ? ' is-active' : '')}
                    onClick={() => setMode('personal')}
                >
                    {t('myTime')}
                </button>
                <button
                    className={'tt-tab' + (mode === 'team' ? ' is-active' : '')}
                    onClick={() => setMode('team')}
                >
                    {t('teamReport')}
                </button>
            </div>
            {mode === 'team' ? <AdminConsole/> : <PersonalReport/>}
        </div>
    );
}
