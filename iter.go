package okx

import (
	"context"
	"iter"
)

// paginate yields items page by page. fetch receives the last item of the
// previous page as the cursor; iteration stops on an empty page or error.
func paginate[T any](ctx context.Context, fetch func(cursor T, hasCursor bool) ([]T, error)) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		var cursor T
		hasCursor := false
		for {
			if err := ctx.Err(); err != nil {
				var zero T
				yield(zero, err)
				return
			}
			page, err := fetch(cursor, hasCursor)
			if err != nil {
				var zero T
				yield(zero, err)
				return
			}
			if len(page) == 0 {
				return
			}
			for _, item := range page {
				if !yield(item, nil) {
					return
				}
			}
			cursor, hasCursor = page[len(page)-1], true
		}
	}
}
