// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {useChannelEnabled} from 'hooks/use_channel_enabled';
import React from 'react';
import {useIntl} from 'react-intl';
import {useSelector} from 'react-redux';

import type {GlobalState} from '@mattermost/types/store';

// ChannelMenuAction renders the "Disable calls (RTK)" / "Enable calls (RTK)"
// item in the channel's "Other actions" dropdown. Only channel admins, team
// admins, and system admins see this item (others get null).
//
// The current enabled state is resolved by the shared `useChannelEnabled`
// hook, which fetches it from the server on first use and caches it in Redux.
export function ChannelMenuAction() {
    const {formatMessage} = useIntl();

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const mmState = useSelector((s: GlobalState) => s) as any;
    const channelId: string = mmState.entities?.channels?.currentChannelId ?? '';
    const userId: string = mmState.entities?.users?.currentUserId ?? '';

    // Permission check: channel_admin or system_admin.
    const memberRoles: string = mmState.entities?.channels?.myMembers?.[channelId]?.roles ?? '';
    const userRoles: string = mmState.entities?.users?.profiles?.[userId]?.roles ?? '';
    const isAdmin =
        memberRoles.includes('channel_admin') ||
        userRoles.includes('system_admin');

    const channelEnabled = useChannelEnabled(channelId);

    if (!isAdmin) {
        return null;
    }

    const isDisabled = channelEnabled === false;

    return (
        <span>
            {formatMessage({
                id: isDisabled ? 'plugin.rtk.channel_header.enable_calls' : 'plugin.rtk.channel_header.disable_calls',
            })}
        </span>
    );
}

export default ChannelMenuAction;

