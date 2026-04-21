// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {pluginFetch} from 'client';
import React, {useState} from 'react';
import {useIntl} from 'react-intl';
import {useSelector, useDispatch} from 'react-redux';
import {setMyActiveCall, upsertCall} from 'redux/calls_slice';
import type {FeatureFlags} from 'redux/calls_slice';
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
    feature_flags?: Record<string, boolean>;
}

interface Props {
    channel: Channel;
    currentUserId: string;
}

// Phone icon SVG matching Mattermost Calls plugin style
const PhoneIcon = () => (
    <svg
        width='18'
        height='18'
        viewBox='0 0 24 24'
        fill='currentColor'
        style={{display: 'block'}}
    >
        <path d='M6.62 10.79a15.053 15.053 0 006.59 6.59l2.2-2.2a1.003 1.003 0 011.01-.24c1.12.37 2.33.57 3.57.57.55 0 1 .45 1 1V20c0 .55-.45 1-1 1-9.39 0-17-7.61-17-17 0-.55.45-1 1-1h3.5c.55 0 1 .45 1 1 0 1.25.2 2.45.57 3.57.1.31.03.66-.25 1.02l-2.19 2.2z'/>
    </svg>
);

// Spinner icon for loading state
const SpinnerIcon = () => (
    <svg
        width='18'
        height='18'
        viewBox='0 0 24 24'
        fill='none'
        stroke='currentColor'
        strokeWidth='2'
        strokeLinecap='round'
        style={{display: 'block', animation: 'rtk-spin 1s linear infinite'}}
    >
        <path d='M12 2a10 10 0 0 1 10 10'/>
    </svg>
);

// Active call indicator (pulsing dot)
const ActiveIndicator = () => (
    <span
        style={{
            width: '8px',
            height: '8px',
            borderRadius: '50%',
            backgroundColor: '#3db887',
            display: 'inline-block',
            marginLeft: '6px',
            animation: 'rtk-pulse 1.5s ease-in-out infinite',
        }}
    />
);

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
    const [hover, setHover] = useState(false);

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

        // Do NOT dispatch upsertCall here. The participant update will arrive via
        // the user_joined WebSocket event, which is emitted by the server only after
        // the RTK webhook confirms the user has actually connected via WebRTC.
        // This prevents the post from showing "participating" before the SDK joins.
        dispatch(setMyActiveCall({
            callId: data.call.id,
            channelId: data.call.channel_id,
            token: data.token,
            featureFlags: data.feature_flags as FeatureFlags | undefined,
        }));
    };

    const handleClick = async () => {
        if (loading || isParticipant) {
            return;
        }

        if (activeCall) {
            // Already connecting to this call (token issued but RTK webhook not yet received)
            if (myActiveCall?.callId === activeCall.id) {
                return;
            }
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
            featureFlags: data.feature_flags as FeatureFlags | undefined,
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
    let hasActiveCall = false;

    if (loading) {
        label = intl.formatMessage({id: 'plugin.rtk.channel_header.starting_call'});
        tooltip = label;
        disabled = true;
    } else if (isParticipant) {
        label = intl.formatMessage({id: 'plugin.rtk.channel_header.in_call'});
        tooltip = intl.formatMessage({id: 'plugin.rtk.channel_header.tooltip_in_call'});
        disabled = true;
        hasActiveCall = true;
    } else if (activeCall) {
        label = intl.formatMessage({id: 'plugin.rtk.channel_header.join_call'});
        tooltip = intl.formatMessage({id: 'plugin.rtk.channel_header.tooltip_join'});
        hasActiveCall = true;
    } else {
        label = intl.formatMessage({id: 'plugin.rtk.channel_header.start_call'});
        tooltip = intl.formatMessage({id: 'plugin.rtk.channel_header.tooltip_start'});
    }

    const buttonStyle: React.CSSProperties = {
        display: 'inline-flex',
        alignItems: 'center',
        gap: '6px',
        height: '32px',
        padding: '0 12px',
        borderRadius: '4px',
        border: 'none',
        fontSize: '12px',
        fontWeight: 600,
        lineHeight: '16px',
        cursor: disabled ? 'default' : 'pointer',
        transition: 'background 0.15s ease, color 0.15s ease',
        color: disabled ?
            'rgba(var(--center-channel-color-rgb), 0.32)' :
            '#3db887',
        background: hover && !disabled ?
            'rgba(var(--center-channel-color-rgb), 0.08)' :
            'transparent',
    };

    return (
        <>
            {/* Keyframe animations injected once */}
            <style>{`
                @keyframes rtk-spin {
                    to { transform: rotate(360deg); }
                }
                @keyframes rtk-pulse {
                    0%, 100% { opacity: 1; }
                    50% { opacity: 0.4; }
                }
            `}</style>

            <button
                type='button'
                className='style--none'
                title={tooltip}
                aria-label={tooltip}
                disabled={disabled}
                onClick={handleClick}
                onMouseEnter={() => setHover(true)}
                onMouseLeave={() => setHover(false)}
                style={buttonStyle}
                data-testid='channel-header-call-button'
            >
                {loading ? <SpinnerIcon/> : <PhoneIcon/>}
                <span data-testid='channel-header-call-button-label'>{label}</span>
                {hasActiveCall && !loading && <ActiveIndicator/>}
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
