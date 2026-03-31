// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import type {Store} from 'redux';

import type {Channel} from '@mattermost/types/channels';
import type {GlobalState} from '@mattermost/types/store';

import {
    clearIncomingCall,
    clearMyActiveCall,
    removeCall,
    setIncomingCall,
    upsertCall,
} from './calls_slice';
import {selectCallByChannel, selectIncomingCall, selectMyActiveCall} from './selectors';

// ---------------------------------------------------------------------------
// Channel type constants (for DM/GM detection)
// ---------------------------------------------------------------------------

const CHANNEL_TYPE_DM = 'D';
const CHANNEL_TYPE_GM = 'G';

// ---------------------------------------------------------------------------
// WS payload type guards (SECURITY-05, SECURITY-13)
// ---------------------------------------------------------------------------

interface CallStartedPayload {
    call_id: string;
    channel_id: string;
    creator_id: string;
    participants: string[];
    start_at: number;
    post_id: string;
}

function isCallStartedPayload(data: unknown): data is CallStartedPayload {
    if (!data || typeof data !== 'object') {
        return false;
    }
    const d = data as Record<string, unknown>;
    return (
        typeof d.call_id === 'string' && d.call_id.length > 0 &&
        typeof d.channel_id === 'string' && d.channel_id.length > 0 &&
        typeof d.creator_id === 'string' && d.creator_id.length > 0 &&
        Array.isArray(d.participants) &&
        typeof d.start_at === 'number' &&
        typeof d.post_id === 'string'
    );
}

interface UserJoinedPayload {
    call_id: string;
    channel_id: string;
    user_id: string;
    participants: string[];
}

function isUserJoinedPayload(data: unknown): data is UserJoinedPayload {
    if (!data || typeof data !== 'object') {
        return false;
    }
    const d = data as Record<string, unknown>;
    return (
        typeof d.call_id === 'string' && d.call_id.length > 0 &&
        typeof d.channel_id === 'string' && d.channel_id.length > 0 &&
        typeof d.user_id === 'string' && d.user_id.length > 0 &&
        Array.isArray(d.participants)
    );
}

interface UserLeftPayload {
    call_id: string;
    channel_id: string;
    user_id: string;
    participants: string[];
}

function isUserLeftPayload(data: unknown): data is UserLeftPayload {
    return isUserJoinedPayload(data);
}

interface CallEndedPayload {
    call_id: string;
    channel_id: string;
    end_at: number;
    duration_ms: number;
}

function isCallEndedPayload(data: unknown): data is CallEndedPayload {
    if (!data || typeof data !== 'object') {
        return false;
    }
    const d = data as Record<string, unknown>;
    return (
        typeof d.call_id === 'string' && d.call_id.length > 0 &&
        typeof d.channel_id === 'string' && d.channel_id.length > 0 &&
        typeof d.end_at === 'number' &&
        typeof d.duration_ms === 'number'
    );
}

interface NotifDismissedPayload {
    call_id: string;
    user_id: string;
}

