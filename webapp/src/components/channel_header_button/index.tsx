// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {pluginFetch} from 'client';
import React, {useState} from 'react';
import {useIntl} from 'react-intl';
import {useSelector, useDispatch} from 'react-redux';
import {setMyActiveCall, upsertCall} from 'redux/calls_slice';
import {
    selectCallByChannel,
    selectIsCurrentUserParticipant,
    selectMyActiveCall,
    selectPluginEnabled,
} from 'redux/selectors';

import type {Channel} from '@mattermost/types/channels';

import SwitchCallModal from 'components/switch_call_modal';

interface CallResponse {
    call: {
        id: string;
        channel_id: string;
        creator_id: string;
        meeting_id: string;
        participants: string[];
        start_at: number;
        end_at: number;
        post_id: string;
    };
    token: string;
}

interface Props {
    channel: Channel;
    currentUserId: string;
}

const ChannelHeaderButton = ({channel, currentUserId}: Props) => {
    const intl = useIntl();
    const dispatch = useDispatch();

    const pluginEnabled = useSelector(selectPluginEnabled);
    const activeCall = useSelector(selectCallByChannel(channel.id));
    const myActiveCall = useSelector(selectMyActiveCall);
    const isParticipant = useSelector(selectIsCurrentUserParticipant(channel.id, currentUserId));

    const [loading, setLoading] = useState(false);
    const [showSwitchModal, setShowSwitchModal] = useState(false);
    const [pendingJoinCallId, setPendingJoinCallId] = useState<string | null>(null);
    const [errorMsg, setErrorMsg] = useState<string | null>(null);

    if (!pluginEnabled) {
        return null;
    }

    const joinCall = async (callId: string) => {
        const result = await pluginFetch<CallResponse>(`/api/v1/calls/${callId}/token`, {
            method: 'POST',
        });
        if ('error' in result) {
            setErrorMsg(result.error);
            return;
        }
        const {data} = result;
        console.log('[rtk-plugin] ChannelHeader joinCall response:', {callId: data.call.id, channelId: data.call.channel_id, tokenLen: data.token?.length, participants: data.call.participants}); // eslint-disable-line no-console
        dispatch(upsertCall({
            id: data.call.id,
            channelId: data.call.channel_id,
            creatorId: data.call.creator_id,
            participants: data.call.participants,
            startAt: data.call.start_at,
            postId: data.call.post_id,
        }));
        dispatch(setMyActiveCall({
            callId: data.call.id,
            channelId: data.call.channel_id,
            token: data.token,
        }));
    };

    const handleClick = async () => {
        if (loading || isParticipant) {
            return;
        }

        if (activeCall) {
            // Join call — check if already in a different call
            if (myActiveCall && myActiveCall.callId !== activeCall.id) {
                setPendingJoinCallId(activeCall.id);
                setShowSwitchModal(true);
                return;
            }
            await joinCall(activeCall.id);
            return;
        }

        // Start call
        setLoading(true);
        const result = await pluginFetch<CallResponse>('/api/v1/calls', {
            method: 'POST',
            body: JSON.stringify({channel_id: channel.id}),
        });
        setLoading(false);

        if ('error' in result) {
            setErrorMsg(result.error);
            return;
        }
        const {data} = result;
        dispatch(upsertCall({
            id: data.call.id,
            channelId: data.call.channel_id,
            creatorId: data.call.creator_id,
            participants: data.call.participants,
            startAt: data.call.start_at,
            postId: data.call.post_id,
        }));
        dispatch(setMyActiveCall({
            callId: data.call.id,
            channelId: data.call.channel_id,
            token: data.token,
        }));
    };

    const handleSwitchConfirm = async () => {
        setShowSwitchModal(false);
        if (!pendingJoinCallId) {
            return;
        }

        // Leave current call first (fire-and-forget)
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

    let label: string;
    let tooltip: string;
    let disabled = false;

    if (loading) {
        label = intl.formatMessage({id: 'plugin.rtk.channel_header.starting_call'});
        tooltip = label;
        disabled = true;
    } else if (isParticipant) {
        label = intl.formatMessage({id: 'plugin.rtk.channel_header.in_call'});
        tooltip = intl.formatMessage({id: 'plugin.rtk.channel_header.tooltip_in_call'});
        disabled = true;
    } else if (activeCall) {
        label = intl.formatMessage({id: 'plugin.rtk.channel_header.join_call'});
        tooltip = intl.formatMessage({id: 'plugin.rtk.channel_header.tooltip_join'});
    } else {
        label = intl.formatMessage({id: 'plugin.rtk.channel_header.start_call'});
        tooltip = intl.formatMessage({id: 'plugin.rtk.channel_header.tooltip_start'});
    }

    return (
        <>
            <button
                type='button'
                className='style--none'
                title={tooltip}
                aria-label={tooltip}
                disabled={disabled}
                onClick={handleClick}
                data-testid='channel-header-call-button'
            >
                <span data-testid='channel-header-call-button-label'>{label}</span>
            </button>

            <SwitchCallModal
                visible={showSwitchModal}
                onConfirm={handleSwitchConfirm}
                onCancel={handleSwitchCancel}
            />

            {errorMsg && (
                <div
                    className='modal fade in'
                    style={{display: 'block', background: 'rgba(0,0,0,0.5)'}}
                    data-testid='call-error-modal'
                >
                    <div className='modal-dialog'>
                        <div className='modal-content'>
                            <div className='modal-body'>
                                <p>{errorMsg}</p>
                            </div>
                            <div className='modal-footer'>
                                <button
                                    type='button'
                                    className='btn btn-primary'
                                    onClick={() => setErrorMsg(null)}
                                    data-testid='call-error-modal-close'
                                >
                                    {intl.formatMessage({id: 'plugin.rtk.call_post.error_close'})}
                                </button>
                            </div>
                        </div>
                    </div>
                </div>
            )}
        </>
    );
};

export default ChannelHeaderButton;
