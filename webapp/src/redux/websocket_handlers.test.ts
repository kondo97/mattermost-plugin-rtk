// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {
    upsertCall,
    removeCall,
    clearMyActiveCall,
    setIncomingCall,
    clearIncomingCall,
} from './calls_slice';
import {
    handleCallStarted,
    handleUserJoined,
    handleUserLeft,
    handleCallEnded,
    handleNotifDismissed,
} from './websocket_handlers';

// Minimal mock store
const makeStore = (state: object) => {
    const dispatched: unknown[] = [];
    return {
        getState: () => state,
        dispatch: (action: unknown) => {
            dispatched.push(action);
        },
        dispatched,
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
    } as any;
};

const baseState = {
    entities: {
        channels: {
            channels: {
                channel1: {id: 'channel1', type: 'O'},
                dm1: {id: 'dm1', type: 'D'},
                gm1: {id: 'gm1', type: 'G'},
            },
        },
    },
    'plugins-com.kondo97.mattermost-plugin-rtk': {
        callsByChannel: {},
        myActiveCall: null,
        incomingCall: null,
        pluginEnabled: true,
    },
};

const makeEvent = (data: object) => ({data: JSON.stringify(data)});

describe('handleCallStarted', () => {
    const currentUserId = 'currentUser';

    it('dispatches upsertCall for a valid payload', () => {
        const store = makeStore(baseState);
        const handler = handleCallStarted(store, currentUserId);
        handler(makeEvent({
            call_id: 'call1',
            channel_id: 'channel1',
            creator_id: 'user1',
            participants: ['user1'],
            start_at: 1000000,
            post_id: '',
            channel_type: 'O',
        }));
        expect(store.dispatched).toHaveLength(1);
        expect(store.dispatched[0]).toEqual(upsertCall({
            id: 'call1',
            channelId: 'channel1',
            creatorId: 'user1',
            participants: ['user1'],
            startAt: 1000000,
            postId: '',
        }));
    });

    it('dispatches setIncomingCall for DM channel when current user is not creator', () => {
        const store = makeStore(baseState);
        const handler = handleCallStarted(store, currentUserId);
        handler(makeEvent({
            call_id: 'call1',
            channel_id: 'dm1',
            creator_id: 'otherUser',
            participants: ['otherUser'],
            start_at: 1000000,
            post_id: '',
            channel_type: 'D',
        }));
        expect(store.dispatched).toHaveLength(2);
        expect(store.dispatched[0]).toEqual(upsertCall({
            id: 'call1',
            channelId: 'dm1',
            creatorId: 'otherUser',
            participants: ['otherUser'],
            startAt: 1000000,
            postId: '',
        }));
        expect(store.dispatched[1]).toEqual(setIncomingCall({
            callId: 'call1',
            channelId: 'dm1',
            creatorId: 'otherUser',
            startAt: 1000000,
        }));
    });

    it('dispatches setIncomingCall for GM channel when current user is not creator', () => {
        const store = makeStore(baseState);
        const handler = handleCallStarted(store, currentUserId);
        handler(makeEvent({
            call_id: 'call1',
            channel_id: 'gm1',
            creator_id: 'otherUser',
            participants: ['otherUser'],
            start_at: 1000000,
            post_id: '',
            channel_type: 'G',
        }));
        const incomingAction = store.dispatched.find(
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            (a: any) => a.type === setIncomingCall({callId: '', channelId: '', creatorId: '', startAt: 0}).type,
        );
        expect(incomingAction).toBeDefined();
    });

    it('does NOT dispatch setIncomingCall for public channel', () => {
        const store = makeStore(baseState);
        const handler = handleCallStarted(store, currentUserId);
        handler(makeEvent({
            call_id: 'call1',
            channel_id: 'channel1',
            creator_id: 'otherUser',
            participants: ['otherUser'],
            start_at: 1000000,
            post_id: '',
            channel_type: 'O',
        }));
        expect(store.dispatched).toHaveLength(1);
        expect(store.dispatched[0]).toEqual(upsertCall(expect.anything()));
    });

    it('does NOT dispatch setIncomingCall when current user is the creator', () => {
        const store = makeStore(baseState);
        const handler = handleCallStarted(store, currentUserId);
        handler(makeEvent({
            call_id: 'call1',
            channel_id: 'dm1',
            creator_id: currentUserId,
            participants: [currentUserId],
            start_at: 1000000,
            post_id: '',
            channel_type: 'D',
        }));
        expect(store.dispatched).toHaveLength(1);
        expect(store.dispatched[0]).toEqual(upsertCall(expect.anything()));
    });

    it('ignores invalid payload (missing required fields)', () => {
        const store = makeStore(baseState);
        const handler = handleCallStarted(store, currentUserId);
        handler(makeEvent({call_id: 'call1'}));
        expect(store.dispatched).toHaveLength(0);
    });

    it('ignores non-JSON data', () => {
        const store = makeStore(baseState);
        const handler = handleCallStarted(store, currentUserId);
        handler({data: 'not-json'});
        expect(store.dispatched).toHaveLength(0);
    });
});

