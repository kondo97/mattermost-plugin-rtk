// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {useIntl} from 'react-intl';

interface Props {
    participants: string[];
    startAt: number;
    isAlreadyInCall: boolean;
    onJoin: () => void;
}

const CallPostActive = ({participants, startAt, isAlreadyInCall, onJoin}: Props) => {
    const intl = useIntl();

    const startedAt = new Date(startAt).toLocaleTimeString([], {
        hour: '2-digit',
        minute: '2-digit',
    });
    const visibleParticipants = participants.slice(0, 3);
    const overflowCount = participants.length > 3 ? participants.length - 3 : 0;

    return (
        <>
            <div style={{display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '8px'}}>
                <span
                    style={{
                        width: '10px',
                        height: '10px',
                        borderRadius: '50%',
                        background: 'var(--online-indicator)',
                        display: 'inline-block',
                    }}
                    data-testid='call-post-status-indicator'
                />
                <span
                    style={{fontWeight: 600}}
                    data-testid='call-post-label'
                >
                    {intl.formatMessage({id: 'plugin.rtk.call_post.label_active'})}
                </span>
                <span
                    style={{color: 'var(--center-channel-color-56)', fontSize: '12px'}}
                    data-testid='call-post-start-time'
                >
                    {intl.formatMessage(
                        {id: 'plugin.rtk.call_post.started_at'},
                        {time: startedAt},
                    )}
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
                        data-testid='call-post-avatar'
                    >
                        {userId.slice(0, 1).toUpperCase()}
                    </span>
                ))}
                {overflowCount > 0 && (
                    <span
                        style={{fontSize: '12px', alignSelf: 'center'}}
                        data-testid='call-post-overflow'
                    >
                        {`(+${overflowCount})`}
                    </span>
                )}
            </div>

            <div>
                {intl.formatMessage(
                    {id: 'plugin.rtk.call_post.participants'},
                    {count: participants.length},
                )}
            </div>

            <button
                type='button'
                className='btn btn-primary btn-sm'
                style={{marginTop: '8px'}}
                disabled={isAlreadyInCall}
                title={isAlreadyInCall ?
                    intl.formatMessage({id: 'plugin.rtk.call_post.tooltip_already_in_call'}) :
                    undefined}
                onClick={isAlreadyInCall ? undefined : onJoin}
                data-testid='call-post-join-button'
            >
                {intl.formatMessage({id: 'plugin.rtk.call_post.join'})}
            </button>
        </>
    );
};

export default CallPostActive;
