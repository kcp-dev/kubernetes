/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package crdb

import (
	"strings"
	"errors"

	"github.com/jackc/pgconn"
	kerrors "k8s.io/apimachinery/pkg/api/errors"

	etcdrpc "go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

func interpretWatchError(err error) error {
	switch {
	case err == etcdrpc.ErrCompacted:
		return kerrors.NewResourceExpired("The resourceVersion for the provided watch is too old.")
	}
	return err
}

const (
	expired         string = "The resourceVersion for the provided list is too old."
	continueExpired string = "The provided continue parameter is too old " +
		"to display a consistent list result. You can start a new list without " +
		"the continue parameter."
	inconsistentContinue string = "The provided continue parameter is too old " +
		"to display a consistent list result. You can start a new list without " +
		"the continue parameter, or use the continue token in this response to " +
		"retrieve the remainder of the results. Continuing with the provided " +
		"token results in an inconsistent list - objects that were created, " +
		"modified, or deleted between the time the first chunk was returned " +
		"and now may show up in the list."
)

func interpretListError(err error, paging bool, continueKey, keyPrefix string) error {
	switch {
	case isGarbageCollectionError(err):
		if paging {
			return handleCompactedErrorForPaging(continueKey, keyPrefix)
		}
		return kerrors.NewResourceExpired(expired)
	}
	return err
}

func handleCompactedErrorForPaging(continueKey, keyPrefix string) error {
	// ContinueToken.ResoureVersion=-1 means that the apiserver can
	// continue the list at the latest resource version. We don't use rv=0
	// for this purpose to distinguish from a bad token that has empty rv.
	newToken, err := EncodeContinue(continueKey, keyPrefix, -1)
	if err != nil {
		utilruntime.HandleError(err)
		return kerrors.NewResourceExpired(continueExpired)
	}
	statusError := kerrors.NewResourceExpired(inconsistentContinue)
	statusError.ErrStatus.ListMeta.Continue = newToken
	return statusError
}

func isGarbageCollectionError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "XXUUU" && strings.Contains(pgErr.Message, "must be after replica GC threshold") {
			return true
		}
	}
	return false
}