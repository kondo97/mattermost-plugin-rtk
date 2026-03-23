// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useEffect, useState} from 'react';
import {useIntl} from 'react-intl';
import {useSelector, useDispatch} from 'react-redux';

import type {GlobalState} from '@mattermost/types/store';

import {pluginFetch} from 'client';
import {clearIncomingCall, setMyActiveCall} from 'redux/calls_slice';
import {selectIncomingCall, selectMyActiveCall} from 'redux/selectors';
import {buildCallTabUrl, getChannelDisplayName} from 'utils/call_tab';
import manifest from 'manifest';

import SwitchCallModal from 'components/switch_call_modal';

const INCOMING_CALL_TIMEOUT_MS = 30_000;

interface CallResponse {
    call: {id: string; channel_id: string};
    token: string;
}

interface Props {
    currentUserId: string;
}

const IncomingCallNotification = ({currentUserId}: Props) => {
    const intl = useIntl();
    const dispatch = useDispatch();

    const incomingCall = useSelector(selectIncomingCall);
    const myActiveCall = useSelector(selectMyActiveCall);
    const channelDisplayName = useSelector(
        (state: GlobalState) => incomingCall
            ? getChannelDisplayName(state, incomingCall.channelId)
            : '',
    );

    const [showSwitchModal, setShowSwitchModal] = useState(false);

    // 30-second auto-dismiss timer — cleanup when incomingCall changes (REL-U3-06)
    useEffect(() => {
        if (!incomingCall) {
            return undefined;
        }
        const timeout = setTimeout(() => {
            dispatch(clearIncomingCall());
        }, INCOMING_CALL_TIMEOUT_MS);
        return () => clearTimeout(timeout);
    }, [incomingCall?.callId]); // eslint-disable-line react-hooks/exhaustive-deps

    if (!incomingCall) {
        return null;
    }

    const handleIgnore = () => {
        // Call server to emit custom_cf_notification_dismissed to all user sessions
        // State cleared when WS event arrives (BR-007) — fire-and-forget
        pluginFetch(`/api/v1/calls/${incomingCall.callId}/dismiss`, {method: 'POST'});
    };

    const joinCall = async (callId: string) => {
        const result = await pluginFetch<CallResponse>(`/api/v1/calls/${callId}/token`, {
            method: 'POST',
        });
        if ('error' in result) {
            return;
        }
        const {data} = result;
        dispatch(setMyActiveCall({
            callId: data.call.id,
            channelId: data.call.channel_id,
            token: data.token,
        }));
        // Token intentionally not logged — SEC-U3-01
        window.open(
            buildCallTabUrl(manifest.id, data.token, data.call.id, channelDisplayName),
            '_blank',
            'noopener,noreferrer',
        );
    };

    const handleJoin = () => {
        if (myActiveCall && myActiveCall.callId !== incomingCall.callId) {
            setShowSwitchModal(true);
            return;
        }
        joinCall(incomingCall.callId);
    };

    const handleSwitchConfirm = async () => {
        setShowSwitchModal(false);
        if (myActiveCall) {
            pluginFetch(`/api/v1/calls/${myActiveCall.callId}/leave`, {method: 'POST'});
        }
        await joinCall(incomingCall.callId);
    };

    const handleSwitchCancel = () => {
        setShowSwitchModal(false);
    };

    return (
        <>
            <div
                style={{
                    position: 'fixed',
                    top: '16px',
                    right: '16px',
                    background: 'var(--center-channel-bg)',
                    border: '1px solid var(--center-channel-color-16)',
                    borderRadius: '8px',
                    padding: '16px',
                    zIndex: 1100,
                    minWidth: '260px',
                    boxShadow: '0 4px 16px rgba(0,0,0,0.15)',
                }}
                data-testid='incoming-call-notification'
            >
                <div
                    style={{fontWeight: 600, marginBottom: '4px'}}
                    data-testid='incoming-call-title'
                >
                    {intl.formatMessage({id: 'plugin.rtk.incoming_call.title'})}
                </div>
                <div
                    style={{marginBottom: '12px'}}
                    data-testid='incoming-call-from'
                >
                    {intl.formatMessage(
                        {id: 'plugin.rtk.incoming_call.from'},
                        {name: incomingCall.creatorId},
                    )}
                </div>
                <div style={{display: 'flex', gap: '8px'}}>
                    <button
                        type='button'
                        className='btn btn-link btn-sm'
                        onClick={handleIgnore}
                        data-testid='incoming-call-ignore'
                    >
                        {intl.formatMessage({id: 'plugin.rtk.incoming_call.ignore'})}
                    </button>
                    <button
                        type='button'
                        className='btn btn-primary btn-sm'
                        onClick={handleJoin}
                        data-testid='incoming-call-join'
                    >
                        {intl.formatMessage({id: 'plugin.rtk.incoming_call.join'})}
                    </button>
                </div>
            </div>

            <SwitchCallModal
                visible={showSwitchModal}
                onConfirm={handleSwitchConfirm}
                onCancel={handleSwitchCancel}
            />
        </>
    );
};

export default IncomingCallNotification;
