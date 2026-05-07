// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import './styles.css';

import {pluginFetch} from 'client';
import manifest from 'manifest';
import React from 'react';
import type {Store} from 'redux';
import {
    callsReducer,
    setCallError,
    setCallLoading,
    setChannelEnabled,
    setChannelEnabledBulk,
    setMyActiveCall,
    setPendingSwitchCallId,
    setPluginEnabled,
    upsertCall,
} from 'redux/calls_slice';
import {
    handleCallEnded,
    handleCallStarted,
    handleNotifDismissed,
    handleUserJoined,
    handleUserLeft,
} from 'redux/websocket_handlers';
import {
    selectCallByChannel,
    selectChannelEnabled,
    selectIsCurrentUserParticipant,
    selectMyActiveCall,
} from 'redux/selectors';
import {getCallErrorMessage} from 'utils/error_messages';
import {playJoinSound} from 'utils/sounds';

import type {GlobalState} from '@mattermost/types/store';
import type {Channel} from '@mattermost/types/channels';

import CallActionsRoot from 'components/call_actions_root';
import EnvVarCredentialSetting from 'components/admin_config/EnvVarCredentialSetting';
import CallPost from 'components/call_post';
import ChannelHeaderButton from 'components/channel_header_button';
import ChannelHeaderDropdownButton from 'components/channel_header_button/DropdownButton';
import ChannelMenuAction from 'components/channel_menu_action';
import FloatingWidget from 'components/floating_widget';
import IncomingCallNotification from 'components/incoming_call_notification';
import ToastBar from 'components/toast_bar';

import type {PluginRegistry} from 'types/mattermost-webapp';

import enTranslations from '../i18n/en.json';
import jaTranslations from '../i18n/ja.json';

interface ConfigStatusResponse {
    enabled: boolean;
}

async function fetchAllChannelEnabled(store: Store<GlobalState>) {
    const result = await pluginFetch<Array<{channel_id: string; enabled?: boolean}>>(
        '/api/v1/channels',
    );
    if ('data' in result) {
        store.dispatch(setChannelEnabledBulk(result.data));
    }
}

async function fetchConfigStatus(store: Store<GlobalState>) {
    const result = await pluginFetch<ConfigStatusResponse>('/api/v1/config/status');
    if ('data' in result) {
        store.dispatch(setPluginEnabled(result.data.enabled));
    } else {
        // Default to disabled on error
        store.dispatch(setPluginEnabled(false));
    }
}

