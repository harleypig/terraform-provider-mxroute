package provider

import (
	"context"
	"net/http"
)

// fetchOne GETs a single object of type T from path and decodes it. A
// NOT_FOUND is reported as (nil, nil) — the signal a resource's Read uses
// to drop the resource from state — so callers distinguish "absent" from a
// real error without repeating the IsNotFound check.
func fetchOne[T any](ctx context.Context, c *Client, path string) (*T, error) {
	var api T

	if err := c.Do(ctx, http.MethodGet, path, nil, &api); err != nil {
		if IsNotFound(err) {
			return nil, nil
		}

		return nil, err
	}

	return &api, nil
}

// fetchFromList GETs a []T from path (an endpoint with no single-object GET)
// and returns the first element for which match reports true, or (nil, nil)
// when none matches or the list endpoint returns NOT_FOUND.
func fetchFromList[T any](ctx context.Context, c *Client, path string, match func(*T) bool) (*T, error) {
	var list []T

	if err := c.Do(ctx, http.MethodGet, path, nil, &list); err != nil {
		if IsNotFound(err) {
			return nil, nil
		}

		return nil, err
	}

	for i := range list {
		if match(&list[i]) {
			return &list[i], nil
		}
	}

	return nil, nil
}
