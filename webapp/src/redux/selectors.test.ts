// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {
    selectPluginEnabled,
    selectCallByChannel,
    selectMyActiveCall,
    selectIncomingCall,
    selectIsCurrentUserParticipant,
} from './selectors';

import type {CallsPluginState} from './calls_slice';

const PLUGIN_KEY = 'plugins-com.mattermost.plugin-rtk';

const sampleCall = {
    id: 'call1',
    channelId: 'channel1',
    creatorId: 'user1',
    participants: ['user1', 'user2'],
    startAt: 1000000,
};

const buildState = (pluginState: Partial<CallsPluginState> = {}) => ({
    [PLUGIN_KEY]: {
        pluginEnabled: false,
        callsByChannel: {},
        myActiveCall: null,
        incomingCall: null,
        ...pluginState,
    },
});

describe('selectPluginEnabled', () => {
    it('returns false when pluginEnabled is false', () => {
        const state = buildState({pluginEnabled: false});
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        expect(selectPluginEnabled(state as any)).toBe(false);
    });

    it('returns true when pluginEnabled is true', () => {
        const state = buildState({pluginEnabled: true});
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        expect(selectPluginEnabled(state as any)).toBe(true);
    });
});

describe('selectCallByChannel', () => {
    it('returns the call for a given channel', () => {
        const state = buildState({callsByChannel: {channel1: sampleCall}});
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        expect(selectCallByChannel('channel1')(state as any)).toEqual(sampleCall);
    });

    it('returns undefined when there is no call for the channel', () => {
        const state = buildState({callsByChannel: {}});
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        expect(selectCallByChannel('channel1')(state as any)).toBeUndefined();
    });

    it('returns undefined for a different channel', () => {
        const state = buildState({callsByChannel: {channel1: sampleCall}});
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        expect(selectCallByChannel('channel2')(state as any)).toBeUndefined();
    });
});

describe('selectMyActiveCall', () => {
    it('returns null when there is no active call', () => {
        const state = buildState({myActiveCall: null});
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        expect(selectMyActiveCall(state as any)).toBeNull();
    });

    it('returns the active call when present', () => {
        const myCall = {callId: 'call1', channelId: 'channel1', token: 'tok123'};
        const state = buildState({myActiveCall: myCall});
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        expect(selectMyActiveCall(state as any)).toEqual(myCall);
    });
});

describe('selectIncomingCall', () => {
    it('returns null when there is no incoming call', () => {
        const state = buildState({incomingCall: null});
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        expect(selectIncomingCall(state as any)).toBeNull();
    });

    it('returns the incoming call when present', () => {
        const incoming = {callId: 'call1', channelId: 'dm1', creatorId: 'otherUser'};
        const state = buildState({incomingCall: incoming});
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        expect(selectIncomingCall(state as any)).toEqual(incoming);
    });
});

describe('selectIsCurrentUserParticipant', () => {
    it('returns true when user is a participant', () => {
        const state = buildState({callsByChannel: {channel1: sampleCall}});
        expect(
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            selectIsCurrentUserParticipant('channel1', 'user1')(state as any),
        ).toBe(true);
    });

    it('returns false when user is NOT a participant', () => {
        const state = buildState({callsByChannel: {channel1: sampleCall}});
        expect(
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            selectIsCurrentUserParticipant('channel1', 'user99')(state as any),
        ).toBe(false);
    });

    it('returns false when there is no active call in the channel', () => {
        const state = buildState({callsByChannel: {}});
        expect(
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            selectIsCurrentUserParticipant('channel1', 'user1')(state as any),
        ).toBe(false);
    });

    it('returns false when checking a different channel', () => {
        const state = buildState({callsByChannel: {channel1: sampleCall}});
        expect(
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            selectIsCurrentUserParticipant('channel2', 'user1')(state as any),
        ).toBe(false);
    });
});
