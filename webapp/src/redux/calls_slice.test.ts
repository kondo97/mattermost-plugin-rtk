// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {
    callsReducer,
    setPluginEnabled,
    upsertCall,
    removeCall,
    setMyActiveCall,
    clearMyActiveCall,
    setIncomingCall,
    clearIncomingCall,
} from './calls_slice';
import type {CallsPluginState} from './calls_slice';

const initialState: CallsPluginState = {
    pluginEnabled: false,
    callsByChannel: {},
    myActiveCall: null,
    incomingCall: null,
};

const sampleCall = {
    id: 'call1',
    channelId: 'channel1',
    creatorId: 'user1',
    participants: ['user1'],
    startAt: 1000000,
    postId: 'post1',
};

describe('callsReducer', () => {
    describe('initialState', () => {
        it('returns initial state when called with undefined', () => {
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            const state = callsReducer(undefined, {type: '@@INIT'} as any);
            expect(state).toEqual(initialState);
        });
    });

    describe('SET_PLUGIN_ENABLED', () => {
        it('sets pluginEnabled to true', () => {
            const state = callsReducer(initialState, setPluginEnabled(true));
            expect(state.pluginEnabled).toBe(true);
        });

        it('sets pluginEnabled to false', () => {
            const prevState: CallsPluginState = {...initialState, pluginEnabled: true};
            const state = callsReducer(prevState, setPluginEnabled(false));
            expect(state.pluginEnabled).toBe(false);
        });

        it('does not mutate other fields', () => {
            const prevState: CallsPluginState = {
                ...initialState,
                callsByChannel: {channel1: sampleCall},
            };
            const state = callsReducer(prevState, setPluginEnabled(true));
            expect(state.callsByChannel).toEqual(prevState.callsByChannel);
            expect(state.myActiveCall).toBeNull();
            expect(state.incomingCall).toBeNull();
        });
    });

    describe('UPSERT_CALL', () => {
        it('adds a new call to callsByChannel', () => {
            const state = callsReducer(initialState, upsertCall(sampleCall));
            expect(state.callsByChannel.channel1).toEqual(sampleCall);
        });

        it('updates an existing call', () => {
            const prevState: CallsPluginState = {
                ...initialState,
                callsByChannel: {channel1: sampleCall},
            };
            const updatedCall = {...sampleCall, participants: ['user1', 'user2']};
            const state = callsReducer(prevState, upsertCall(updatedCall));
            expect(state.callsByChannel.channel1.participants).toEqual(['user1', 'user2']);
        });

        it('preserves other channels when upserting', () => {
            const otherCall = {...sampleCall, id: 'call2', channelId: 'channel2'};
            const prevState: CallsPluginState = {
                ...initialState,
                callsByChannel: {channel2: otherCall},
            };
            const state = callsReducer(prevState, upsertCall(sampleCall));
            expect(state.callsByChannel.channel1).toEqual(sampleCall);
            expect(state.callsByChannel.channel2).toEqual(otherCall);
        });
    });

    describe('REMOVE_CALL', () => {
        it('removes a call from callsByChannel', () => {
            const prevState: CallsPluginState = {
                ...initialState,
                callsByChannel: {channel1: sampleCall},
            };
            const state = callsReducer(prevState, removeCall('channel1'));
            expect(state.callsByChannel.channel1).toBeUndefined();
        });

        it('preserves other channels when removing', () => {
            const otherCall = {...sampleCall, id: 'call2', channelId: 'channel2'};
            const prevState: CallsPluginState = {
                ...initialState,
                callsByChannel: {channel1: sampleCall, channel2: otherCall},
            };
            const state = callsReducer(prevState, removeCall('channel1'));
            expect(state.callsByChannel.channel1).toBeUndefined();
            expect(state.callsByChannel.channel2).toEqual(otherCall);
        });

        it('is a no-op if the channel does not exist', () => {
            const state = callsReducer(initialState, removeCall('nonexistent'));
            expect(state.callsByChannel).toEqual({});
        });

        it('uses spread/delete pattern (does not mutate state)', () => {
            const prevState: CallsPluginState = {
                ...initialState,
                callsByChannel: {channel1: sampleCall},
            };
            const state = callsReducer(prevState, removeCall('channel1'));
            expect(state).not.toBe(prevState);
            expect(state.callsByChannel).not.toBe(prevState.callsByChannel);
        });
    });

    describe('SET_MY_ACTIVE_CALL', () => {
        it('sets myActiveCall', () => {
            const myCall = {callId: 'call1', channelId: 'channel1', token: 'tok123'};
            const state = callsReducer(initialState, setMyActiveCall(myCall));
            expect(state.myActiveCall).toEqual(myCall);
        });

        it('sets myActiveCall with featureFlags', () => {
            const flags = {recording: false, screenShare: true, polls: true, transcription: false, waitingRoom: false, video: true, chat: true, plugins: true, participants: true, raiseHand: true};
            const myCall = {callId: 'call1', channelId: 'channel1', token: 'tok123', featureFlags: flags};
            const state = callsReducer(initialState, setMyActiveCall(myCall));
            expect(state.myActiveCall).toEqual(myCall);
            expect(state.myActiveCall?.featureFlags?.recording).toBe(false);
        });

        it('replaces an existing myActiveCall', () => {
            const myCall1 = {callId: 'call1', channelId: 'channel1', token: 'tok1'};
            const myCall2 = {callId: 'call2', channelId: 'channel2', token: 'tok2'};
            const prevState: CallsPluginState = {...initialState, myActiveCall: myCall1};
            const state = callsReducer(prevState, setMyActiveCall(myCall2));
            expect(state.myActiveCall).toEqual(myCall2);
        });
    });

    describe('CLEAR_MY_ACTIVE_CALL', () => {
        it('clears myActiveCall to null', () => {
            const myCall = {callId: 'call1', channelId: 'channel1', token: 'tok123'};
            const prevState: CallsPluginState = {...initialState, myActiveCall: myCall};
            const state = callsReducer(prevState, clearMyActiveCall());
            expect(state.myActiveCall).toBeNull();
        });

        it('is a no-op when myActiveCall is already null', () => {
            const state = callsReducer(initialState, clearMyActiveCall());
            expect(state.myActiveCall).toBeNull();
        });
    });

    describe('SET_INCOMING_CALL', () => {
        it('sets incomingCall', () => {
            const incoming = {callId: 'call1', channelId: 'channel1', creatorId: 'user2', startAt: 0};
            const state = callsReducer(initialState, setIncomingCall(incoming));
            expect(state.incomingCall).toEqual(incoming);
        });

        it('replaces an existing incomingCall', () => {
            const incoming1 = {callId: 'call1', channelId: 'channel1', creatorId: 'user2', startAt: 0};
            const incoming2 = {callId: 'call2', channelId: 'channel2', creatorId: 'user3', startAt: 0};
            const prevState: CallsPluginState = {...initialState, incomingCall: incoming1};
            const state = callsReducer(prevState, setIncomingCall(incoming2));
            expect(state.incomingCall).toEqual(incoming2);
        });
    });

    describe('CLEAR_INCOMING_CALL', () => {
        it('clears incomingCall to null', () => {
            const incoming = {callId: 'call1', channelId: 'channel1', creatorId: 'user2', startAt: 0};
            const prevState: CallsPluginState = {...initialState, incomingCall: incoming};
            const state = callsReducer(prevState, clearIncomingCall());
            expect(state.incomingCall).toBeNull();
        });

        it('is a no-op when incomingCall is already null', () => {
            const state = callsReducer(initialState, clearIncomingCall());
            expect(state.incomingCall).toBeNull();
        });
    });

    describe('unknown action', () => {
        it('returns state unchanged for unknown action type', () => {
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            const state = callsReducer(initialState, {type: 'UNKNOWN_ACTION'} as any);
            expect(state).toEqual(initialState);
        });
    });
});
