// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// Minimal stub for react-intl (host-provided external; not installed as npm dep).
// Individual test files override this via jest.mock('react-intl', ...) as needed.

const useIntl = () => ({
    formatMessage: ({id}) => id,
    locale: 'en',
});

const FormattedMessage = ({id}) => id;
FormattedMessage.displayName = 'FormattedMessage';

const IntlProvider = ({children}) => children;
IntlProvider.displayName = 'IntlProvider';

const defineMessages = (msgs) => msgs;

module.exports = {
    useIntl,
    FormattedMessage,
    IntlProvider,
    defineMessages,
};
