// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {useIntl} from 'react-intl';

interface Props {
    visible: boolean;
    onConfirm: () => void;
    onCancel: () => void;
}

const SwitchCallModal = ({visible, onConfirm, onCancel}: Props) => {
    const intl = useIntl();

    if (!visible) {
        return null;
    }

    return (
        <div
            className='modal fade in'
            style={{display: 'block', background: 'rgba(0,0,0,0.5)'}}
            data-testid='switch-call-modal'
        >
            <div className='modal-dialog'>
                <div className='modal-content'>
                    <div className='modal-header'>
                        <h4 className='modal-title'>
                            {intl.formatMessage({id: 'plugin.rtk.switch_call_modal.title'})}
                        </h4>
                    </div>
                    <div className='modal-body'>
                        <p>{intl.formatMessage({id: 'plugin.rtk.switch_call_modal.body'})}</p>
                    </div>
                    <div className='modal-footer'>
                        <button
                            type='button'
                            className='btn btn-link'
                            onClick={onCancel}
                            data-testid='switch-call-modal-cancel'
                        >
                            {intl.formatMessage({id: 'plugin.rtk.switch_call_modal.cancel'})}
                        </button>
                        <button
                            type='button'
                            className='btn btn-primary'
                            onClick={onConfirm}
                            data-testid='switch-call-modal-confirm'
                        >
                            {intl.formatMessage({id: 'plugin.rtk.switch_call_modal.confirm'})}
                        </button>
                    </div>
                </div>
            </div>
        </div>
    );
};

export default SwitchCallModal;
