// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {setLocale, t} from 'i18n';
import manifest from 'manifest';
import React from 'react';
import type {Store} from 'redux';
import {setIsAdmin} from 'session';
import {setTimezone} from 'utils/time';

import type {GlobalState} from '@mattermost/types/store';

import ClockworkApp from 'components/ClockworkApp';
import {ClockIcon} from 'components/icons';
import RHSView from 'components/rhs/RHSView';

import type {PluginRegistry} from 'types/mattermost-webapp';

// Header rendered in the global header bar while the Clockwork product is active.
const ClockworkProductHeader = (): JSX.Element => (
    <span className='tt-product__title'>{t('title')}</span>
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

        const {showRHSPlugin} = registry.registerRightHandSidebarComponent(RHSView, t('rhsTitle'));

        registry.registerChannelHeaderButtonAction(
            <ClockIcon size={20}/>,
            () => store.dispatch(showRHSPlugin),
            t('rhsTitle'),
            t('channelHeaderTooltip'),
        );

        // Capture whether the user is a system admin so the product can offer the
        // team-report toggle. The server still authorizes every admin request.
        let isAdmin = false;
        try {
            const state: any = store.getState();
            const userId = state?.entities?.users?.currentUserId;
            const roles = userId && state?.entities?.users?.profiles?.[userId]?.roles;
            isAdmin = Boolean(roles && roles.split(' ').includes('system_admin'));
        } catch {
            isAdmin = false;
        }
        setIsAdmin(isAdmin);

        // Register Clockwork as a full product: it appears in the product switcher
        // next to Channels/Playbooks/Boards and opens a full-page time report.
        // Every user gets their own report; admins get a team-report toggle inside.
        if (registry.registerProduct) {
            registry.registerProduct({
                baseURL: '/clockwork',
                switcherIcon: 'clock-outline',
                switcherText: t('title'),
                switcherLinkURL: '/clockwork',
                mainComponent: ClockworkApp,
                headerCentreComponent: ClockworkProductHeader,
                showTeamSidebar: false,
            });
        }
    }
}

declare global {
    interface Window {
        registerPlugin(pluginId: string, plugin: Plugin): void;
    }
}

window.registerPlugin(manifest.id, new Plugin());
