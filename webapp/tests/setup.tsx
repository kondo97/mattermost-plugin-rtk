// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import Adapter from '@wojtekmaj/enzyme-adapter-react-17';
import Enzyme from 'enzyme';

Enzyme.configure({adapter: new Adapter()});

// structuredClone is not available in older jsdom/Node versions used by Jest.
// @cloudflare/realtimekit-ui calls it at module init time.
if (typeof globalThis.structuredClone === 'undefined') {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (globalThis as any).structuredClone = (obj: unknown) => JSON.parse(JSON.stringify(obj));
}
