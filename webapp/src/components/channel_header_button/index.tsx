// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {useIntl} from 'react-intl';
import {useSelector} from 'react-redux';
import {
    selectCallByChannel,
    selectCallLoading,
    selectIsCurrentUserParticipant,
    selectPluginEnabled,
} from 'redux/selectors';

import type {Channel} from '@mattermost/types/channels';

interface Props {
    channel: Channel;
    currentUserId: string;
}

/**
 * Display-only channel header button.
 * Click handling is done by the `fn` argument of registerCallButtonAction in index.tsx.
 * Mattermost wraps this component in a <button onClick={fn}>, so we must NOT render
 * an inner <button> (nested buttons cause click events to be lost).
 */
const ChannelHeaderButton = ({channel, currentUserId}: Props) => {
    const intl = useIntl();

    const pluginEnabled = useSelector(selectPluginEnabled);
    const activeCall = useSelector(selectCallByChannel(channel.id));
    const isParticipant = useSelector(selectIsCurrentUserParticipant(channel.id, currentUserId));
    const loading = useSelector(selectCallLoading);

    if (!pluginEnabled) {
        return null;
    }

    let label: string;
    let tooltip: string;

    if (loading) {
        label = intl.formatMessage({id: 'plugin.rtk.channel_header.starting_call'});
        tooltip = label;
    } else if (isParticipant) {
        label = intl.formatMessage({id: 'plugin.rtk.channel_header.in_call'});
        tooltip = intl.formatMessage({id: 'plugin.rtk.channel_header.tooltip_in_call'});
    } else if (activeCall) {
        label = intl.formatMessage({id: 'plugin.rtk.channel_header.join_call'});
        tooltip = intl.formatMessage({id: 'plugin.rtk.channel_header.tooltip_join'});
    } else {
        label = intl.formatMessage({id: 'plugin.rtk.channel_header.start_call'});
        tooltip = intl.formatMessage({id: 'plugin.rtk.channel_header.tooltip_start'});
    }

    return (
        <button
            className='style--none call-button'
            title={tooltip}
            aria-label={tooltip}
            data-testid='channel-header-call-button'
        >
            <i
                className={loading ? 'icon icon-loading icon-spin' : 'icon icon-phone'}
                style={{padding: 0}}
            />
            <span
                className='call-button-label'
                data-testid='channel-header-call-button-label'
            >
                {label}
            </span>
        </button>
    );
};

export default ChannelHeaderButton;
