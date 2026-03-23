// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// Standalone call page entry point.
// This file is bundled as call.js and served from server/assets/call.js.
// It has NO dependency on the Mattermost webapp framework.

import React from 'react';
import ReactDOM from 'react-dom/client';

import CallPage from './CallPage';

// Parse URL parameters (Pattern U4-2)
const params = new URLSearchParams(window.location.search);
const token = params.get('token') ?? '';
const callId = params.get('call_id') ?? '';
const channelName = decodeURIComponent(params.get('channel_name') ?? '');

// Set browser tab title (BR-U4-008, US-006)
document.title = channelName ? `Call in #${channelName}` : 'RTK Call';

// Mount the call page — error screen rendered inside CallPage if token is missing
const rootEl = document.getElementById('root') ?? document.body;
ReactDOM.createRoot(rootEl).render(
    <CallPage
        token={token}
        callId={callId}
    />,
);
