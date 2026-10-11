// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package credential

import (
	"context"
)

// ResolveSupportedIdentities reads provider account metadata without resolving
// tokens or calling user_info. Command-tree construction and local discovery can
// apply strict mode without authenticating a user. Account selection is shared
// with ResolveAccount so later platform requests keep the same provider.
func (p *CredentialProvider) ResolveSupportedIdentities(ctx context.Context) (uint8, error) {
	account, _, err := p.resolveAccountMetadata(ctx)
	if err != nil {
		return 0, err
	}
	if account == nil {
		return 0, nil
	}
	return account.SupportedIdentities, nil
}

func (p *CredentialProvider) resolveAccountMetadata(ctx context.Context) (*Account, credentialSource, error) {
	p.metadataOnce.Do(func() {
		p.metadataAccount, p.metadataSource, p.metadataErr = p.selectAccount(ctx)
	})
	return p.metadataAccount, p.metadataSource, p.metadataErr
}
