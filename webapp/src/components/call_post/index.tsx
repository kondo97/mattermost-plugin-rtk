// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useState} from 'react';
import {useIntl} from 'react-intl';
import {useSelector, useDispatch} from 'react-redux';

import type {GlobalState} from '@mattermost/types/store';

import manifest from 'manifest';
import {pluginFetch} from 'client';
import {setMyActiveCall} from 'redux/calls_slice';
import {selectCallByChannel, selectMyActiveCall} from 'redux/selectors';
import {buildCallTabUrl, getChannelDisplayName} from 'utils/call_tab';
import SwitchCallModal from 'components/switch_call_modal';

import CallPostActive from './CallPostActive';
import CallPostEnded from './CallPostEnded';

interface CallPostProps {
    call_id: string;
    channel_id: string;
    creator_id: string;
    start_at: number;
    end_at: number;
    participants: string[];
}

interface CallResponse {
    call: {id: string; channel_id: string};
    token: string;
}

// Post type injected by Mattermost post renderer
interface Post {
    id: string;
    props: CallPostProps;
}

interface Props {
    post: Post;
}

const CallPost = ({post}: Props) => {
    const intl = useIntl();
    const dispatch = useDispatch();

    const props = post.props;
    const channelId = props.channel_id;

    // Live Redux state (may be undefined before first WS event — pattern U4-4)
    const liveCall = useSelector(selectCallByChannel(channelId));
    const myActiveCall = useSelector(selectMyActiveCall);
    const channelName = useSelector(
        (state: GlobalState) => getChannelDisplayName(state, channelId),
    );

    const [showSwitchModal, setShowSwitchModal] = useState(false);
    const [pendingCallId, setPendingCallId] = useState<string | null>(null);
    const [errorMsg, setErrorMsg] = useState<string | null>(null);

    // Merge: Redux wins if available, post.props as fallback (pattern U4-4)
    const participants = liveCall?.participants ?? props.participants ?? [];
    const startAt = liveCall?.startAt ?? props.start_at;
    const endAt = liveCall
        ? ((liveCall as unknown as {endAt?: number}).endAt ?? 0)
        : props.end_at;
    const isEnded = endAt > 0;

    const isAlreadyInCall = myActiveCall?.callId === props.call_id;

    const joinCall = async (callId: string) => {
        const result = await pluginFetch<CallResponse>(`/api/v1/calls/${callId}/token`, {
            method: 'POST',
        });
        if ('error' in result) {
            setErrorMsg(result.error);
            return;
        }
        const {data} = result;
        dispatch(setMyActiveCall({
            callId: data.call.id,
            channelId: data.call.channel_id,
            token: data.token,
        }));
        // Token intentionally not logged — SEC-U4-01
        window.open(
            buildCallTabUrl(manifest.id, data.token, data.call.id, channelName),
            '_blank',
            'noopener,noreferrer',
        );
    };

    const handleJoin = () => {
        if (isAlreadyInCall) {
            return;
        }
        if (myActiveCall && myActiveCall.callId !== props.call_id) {
            setPendingCallId(props.call_id);
            setShowSwitchModal(true);
            return;
        }
        joinCall(props.call_id);
    };

    const handleSwitchConfirm = async () => {
        setShowSwitchModal(false);
        if (!pendingCallId) {
            return;
        }
        if (myActiveCall) {
            pluginFetch(`/api/v1/calls/${myActiveCall.callId}/leave`, {method: 'POST'});
        }
        await joinCall(pendingCallId);
        setPendingCallId(null);
    };

    const handleSwitchCancel = () => {
        setShowSwitchModal(false);
        setPendingCallId(null);
    };

    return (
        <>
            <div
                style={{
                    padding: '12px 16px',
                    border: '1px solid var(--center-channel-color-16)',
                    borderRadius: '8px',
                    background: 'var(--center-channel-bg)',
                    maxWidth: '400px',
                }}
                data-testid='call-post'
            >
                {isEnded ? (
                    <CallPostEnded
                        startAt={startAt}
                        endAt={endAt}
                    />
                ) : (
                    <CallPostActive
                        participants={participants}
                        startAt={startAt}
                        isAlreadyInCall={isAlreadyInCall}
                        onJoin={handleJoin}
                    />
                )}
            </div>

            <SwitchCallModal
                visible={showSwitchModal}
                onConfirm={handleSwitchConfirm}
                onCancel={handleSwitchCancel}
            />

            {errorMsg && (
                <div
                    className='modal fade in'
                    style={{display: 'block', background: 'rgba(0,0,0,0.5)'}}
                    data-testid='call-post-error-modal'
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
                                    data-testid='call-post-error-modal-close'
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

export default CallPost;