describe('handleUserJoined', () => {
    const currentUserId = 'currentUser';
    const stateWithCall = {
        ...baseState,
        'plugins-com.kondo97.mattermost-plugin-rtk': {
            ...baseState['plugins-com.kondo97.mattermost-plugin-rtk'],
            callsByChannel: {
                channel1: {
                    id: 'call1',
                    channelId: 'channel1',
                    creatorId: 'user1',
                    participants: ['user1'],
                    startAt: 1000000,
                },
            },
        },
    };

    it('dispatches upsertCall with updated participants', () => {
        const store = makeStore(stateWithCall);
        const handler = handleUserJoined(store, currentUserId);
        handler(makeEvent({call_id: 'call1', user_id: 'user2', channel_id: 'channel1', participants: ['user1', 'user2']}));
        expect(store.dispatched).toHaveLength(1);
        const action = store.dispatched[0] as ReturnType<typeof upsertCall>;
        expect(action.payload.participants).toContain('user2');
        expect(action.payload.participants).toContain('user1');
    });

    it('does NOT dispatch setMyActiveCall when joined user is current user (token only from API)', () => {
        const store = makeStore(stateWithCall);
        const handler = handleUserJoined(store, currentUserId);
        handler(makeEvent({call_id: 'call1', user_id: currentUserId, channel_id: 'channel1', participants: ['user1', currentUserId]}));
        expect(store.dispatched).toHaveLength(1);
        expect(store.dispatched[0]).toEqual(upsertCall(expect.anything()));
    });

    it('ignores invalid payload', () => {
        const store = makeStore(stateWithCall);
        const handler = handleUserJoined(store, currentUserId);
        handler(makeEvent({user_id: 'user2'}));
        expect(store.dispatched).toHaveLength(0);
    });
});

describe('handleUserLeft', () => {
    const currentUserId = 'currentUser';
    const stateWithCall = {
        ...baseState,
        'plugins-com.kondo97.mattermost-plugin-rtk': {
            ...baseState['plugins-com.kondo97.mattermost-plugin-rtk'],
            callsByChannel: {
                channel1: {
                    id: 'call1',
                    channelId: 'channel1',
                    creatorId: 'user1',
                    participants: ['user1', 'user2'],
                    startAt: 1000000,
                },
            },
        },
    };

    it('dispatches upsertCall with participant removed', () => {
        const store = makeStore(stateWithCall);
        const handler = handleUserLeft(store, currentUserId);
        handler(makeEvent({call_id: 'call1', user_id: 'user2', channel_id: 'channel1', participants: ['user1']}));
        expect(store.dispatched).toHaveLength(1);
        const action = store.dispatched[0] as ReturnType<typeof upsertCall>;
        expect(action.payload.participants).not.toContain('user2');
        expect(action.payload.participants).toContain('user1');
    });

    it('dispatches clearMyActiveCall when leaving user is current user', () => {
        const stateWithMyCall = {
            ...stateWithCall,
            'plugins-com.kondo97.mattermost-plugin-rtk': {
                ...stateWithCall['plugins-com.kondo97.mattermost-plugin-rtk'],
                myActiveCall: {callId: 'call1', channelId: 'channel1', token: 'tok1'},
            },
        };
        const store = makeStore(stateWithMyCall);
        const handler = handleUserLeft(store, currentUserId);
        handler(makeEvent({call_id: 'call1', user_id: currentUserId, channel_id: 'channel1', participants: []}));
        const clearAction = store.dispatched.find(
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            (a: any) => a.type === clearMyActiveCall().type,
        );
        expect(clearAction).toBeDefined();
    });

    it('does NOT dispatch clearMyActiveCall for other users', () => {
        const store = makeStore(stateWithCall);
        const handler = handleUserLeft(store, currentUserId);
        handler(makeEvent({call_id: 'call1', user_id: 'user2', channel_id: 'channel1', participants: ['user1']}));
        const clearAction = store.dispatched.find(
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            (a: any) => a.type === clearMyActiveCall().type,
        );
        expect(clearAction).toBeUndefined();
    });

    it('ignores invalid payload', () => {
        const store = makeStore(stateWithCall);
        const handler = handleUserLeft(store, currentUserId);
        handler(makeEvent({call_id: 'call1'}));
        expect(store.dispatched).toHaveLength(0);
    });
});

