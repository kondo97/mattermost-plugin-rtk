// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import type {GlobalState} from '@mattermost/types/store';

/**
 * Builds the URL for opening the standalone call tab.
 *
 * Includes call_id and channel_name as URL parameters so the call page
 * can run its heartbeat loop and display the correct browser tab title.
 *
 * Token is JWT-safe and does not need additional encoding.
 * call_id and channel_name are encoded to prevent URL injection (BR-U4-019).
 */
export function buildCallTabUrl(
    pluginId: string,
    token: string,
    callId: string,
    channelName: string,
): string {
    const encodedCallId = encodeURIComponent(callId);
    const encodedChannelName = encodeURIComponent(channelName);
    return `/plugins/${pluginId}/call?token=${token}&call_id=${encodedCallId}&channel_name=${encodedChannelName}`;
}

/**
 * Resolves the display name for a channel from the Mattermost Redux state.
 * Returns an empty string if the channel is not found.
 */
export function getChannelDisplayName(state: GlobalState, channelId: string): string {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    return (state as unknown as any).entities?.channels?.channels?.[channelId]?.display_name ?? '';
}
