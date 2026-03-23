// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// ---------------------------------------------------------------------------
// Domain types
// ---------------------------------------------------------------------------

export interface ActiveCall {
    id: string;
    channelId: string;
    creatorId: string;
    participants: string[];
    startAt: number; // Unix ms
    postId: string;
}

export interface MyActiveCall {
    callId: string;
    channelId: string;
    token: string; // JWT for reopening the call tab — MUST NOT be logged
}

export interface IncomingCall {
    callId: string;
    channelId: string;
    creatorId: string;
    startAt: number; // Unix ms
}

export interface CallsPluginState {
    callsByChannel: Record<string, ActiveCall>;
    myActiveCall: MyActiveCall | null;
    incomingCall: IncomingCall | null;
    pluginEnabled: boolean;
}

// ---------------------------------------------------------------------------
// Action type constants
// ---------------------------------------------------------------------------

const SET_PLUGIN_ENABLED = 'rtk-calls/setPluginEnabled' as const;
const UPSERT_CALL = 'rtk-calls/upsertCall' as const;
const REMOVE_CALL = 'rtk-calls/removeCall' as const;
const SET_MY_ACTIVE_CALL = 'rtk-calls/setMyActiveCall' as const;
const CLEAR_MY_ACTIVE_CALL = 'rtk-calls/clearMyActiveCall' as const;
const SET_INCOMING_CALL = 'rtk-calls/setIncomingCall' as const;
const CLEAR_INCOMING_CALL = 'rtk-calls/clearIncomingCall' as const;

// ---------------------------------------------------------------------------
// Action creators
// ---------------------------------------------------------------------------

export const setPluginEnabled = (enabled: boolean) =>
    ({type: SET_PLUGIN_ENABLED, payload: enabled} as const);

export const upsertCall = (call: ActiveCall) =>
    ({type: UPSERT_CALL, payload: call} as const);

export const removeCall = (channelId: string) =>
    ({type: REMOVE_CALL, payload: channelId} as const);

export const setMyActiveCall = (call: MyActiveCall) =>
    ({type: SET_MY_ACTIVE_CALL, payload: call} as const);

export const clearMyActiveCall = () =>
    ({type: CLEAR_MY_ACTIVE_CALL} as const);

export const setIncomingCall = (call: IncomingCall) =>
    ({type: SET_INCOMING_CALL, payload: call} as const);

export const clearIncomingCall = () =>
    ({type: CLEAR_INCOMING_CALL} as const);

// ---------------------------------------------------------------------------
// Action union type
// ---------------------------------------------------------------------------

type CallsAction =
    | ReturnType<typeof setPluginEnabled>
    | ReturnType<typeof upsertCall>
    | ReturnType<typeof removeCall>
    | ReturnType<typeof setMyActiveCall>
    | ReturnType<typeof clearMyActiveCall>
    | ReturnType<typeof setIncomingCall>
    | ReturnType<typeof clearIncomingCall>;

// ---------------------------------------------------------------------------
// Reducer
// ---------------------------------------------------------------------------

const initialState: CallsPluginState = {
    callsByChannel: {},
    myActiveCall: null,
    incomingCall: null,
    pluginEnabled: false,
};

export function callsReducer(
    state: CallsPluginState = initialState,
    action: CallsAction,
): CallsPluginState {
    switch (action.type) {
    case SET_PLUGIN_ENABLED:
        return {...state, pluginEnabled: action.payload};

    case UPSERT_CALL:
        return {
            ...state,
            callsByChannel: {
                ...state.callsByChannel,
                [action.payload.channelId]: action.payload,
            },
        };

    case REMOVE_CALL: {
        // eslint-disable-next-line @typescript-eslint/no-unused-vars, @typescript-eslint/naming-convention
        const {[action.payload]: _removed, ...remaining} = state.callsByChannel;
        return {...state, callsByChannel: remaining};
    }

    case SET_MY_ACTIVE_CALL:
        return {...state, myActiveCall: action.payload};

    case CLEAR_MY_ACTIVE_CALL:
        return {...state, myActiveCall: null};

    case SET_INCOMING_CALL:
        return {...state, incomingCall: action.payload};

    case CLEAR_INCOMING_CALL:
        return {...state, incomingCall: null};

    default:
        return state;
    }
}