function isNotifDismissedPayload(data: unknown): data is NotifDismissedPayload {
    if (!data || typeof data !== 'object') {
        return false;
    }
    const d = data as Record<string, unknown>;
    return (
        typeof d.call_id === 'string' && d.call_id.length > 0 &&
        typeof d.user_id === 'string' && d.user_id.length > 0
    );
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function getChannelType(store: Store<GlobalState>, channelId: string): string {
    const state = store.getState();

    // mattermost-redux stores channels at state.entities.channels.channels
    const channels = (state as unknown as {entities: {channels: {channels: Record<string, Channel>}}}).
        entities?.channels?.channels;
    return channels?.[channelId]?.type ?? '';
}

// ---------------------------------------------------------------------------
// WS event handlers
// ---------------------------------------------------------------------------

function parseEventData(msg: {data: unknown}): unknown {
    try {
        return typeof msg.data === 'string' ? JSON.parse(msg.data) : msg.data;
    } catch {
        return undefined;
    }
}

export function handleCallStarted(store: Store<GlobalState>, currentUserId: string) {
    return (msg: {data: unknown}) => {
        const data = parseEventData(msg);
        if (!isCallStartedPayload(data)) {
            console.error('[rtk-plugin] invalid custom_cf_call_started payload', data); // eslint-disable-line no-console
            return;
        }

        store.dispatch(upsertCall({
            id: data.call_id,
            channelId: data.channel_id,
            creatorId: data.creator_id,
            participants: data.participants,
            startAt: data.start_at,
            postId: data.post_id,
        }));

        // Show ringing notification only for DM/GM channels, not for the call creator
        const channelType = getChannelType(store, data.channel_id);
        const isDmOrGm = channelType === CHANNEL_TYPE_DM || channelType === CHANNEL_TYPE_GM;
        if (isDmOrGm && data.creator_id !== currentUserId) {
            store.dispatch(setIncomingCall({
                callId: data.call_id,
                channelId: data.channel_id,
                creatorId: data.creator_id,
                startAt: data.start_at,
            }));
        }
    };
}

// eslint-disable-next-line @typescript-eslint/no-unused-vars
export function handleUserJoined(store: Store<GlobalState>, _currentUserId: string) {
    return (msg: {data: unknown}) => {
        const data = parseEventData(msg);
        if (!isUserJoinedPayload(data)) {
            console.error('[rtk-plugin] invalid custom_cf_user_joined payload', data); // eslint-disable-line no-console
            return;
        }

        const existing = selectCallByChannel(data.channel_id)(store.getState() as unknown as GlobalState);

        if (existing) {
            store.dispatch(upsertCall({
                ...existing,
                channelId: data.channel_id,
                participants: data.participants,
            }));
        }

        // Note: myActiveCall is NOT set here — the token is only available
        // from the API response (joinCall). Setting it with an empty token
        // would race with the API response and prevent RTK SDK initialization.
    };
}

export function handleUserLeft(store: Store<GlobalState>, currentUserId: string) {
    return (msg: {data: unknown}) => {
        const data = parseEventData(msg);
        if (!isUserLeftPayload(data)) {
            console.error('[rtk-plugin] invalid custom_cf_user_left payload', data); // eslint-disable-line no-console
            return;
        }

        const existing = selectCallByChannel(data.channel_id)(store.getState() as unknown as GlobalState);

        if (existing) {
            store.dispatch(upsertCall({
                ...existing,
                channelId: data.channel_id,
                participants: data.participants,
            }));
        }

        if (data.user_id === currentUserId) {
            store.dispatch(clearMyActiveCall());
        }
    };
}

export function handleCallEnded(store: Store<GlobalState>, currentUserId: string) {
    return (msg: {data: unknown}) => {
        const data = parseEventData(msg);
        if (!isCallEndedPayload(data)) {
            console.error('[rtk-plugin] invalid custom_cf_call_ended payload', data); // eslint-disable-line no-console
            return;
        }

        store.dispatch(removeCall(data.channel_id));

        const myActiveCall = selectMyActiveCall(store.getState() as unknown as GlobalState);
        if (myActiveCall?.callId === data.call_id) {
            store.dispatch(clearMyActiveCall());
        }

        const incomingCall = selectIncomingCall(store.getState() as unknown as GlobalState);
        if (incomingCall?.callId === data.call_id) {
            store.dispatch(clearIncomingCall());
        }

        // suppress unused parameter warning
        void currentUserId; // eslint-disable-line no-void
    };
}

export function handleNotifDismissed(store: Store<GlobalState>, currentUserId: string) {
    return (msg: {data: unknown}) => {
        const data = parseEventData(msg);
        if (!isNotifDismissedPayload(data)) {
            console.error('[rtk-plugin] invalid custom_cf_notification_dismissed payload', data); // eslint-disable-line no-console
            return;
        }

        if (data.user_id === currentUserId) {
            const incomingCall = selectIncomingCall(store.getState() as unknown as GlobalState);
            if (incomingCall?.callId === data.call_id) {
                store.dispatch(clearIncomingCall());
            }
        }
    };
}
