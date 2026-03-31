// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

/* eslint-disable react/prop-types */

import {pluginFetch} from 'client';
import React, {useEffect, useState} from 'react';
import {useIntl} from 'react-intl';
import {useSelector, useDispatch} from 'react-redux';
import {setMyActiveCall, upsertCall} from 'redux/calls_slice';
import {selectCallByChannel, selectMyActiveCall} from 'redux/selectors';

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

    const [showSwitchModal, setShowSwitchModal] = useState(false);
    const [pendingCallId, setPendingCallId] = useState<string | null>(null);
    const [errorMsg, setErrorMsg] = useState<string | null>(null);

    // On mount, fetch the latest call state from the server so that the post
    // reflects reality even right after a page reload (before any WS event).
    useEffect(() => {
        if (props.end_at > 0 || (liveCall?.id === props.call_id)) {
            return;
        }
        pluginFetch<CallResponse['call']>(`/api/v1/calls/${props.call_id}`).then((result) => {
            if ('data' in result) {
                const d = result.data;
                dispatch(upsertCall({
                    id: d.id,
                    channelId: d.channel_id,
                    creatorId: d.creator_id,
                    participants: d.participants,
                    startAt: d.start_at,
                    postId: d.post_id,
                }));
            }
        });
    }, [props.call_id, props.end_at, liveCall?.id, dispatch]);

    // Only use live Redux state when it matches THIS post's call (pattern U4-4).
    // Without this guard every call post in the channel would appear active
    // whenever any call in the channel is ongoing.
    const matchingLiveCall = liveCall?.id === props.call_id ? liveCall : undefined;

    const participants = matchingLiveCall?.participants ?? props.participants ?? [];
    const startAt = matchingLiveCall?.startAt ?? props.start_at;
    const endAt = matchingLiveCall ?
        ((matchingLiveCall as unknown as {endAt?: number}).endAt ?? 0) :
        props.end_at;
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
        console.log('[rtk-plugin] CallPost joinCall response:', {callId: data.call.id, channelId: data.call.channel_id, tokenLen: data.token?.length, participants: data.call.participants}); // eslint-disable-line no-console
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