export default class Plugin {
    public async initialize(registry: PluginRegistry, store: Store<GlobalState>) {
        // 1. Register translations first — before any async work or UI component registrations.
        // This ensures i18n messages are always available when post-type components render,
        // even if the page loads while fetchConfigStatus is still in-flight.
        registry.registerTranslations((locale: string) => {
            if (locale === 'ja') {
                return jaTranslations;
            }
            return enTranslations;
        });

        // 2. Register Redux reducer
        registry.registerReducer(callsReducer as never);

        // 3. Fetch initial config status
        await fetchConfigStatus(store);

        // 3b. Prefetch enabled state for all user-visible call channels so that
        // ChannelHeaderButton can render with the correct visibility on the very
        // first frame (no flicker for disabled channels). Failures are non-fatal:
        // the per-channel useChannelEnabled hook will fill the gap on demand.
        await fetchAllChannelEnabled(store);

        // 4. Get current user ID for WS handlers
        const state = store.getState();
        const currentUserId = (state as unknown as {entities: {users: {currentUserId: string}}}).
            entities?.users?.currentUserId ?? '';

        // 5. Register WebSocket event handlers
        registry.registerWebSocketEventHandler(
            `custom_${manifest.id}_call_started`,
            handleCallStarted(store as unknown as Store<GlobalState>, currentUserId),
        );
        registry.registerWebSocketEventHandler(
            `custom_${manifest.id}_user_joined`,
            handleUserJoined(store as unknown as Store<GlobalState>),
        );
        registry.registerWebSocketEventHandler(
            `custom_${manifest.id}_user_left`,
            handleUserLeft(store as unknown as Store<GlobalState>, currentUserId),
        );
        registry.registerWebSocketEventHandler(
            `custom_${manifest.id}_call_ended`,
            handleCallEnded(store as unknown as Store<GlobalState>),
        );
        registry.registerWebSocketEventHandler(
            `custom_${manifest.id}_notification_dismissed`,
            handleNotifDismissed(store as unknown as Store<GlobalState>, currentUserId),
        );

        // 6. Re-fetch config on WebSocket reconnect
        registry.registerReconnectHandler(async () => {
            await fetchConfigStatus(store);
            await fetchAllChannelEnabled(store);
        });

        // 7. Register UI components
        registry.registerCallButtonAction(

            // button: shown when this is the only call plugin
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

            // dropdownButton: shown when multiple call plugins are registered (e.g., alongside Calls plugin)
            () => {
                const channelId = (store.getState() as unknown as {
                    entities: {channels: {currentChannelId: string}};
                }).entities?.channels?.currentChannelId ?? '';

                return (
                    <ChannelHeaderDropdownButton
                        channel={{id: channelId} as never}
                        currentUserId={currentUserId}
                    />
                );
            },

            // fn: click handler — all call logic lives here because Mattermost wraps the
            // button component in a <button onClick={fn}>, making nested <button> elements invalid.
            async (channel: Channel) => {
                const s = store.getState() as unknown as GlobalState;
                const activeCall = selectCallByChannel(channel.id)(s);
                const myActiveCall = selectMyActiveCall(s);
                const isParticipant = selectIsCurrentUserParticipant(channel.id, currentUserId)(s);

                if (isParticipant) {
                    return;
                }

                // Both endpoints return the full CallSession struct from the server.
                // Note: the HTTP JSON field is "create_at" (from CallSession.CreateAt),
                // whereas the call_started WS event uses "start_at" — they are different serializations.
                interface CallSessionData {
                    id: string;
                    channel_id: string;
                    creator_id: string;
                    participants: string[];
                    create_at: number;
                    post_id: string;
                }
                interface CallTokenResponse {
                    call: CallSessionData;
                    token: string;
                }
                interface StartCallResponse {
                    call: CallSessionData;
                    token: string;
                }

                if (activeCall) {
                    if (myActiveCall?.callId === activeCall.id) {
                        return;
                    }
                    if (myActiveCall && myActiveCall.callId !== activeCall.id) {
                        store.dispatch(setPendingSwitchCallId(activeCall.id));
                        return;
                    }
                    store.dispatch(setCallLoading(true));
                    const result = await pluginFetch<CallTokenResponse>(
                        `/api/v1/calls/${activeCall.id}/token`,
                        {method: 'POST'},
                    );
                    store.dispatch(setCallLoading(false));
                    if ('error' in result) {
                        store.dispatch(setCallError(getCallErrorMessage(result.code, result.error)));
                        return;
                    }
                    const {data: joinData} = result;
                    store.dispatch(upsertCall({
                        id: joinData.call.id,
                        channelId: joinData.call.channel_id,
                        creatorId: joinData.call.creator_id,
                        participants: joinData.call.participants,
                        startAt: joinData.call.create_at,
                        postId: joinData.call.post_id,
                    }));
                    playJoinSound();
                    store.dispatch(setMyActiveCall({
                        callId: joinData.call.id,
                        channelId: joinData.call.channel_id,
                        token: joinData.token,
                    }));
                    return;
                }

                // Start new call
                store.dispatch(setCallLoading(true));
                const result = await pluginFetch<StartCallResponse>('/api/v1/calls', {
                    method: 'POST',
                    body: JSON.stringify({channel_id: channel.id}),
                });
                store.dispatch(setCallLoading(false));
                if ('error' in result) {
                    store.dispatch(setCallError(getCallErrorMessage(result.code, result.error)));
                    return;
                }
                const {data} = result;
                store.dispatch(upsertCall({
                    id: data.call.id,
                    channelId: data.call.channel_id,
                    creatorId: data.call.creator_id,
                    participants: data.call.participants,
                    startAt: data.call.create_at,
                    postId: data.call.post_id,
                }));
                playJoinSound();
                store.dispatch(setMyActiveCall({
                    callId: data.call.id,
                    channelId: data.call.channel_id,
                    token: data.token,
                }));
            },
        );

        // 7b. Register "Disable/Enable calls" in the channel "Other actions" menu.
        // Only visible to channel admins and system admins (enforced in the component).
        registry.registerChannelHeaderMenuAction(
            ChannelMenuAction,
            async (channelId: string) => {
                const s = store.getState() as unknown as GlobalState;
                const currentEnabled = selectChannelEnabled(channelId)(s) ?? true;
                const newEnabled = !currentEnabled;

                const result = await pluginFetch<{channel_id: string; enabled: boolean}>(
                    `/api/v1/channels/${channelId}`,
                    {
                        method: 'PUT',
                        body: JSON.stringify({enabled: newEnabled}),
                    },
                );

                if ('error' in result) {
                    return;
                }

                // Trust the server's reported state (in case of races, future
                // server-side coercion, or other discrepancies). This update
                // makes the user's own UI react to their own toggle: the
                // header "Start Call (RTK)" button hides, and the menu item
                // label flips between Enable/Disable.
                store.dispatch(setChannelEnabled(result.data.channel_id, result.data.enabled));
            },
        );

        registry.registerRootComponent(() => (
            <CallActionsRoot/>
        ));

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
            <IncomingCallNotification/>
        ));

        // 8. Register custom post type renderer
        registry.registerPostTypeComponent(
            'custom_cf_call',
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            CallPost as any,
        );

        // 9. Register custom admin console settings (show env var status)
        registry.registerAdminConsoleCustomSetting(
            'CloudflareAccountID',
            EnvVarCredentialSetting as never,
            {showTitle: true},
        );
        registry.registerAdminConsoleCustomSetting(
            'CloudflareAppID',
            EnvVarCredentialSetting as never,
            {showTitle: true},
        );
        registry.registerAdminConsoleCustomSetting(
            'CloudflareAPIToken',
            EnvVarCredentialSetting as never,
            {showTitle: true},
        );
    }
}

declare global {
    interface Window {
        registerPlugin(pluginId: string, plugin: Plugin): void;
    }
}

window.registerPlugin(manifest.id, new Plugin());
