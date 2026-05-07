// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// Note: the Jest config maps i18n JSON files to a shared empty mock
// (`tests/i18n_mock.json`), so we can only assert the fallback behaviour
// here. The actual translation strings are validated at build time and
// in the running plugin.
import {getCallErrorMessage} from './error_messages';

describe('getCallErrorMessage', () => {
    it('returns the fallback when no code is supplied', () => {
        expect(getCallErrorMessage(undefined, 'oops')).toBe('oops');
    });

    it('returns the fallback when the code is unknown', () => {
        expect(getCallErrorMessage('something_else', 'oops')).toBe('oops');
    });

    it('returns the fallback when the locale dictionary is empty (i18n mock)', () => {
        // Verifies the lookup chain resolves to the fallback rather than
        // throwing or returning undefined when no translation exists.
        expect(getCallErrorMessage('calls_disabled', 'fallback')).toBe('fallback');
    });
});