describe('handleCallEnded', () => {
    const currentUserId = 'currentUser';
    const stateWithCall = {
        ...baseState,
        'plugins-com.kondo97.mattermost-plugin-rtk': {
            ...baseState['plugins-com.kondo97.mattermost-plugin-rtk'],
            callsByChannel: {
                channel1: {
                    id: 'call1',
                    channelId: 'channel1',
                    creatorId: 'user1',
                    participants: ['user1'],
                    startAt: 1000000,
                },
            },
            myActiveCall: {callId: 'call1', channelId: 'channel1', token: 'tok1'},
        },
    };

    it('dispatches removeCall', () => {
        const store = makeStore(stateWithCall);
        const handler = handleCallEnded(store, currentUserId);
        handler(makeEvent({call_id: 'call1', channel_id: 'channel1', end_at: 2000000, duration_ms: 1000000}));
        const removeAction = store.dispatched.find(
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            (a: any) => a.type === removeCall('').type,
        );
        expect(removeAction).toBeDefined();
        expect((removeAction as ReturnType<typeof removeCall>).payload).toBe('channel1');
    });

    it('dispatches clearMyActiveCall when active call matches ended call', () => {
        const store = makeStore(stateWithCall);
        const handler = handleCallEnded(store, currentUserId);
        handler(makeEvent({call_id: 'call1', channel_id: 'channel1', end_at: 2000000, duration_ms: 1000000}));
        const clearAction = store.dispatched.find(
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            (a: any) => a.type === clearMyActiveCall().type,
        );
        expect(clearAction).toBeDefined();
    });

    it('does NOT dispatch clearMyActiveCall when active call is different', () => {
        const stateWithDifferentCall = {
            ...stateWithCall,
            'plugins-com.kondo97.mattermost-plugin-rtk': {
                ...stateWithCall['plugins-com.kondo97.mattermost-plugin-rtk'],
                myActiveCall: {callId: 'call99', channelId: 'channel99', token: 'tok99'},
            },
        };
        const store = makeStore(stateWithDifferentCall);
        const handler = handleCallEnded(store, currentUserId);
        handler(makeEvent({call_id: 'call1', channel_id: 'channel1', end_at: 2000000, duration_ms: 1000000}));
        const clearAction = store.dispatched.find(
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            (a: any) => a.type === clearMyActiveCall().type,
        );
        expect(clearAction).toBeUndefined();
    });

    it('ignores invalid payload', () => {
        const store = makeStore(stateWithCall);
        const handler = handleCallEnded(store, currentUserId);
        handler(makeEvent({channel_id: 'channel1'}));
        expect(store.dispatched).toHaveLength(0);
    });
});

describe('handleNotifDismissed', () => {
    const currentUserId = 'currentUser';
    const stateWithIncoming = {
        ...baseState,
        'plugins-com.kondo97.mattermost-plugin-rtk': {
            ...baseState['plugins-com.kondo97.mattermost-plugin-rtk'],
            incomingCall: {callId: 'call1', channelId: 'dm1', creatorId: 'otherUser'},
        },
    };

    it('dispatches clearIncomingCall when call_id matches incoming', () => {
        const store = makeStore(stateWithIncoming);
        const handler = handleNotifDismissed(store, currentUserId);
        handler(makeEvent({call_id: 'call1', user_id: currentUserId}));
        const clearAction = store.dispatched.find(
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            (a: any) => a.type === clearIncomingCall().type,
        );
        expect(clearAction).toBeDefined();
    });

    it('does NOT dispatch clearIncomingCall when call_id does not match', () => {
        const store = makeStore(stateWithIncoming);
        const handler = handleNotifDismissed(store, currentUserId);
        handler(makeEvent({call_id: 'call99', user_id: currentUserId}));
        expect(store.dispatched).toHaveLength(0);
    });

    it('does NOT dispatch clearIncomingCall for other user_id (dismiss is user-scoped on server)', () => {
        const store = makeStore(stateWithIncoming);
        const handler = handleNotifDismissed(store, currentUserId);
        handler(makeEvent({call_id: 'call1', user_id: 'otherUser'}));
        expect(store.dispatched).toHaveLength(0);
    });

    it('ignores invalid payload', () => {
        const store = makeStore(stateWithIncoming);
        const handler = handleNotifDismissed(store, currentUserId);
        handler(makeEvent({user_id: currentUserId}));
        expect(store.dispatched).toHaveLength(0);
    });
});
