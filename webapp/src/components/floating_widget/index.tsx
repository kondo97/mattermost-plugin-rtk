// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useEffect, useState} from 'react';
import {useIntl} from 'react-intl';
import {useSelector} from 'react-redux';

import type {GlobalState} from '@mattermost/types/store';

import {pluginFetch} from 'client';
import {selectCallByChannel, selectMyActiveCall} from 'redux/selectors';
import {buildCallTabUrl, getChannelDisplayName} from 'utils/call_tab';
import manifest from 'manifest';

interface CallResponse {
    call: {id: string; channel_id: string};
    token: string;
}

const FloatingWidget = () => {
    const intl = useIntl();
    const myActiveCall = useSelector(selectMyActiveCall);
    const activeCall = useSelector(
        myActiveCall ? selectCallByChannel(myActiveCall.channelId) : () => undefined,
    );
    const channelDisplayName = useSelector(
        (state: GlobalState) => myActiveCall
            ? getChannelDisplayName(state, myActiveCall.channelId)
            : '',
    );

    const [elapsedSeconds, setElapsedSeconds] = useState(0);

    // Duration timer — cleanup on unmount or call change (REL-U3-05)
    useEffect(() => {
        if (!myActiveCall || !activeCall) {
            return undefined;
        }
        const startAt = activeCall.startAt;
        setElapsedSeconds(Math.floor((Date.now() - startAt) / 1000));
        const interval = setInterval(() => {
            setElapsedSeconds(Math.floor((Date.now() - startAt) / 1000));
        }, 1000);
        return () => clearInterval(interval);
    }, [myActiveCall?.callId]); // eslint-disable-line react-hooks/exhaustive-deps

    if (!myActiveCall || !activeCall) {
        return null;
    }

    const formatDuration = (seconds: number): string => {
        const h = Math.floor(seconds / 3600);
        const m = Math.floor((seconds % 3600) / 60);
        const s = seconds % 60;
        const pad = (n: number) => String(n).padStart(2, '0');
        if (h > 0) {
            return `${h}:${pad(m)}:${pad(s)}`;
        }
        return `${pad(m)}:${pad(s)}`;
    };

    const handleOpenInNewTab = async () => {
        // Always fetch a fresh token (BR-011, Pattern U3-7)
        const result = await pluginFetch<CallResponse>(
            `/api/v1/calls/${myActiveCall.callId}/token`,
            {method: 'POST'},
        );
        if ('error' in result) {
            return;
        }
        // Token intentionally not logged — SEC-U3-01
        window.open(
            buildCallTabUrl(manifest.id, result.data.token, myActiveCall.callId, channelDisplayName),
            '_blank',
            'noopener,noreferrer',
        );
    };

    const visibleParticipants = activeCall.participants.slice(0, 3);
    const overflowCount = activeCall.participants.length > 3 ?
        activeCall.participants.length - 3 : 0;

    return (
        <div
            className='rtk-floating-widget'
            style={{
                position: 'fixed',
                bottom: '16px',
                right: '16px',
                background: 'var(--center-channel-bg)',
                border: '1px solid var(--center-channel-color-16)',
                borderRadius: '8px',
                padding: '12px 16px',
                zIndex: 1000,
                minWidth: '220px',
                boxShadow: '0 4px 16px rgba(0,0,0,0.15)',
            }}
            data-testid='floating-widget'
        >
            <div
                style={{fontWeight: 600, marginBottom: '4px'}}
                data-testid='floating-widget-channel'
            >
                {`#${myActiveCall.channelId}`}
            </div>
            <div
                style={{display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '8px'}}
                data-testid='floating-widget-info'
            >
                <span data-testid='floating-widget-participants'>
                    {intl.formatMessage(
                        {id: 'plugin.rtk.floating_widget.participants'},
                        {count: activeCall.participants.length},
                    )}
                    {overflowCount > 0 && ` (+${overflowCount})`}
                </span>
                <span
                    style={{marginLeft: 'auto', fontFamily: 'monospace'}}
                    data-testid='floating-widget-duration'
                >
                    {formatDuration(elapsedSeconds)}
                </span>
            </div>
            <div style={{display: 'flex', gap: '4px', marginBottom: '8px'}}>
                {visibleParticipants.map((userId) => (
                    <span
                        key={userId}
                        style={{
                            width: '24px',
                            height: '24px',
                            borderRadius: '50%',
                            background: 'var(--button-bg)',
                            display: 'inline-flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            fontSize: '10px',
                            color: 'var(--button-color)',
                        }}
                        title={userId}
                    >
                        {userId.slice(0, 1).toUpperCase()}
                    </span>
                ))}
            </div>
            <button
                type='button'
                className='btn btn-primary btn-sm'
                style={{width: '100%'}}
                onClick={handleOpenInNewTab}
                data-testid='floating-widget-open-tab'
            >
                {intl.formatMessage({id: 'plugin.rtk.floating_widget.open_in_tab'})}
            </button>
        </div>
    );
};

export default FloatingWidget;
