// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import manifest from 'manifest';

import type {GlobalState} from '@mattermost/types/store';

import type {ActiveCall, CallsPluginState, IncomingCall, MyActiveCall} from './calls_slice';

// ---------------------------------------------------------------------------
// Plugin state accessor
// ---------------------------------------------------------------------------

const PLUGIN_STATE_KEY = `plugins-${manifest.id}`;

function selectPluginState(state: GlobalState): CallsPluginState {
    return (state as unknown as Record<string, CallsPluginState>)[PLUGIN_STATE_KEY] ?? {
        callsByChannel: {},
        myActiveCall: null,
        incomingCall: null,
        pluginEnabled: false,
        callLoading: false,
        callError: null,
        pendingSwitchCallId: null,
    };
}

// ---------------------------------------------------------------------------
// Selectors
// ---------------------------------------------------------------------------

export function selectPluginEnabled(state: GlobalState): boolean {
    return selectPluginState(state).pluginEnabled;
}

export function selectCallByChannel(channelId: string) {
    return (state: GlobalState): ActiveCall | undefined =>
        selectPluginState(state).callsByChannel[channelId];
}

export function selectMyActiveCall(state: GlobalState): MyActiveCall | null {
    return selectPluginState(state).myActiveCall;
}

export function selectIncomingCall(state: GlobalState): IncomingCall | null {
    return selectPluginState(state).incomingCall;
}

export function selectIsCurrentUserParticipant(channelId: string, currentUserId: string) {
    return (state: GlobalState): boolean => {
        const call = selectPluginState(state).callsByChannel[channelId];
        return call?.participants.includes(currentUserId) ?? false;
    };
}

export function selectCallLoading(state: GlobalState): boolean {
    return selectPluginState(state).callLoading;
}

export function selectCallError(state: GlobalState): string | null {
    return selectPluginState(state).callError;
}

export function selectPendingSwitchCallId(state: GlobalState): string | null {
    return selectPluginState(state).pendingSwitchCallId;
}
