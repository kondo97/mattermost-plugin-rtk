// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {useIntl} from 'react-intl';

interface Props {
    startAt: number;
    endAt: number;
}

function formatDuration(ms: number): string {
    const totalSeconds = Math.floor(ms / 1000);
    const hours = Math.floor(totalSeconds / 3600);
    const minutes = Math.floor((totalSeconds % 3600) / 60);
    if (hours > 0) {
        return `${hours}h ${minutes}m`;
    }
    return `${minutes}m`;
}

const CallPostEnded = ({startAt, endAt}: Props) => {
    const intl = useIntl();

    const endedAt = new Date(endAt).toLocaleTimeString([], {
        hour: '2-digit',
        minute: '2-digit',
    });
    const duration = formatDuration(endAt - startAt);

    return (
        <>
            <div style={{display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '8px'}}>
                <span
                    style={{
                        width: '10px',
                        height: '10px',
                        borderRadius: '50%',
                        background: 'var(--center-channel-color-40)',
                        display: 'inline-block',
                    }}
                    data-testid='call-post-status-indicator'
                />
                <span
                    style={{fontWeight: 600}}
                    data-testid='call-post-label'
                >
                    {intl.formatMessage({id: 'plugin.rtk.call_post.label_ended'})}
                </span>
                <span
                    style={{color: 'var(--center-channel-color-56)', fontSize: '12px'}}
                    data-testid='call-post-end-time'
                >
                    {intl.formatMessage(
                        {id: 'plugin.rtk.call_post.ended_at'},
                        {time: endedAt},
                    )}
                </span>
            </div>
            <div
                style={{color: 'var(--center-channel-color-56)', fontSize: '12px'}}
                data-testid='call-post-duration'
            >
                {intl.formatMessage(
                    {id: 'plugin.rtk.call_post.duration'},
                    {duration},
                )}
            </div>
        </>
    );
};

export default CallPostEnded;
