// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {pluginFetch} from 'client';
import {useEffect} from 'react';
import {useDispatch, useSelector} from 'react-redux';
import {setChannelEnabled} from 'redux/calls_slice';
import {selectChannelEnabled} from 'redux/selectors';

type ChannelStateResponse = {
    channel_id: string;
    enabled: boolean;
};

// useChannelEnabled returns the cached enabled state for a channel. When the
// state is not yet present in Redux it triggers a single `GET /api/v1/channels/{id}`
// fetch and caches the result.
//
// Returns:
//   - true  : calls are enabled (or no row exists; default-enabled is treated as true here)
//   - false : calls are explicitly disabled
//   - undefined : not yet known (initial render before fetch resolves)
export function useChannelEnabled(channelId: string): boolean | undefined {
    const dispatch = useDispatch();
    const channelEnabled = useSelector(selectChannelEnabled(channelId));

    useEffect(() => {
        if (!channelId || channelEnabled !== undefined) {
            return;
        }
        pluginFetch<ChannelStateResponse>(`/api/v1/channels/${channelId}`).then((result) => {
            if ('data' in result) {
                dispatch(setChannelEnabled(channelId, result.data.enabled));
            }
        });
    }, [channelId, channelEnabled, dispatch]);

    return channelEnabled;
}
