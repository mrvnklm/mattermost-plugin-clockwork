// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {setLocale} from 'i18n';
import manifest from 'manifest';
import React from 'react';
import type {Store} from 'redux';
import {setTimezone} from 'utils/time';

import type {GlobalState} from '@mattermost/types/store';

import AdminConsole from 'components/admin/AdminConsole';
import RHSView from 'components/rhs/RHSView';

import type {PluginRegistry} from 'types/mattermost-webapp';

// Simple inline clock icon for the channel header button.
const ClockIcon = () => (
    <svg
        width='20'
        height='20'
        viewBox='0 0 24 24'
        fill='none'
        stroke='currentColor'
        strokeWidth='2'
        strokeLinecap='round'
        strokeLinejoin='round'
        aria-hidden='true'
    >
        <circle
            cx='12'
            cy='12'
            r='9'
        />
        <polyline points='12 7 12 12 16 14'/>
    </svg>
);

export default class Plugin {
    public async initialize(registry: PluginRegistry, store: Store<GlobalState>) {
        // Pick locale and timezone from the current user so day-grouping and
        // labels match the user's Mattermost settings (default 'de'/local tz).
        try {
            const state: any = store.getState();
            const userId = state?.entities?.users?.currentUserId;
            const profile = userId && state?.entities?.users?.profiles?.[userId];
            setLocale(profile?.locale);

            const tz = profile?.timezone;
            if (tz) {
                const useAuto = tz.useAutomaticTimezone === true || tz.useAutomaticTimezone === 'true';
                setTimezone(useAuto ? tz.automaticTimezone : tz.manualTimezone);
            }
        } catch {
            // ignore — i18n defaults to German, timezone to browser-local
        }

        const {showRHSPlugin} = registry.registerRightHandSidebarComponent(RHSView, 'Time Tracking');

        registry.registerChannelHeaderButtonAction(
            <ClockIcon/>,
            () => store.dispatch(showRHSPlugin),
            'Time Tracking',
            'Track work time',
        );

        // Dedicated full-page team report at /<team>/<plugin-id>/admin. The route
        // is always registered (the server authorizes every request); the menu
        // entry that links to it is only added for system admins.
        registry.registerNeedsTeamRoute('/admin', AdminConsole);

        let isAdmin = false;
        try {
            const state: any = store.getState();
            const userId = state?.entities?.users?.currentUserId;
            const roles = userId && state?.entities?.users?.profiles?.[userId]?.roles;
            isAdmin = Boolean(roles && roles.split(' ').includes('system_admin'));
        } catch {
            isAdmin = false;
        }

        if (isAdmin && registry.registerMainMenuAction) {
            registry.registerMainMenuAction(
                'Time Tracking — Team report',
                () => {
                    const s: any = store.getState();
                    const teamId = s?.entities?.teams?.currentTeamId;
                    const team = teamId && s?.entities?.teams?.teams?.[teamId];
                    if (team) {
                        window.location.assign(`/${team.name}/${manifest.id}/admin`);
                    }
                },
                <ClockIcon/>,
            );
        }
    }
}

declare global {
    interface Window {
        registerPlugin(pluginId: string, plugin: Plugin): void;
    }
}

window.registerPlugin(manifest.id, new Plugin());
