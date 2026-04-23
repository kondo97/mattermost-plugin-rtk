// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {useIntl} from 'react-intl';
import {useSelector} from 'react-redux';
import {
    selectCallByChannel,
    selectIsCurrentUserParticipant,
    selectPluginEnabled,
} from 'redux/selectors';

import type {Channel} from '@mattermost/types/channels';

interface Props {
    channel: Channel;
    currentUserId: string;
}

/**
 * Dropdown variant of the channel header call button.
 * Shown when multiple call plugins are registered (e.g., alongside Mattermost Calls).
 * Matches the Calls plugin's dropdown layout: icon + label + sublabel.
 */
const ChannelHeaderDropdownButton = ({channel, currentUserId}: Props) => {
    const intl = useIntl();

    const pluginEnabled = useSelector(selectPluginEnabled);
    const activeCall = useSelector(selectCallByChannel(channel.id));
    const isParticipant = useSelector(selectIsCurrentUserParticipant(channel.id, currentUserId));

    if (!pluginEnabled) {
        return null;
    }

    let label: string;

    if (isParticipant) {
        label = intl.formatMessage({id: 'plugin.rtk.channel_header.in_call'});
    } else if (activeCall) {
        label = intl.formatMessage({id: 'plugin.rtk.channel_header.join_call'});
    } else {
        label = intl.formatMessage({id: 'plugin.rtk.channel_header.start_call'});
    }

    const sublabel = intl.formatMessage({id: 'plugin.rtk.channel_header.in_this_channel'});

    return (
        <button
            className='style--none call-button-dropdown'
            data-testid='channel-header-call-dropdown-button'
        >
            <i
                className='icon icon-phone'
                style={{padding: 0}}
            />
            <div>
                <span className='call-button-label'>{label}</span>
                <span className='call-button-dropdown-sublabel'>{sublabel}</span>
            </div>
        </button>
    );
};

export default ChannelHeaderDropdownButton;
