//go:build !darwin && !windows

package keystore

import "arion.local/desktop/internal/account"

func (Store) Put(string) error     { return account.ErrUnsupported }
func (Store) Get() (string, error) { return "", account.ErrUnsupported }
func (Store) Delete() error        { return account.ErrUnsupported }
