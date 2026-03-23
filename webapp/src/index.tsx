// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {pluginFetch} from 'client';
import manifest from 'manifest';
import React from 'react';
import type {Store} from 'redux';
import {callsReducer, setPluginEnabled} from 'redux/calls_slice';
import {
    handleCallEnded,
    handleCallStarted,
    handleNotifDismissed,
    handleUserJoined,
    handleUserLeft,
} from 'redux/websocket_handlers';

import type {GlobalState} from '@mattermost/types/store';

import CallPost from 'components/call_post';
import ChannelHeaderButton from 'components/channel_header_button';
import FloatingWidget from 'components/floating_widget';
import IncomingCallNotification from 'components/incoming_call_notification';
import ToastBar from 'components/toast_bar';

import type {PluginRegistry} from 'types/mattermost-webapp';

import enTranslations from '../i18n/en.json';
import jaTranslations from '../i18n/ja.json';

interface ConfigStatusResponse {
    enabled: boolean;
}

async function fetchConfigStatus(store: Store<GlobalState>) {
    const result = await pluginFetch<ConfigStatusResponse>('/api/v1/config/status');
    if ('data' in result) {
        store.dispatch(setPluginEnabled(result.data.enabled));
    } else {
        // Default to disabled on error (BR-001, REL-U3-03)
        store.dispatch(setPluginEnabled(false));
    }
}

export default class Plugin {
    public async initialize(registry: PluginRegistry, store: Store<GlobalState>) {
        // 1. Register Redux reducer
        registry.registerReducer(callsReducer);

        // 2. Fetch initial config status
        await fetchConfigStatus(store);

        // 3. Get current user ID for WS handlers
        const state = store.getState();
        const currentUserId = (state as unknown as {entities: {users: {currentUserId: string}}}).
            entities?.users?.currentUserId ?? '';

        // 4. Register WebSocket event handlers
        registry.registerWebSocketEventHandler(
            `custom_${manifest.id}_call_started`,
            handleCallStarted(store as unknown as Store<GlobalState>, currentUserId),
        );
        registry.registerWebSocketEventHandler(
            `custom_${manifest.id}_user_joined`,
            handleUserJoined(store as unknown as Store<GlobalState>, currentUserId),
        );
        registry.registerWebSocketEventHandler(
            `custom_${manifest.id}_user_left`,
            handleUserLeft(store as unknown as Store<GlobalState>, currentUserId),
        );
        registry.registerWebSocketEventHandler(
            `custom_${manifest.id}_call_ended`,
            handleCallEnded(store as unknown as Store<GlobalState>, currentUserId),
        );
        registry.registerWebSocketEventHandler(
            `custom_${manifest.id}_notification_dismissed`,
            handleNotifDismissed(store as unknown as Store<GlobalState>, currentUserId),
        );

        // 5. Re-fetch config on WebSocket reconnect (BR-014)
        registry.registerReconnectHandler(async () => {
            await fetchConfigStatus(store);
        });

        // 6. Register UI components
        registry.registerCallButtonAction(

            // button: component that reads Redux and renders the appropriate state
            () => {
                const channelId = (store.getState() as unknown as {
                    entities: {channels: {currentChannelId: string}};
                }).entities?.channels?.currentChannelId ?? '';

                return (
                    <ChannelHeaderButton
                        channel={{id: channelId} as never}
                        currentUserId={currentUserId}
                    />
                );
            },

            // dropdownButton: same component for dropdown context
            () => null,

            // fn: click handler (logic is inside ChannelHeaderButton itself)
            () => { /* no-op — click handled inside ChannelHeaderButton */ },
        );

        registry.registerRootComponent(() => {
            const channelId = (store.getState() as unknown as {
                entities: {channels: {currentChannelId: string}};
            }).entities?.channels?.currentChannelId ?? '';
            return (
                <ToastBar
                    currentChannelId={channelId}
                    currentUserId={currentUserId}
                />
            );
        });

        registry.registerGlobalComponent(() => (
            <FloatingWidget/>
        ));

        registry.registerGlobalComponent(() => (
            <IncomingCallNotification currentUserId={currentUserId}/>
        ));

        // 7. Register custom post type renderer
        registry.registerPostTypeComponent(
            'custom_cf_call',
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            CallPost as any,
        );

        // 8. Register translations (i18n)
        registry.registerTranslations((locale: string) => {
            if (locale === 'ja') {
                return jaTranslations;
            }
            return enTranslations;
        });
    }
}

declare global {
    interface Window {
        registerPlugin(pluginId: string, plugin: Plugin): void;
    }
}

window.registerPlugin(manifest.id, new Plugin());
