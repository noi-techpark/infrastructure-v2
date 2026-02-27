// SPDX-FileCopyrightText: 2024 NOI Techpark <digital@noi.bz.it>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/noi-techpark/opendatahub-go-sdk/ingest/urn"
)

type readyPayload struct {
	ID         string `json:"id"`
	DB         string `json:"db"`
	Collection string `json:"collection"`
	URN        string `json:"urn"`
}

func publishReady(ctx context.Context, m meta, wr writeResult) error {
	u, ok := urn.RawUrnFromProviderURI(m.Provider)
	if !ok {
		return fmt.Errorf("could not build URN from provider %q", m.Provider)
	}
	if err := u.AddNSS(wr.ID); err != nil {
		return fmt.Errorf("could not add NSS to URN: %w", err)
	}

	payload, err := json.Marshal(readyPayload{
		ID:         wr.ID,
		DB:         wr.DB,
		Collection: wr.Coll,
		URN:        u.String(),
	})
	if err != nil {
		return fmt.Errorf("marshal ready payload: %w", err)
	}

	return readyPublisher.Publish(ctx, payload, "")
}
