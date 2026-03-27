// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {pluginFetch} from 'client';
import React, {useState} from 'react';
import {useIntl} from 'react-intl';
import {useSelector, useDispatch} from 'react-redux';
import {setMyActiveCall} from 'redux/calls_slice';
import {
    selectCallByChannel,
    selectIsCurrentUserParticipant,
    selectMyActiveCall,
} from 'redux/selectors';

import SwitchCallModal from 'components/switch_call_modal';

interface CallResponse {
    call: {
        id: string;
        channel_id: string;
    };
    token: string;
}

interface Props {
    currentChannelId: string;
    currentUserId: string;
}

const ToastBar = ({currentChannelId, currentUserId}: Props) => {
    const intl = useIntl();
    const dispatch = useDispatch();

    const activeCall = useSelector(selectCallByChannel(currentChannelId));
    const isParticipant = useSelector(
        selectIsCurrentUserParticipant(currentChannelId, currentUserId),
    );
    const myActiveCall = useSelector(selectMyActiveCall);

    const [dismissed, setDismissed] = useState(false);
    const [showSwitchModal, setShowSwitchModal] = useState(false);
    const [pendingJoinCallId, setPendingJoinCallId] = useState<string | null>(null);

    // Visibility: active call in current channel, user not a participant, not dismissed
    if (!activeCall || isParticipant || dismissed) {
        return null;
    }

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
    };

    const handleJoin = () => {
        if (myActiveCall && myActiveCall.callId !== activeCall.id) {
            setPendingJoinCallId(activeCall.id);
            setShowSwitchModal(true);
            return;
        }
        joinCall(activeCall.id);
    };

    const handleSwitchConfirm = async () => {
        setShowSwitchModal(false);
        if (!pendingJoinCallId) {
            return;
        }
        if (myActiveCall) {
            pluginFetch(`/api/v1/calls/${myActiveCall.callId}/leave`, {method: 'POST'});
        }
        await joinCall(pendingJoinCallId);
        setPendingJoinCallId(null);
    };

    const handleSwitchCancel = () => {
        setShowSwitchModal(false);
        setPendingJoinCallId(null);
    };

    const startedAt = new Date(activeCall.startAt).toLocaleTimeString([], {
        hour: '2-digit',
        minute: '2-digit',
    });
    const overflowCount = activeCall.participants.length > 3 ?
        activeCall.participants.length - 3 : 0;

    return (
        <>
            <div
                className='toast toast__visible'
                data-testid='call-toast-bar'
            >
                <div
                    className='toast__message'
                    data-testid='call-toast-bar-info'
                >
                    <span>
                        {intl.formatMessage(
                            {id: 'plugin.rtk.toast_bar.started_at'},
                            {time: startedAt},
                        )}
                    </span>
                    <span>
                        {intl.formatMessage(
                            {id: 'plugin.rtk.toast_bar.participants'},
                            {count: activeCall.participants.length},
                        )}
                        {overflowCount > 0 && ` (+${overflowCount})`}
                    </span>
                </div>
                <button
                    type='button'
                    className='btn btn-primary btn-sm'
                    onClick={handleJoin}
                    data-testid='call-toast-bar-join'
                >
                    {intl.formatMessage({id: 'plugin.rtk.toast_bar.join'})}
                </button>
                <button
                    type='button'
                    className='style--none toast__dismiss'
                    onClick={() => setDismissed(true)}
                    aria-label={intl.formatMessage({id: 'plugin.rtk.toast_bar.dismiss'})}
                    data-testid='call-toast-bar-dismiss'
                >
                    {'×'}
                </button>
            </div>

            <SwitchCallModal
                visible={showSwitchModal}
                onConfirm={handleSwitchConfirm}
                onCancel={handleSwitchCancel}
            />
        </>
    );
};

export default ToastBar;
