package reddit

import (
	"context"
	"errors"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Pagination", func() {
	var (
		ctx context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
	})

	Describe("PaginateAll", func() {
		Context("with a simple fetch function", func() {
			var (
				pages [][]string
				calls []string
			)

			BeforeEach(func() {
				pages = [][]string{
					{"item1", "item2", "item3"},
					{"item4", "item5"},
					{"item6"},
				}
				calls = []string{}
			})

			It("should fetch all pages when no limit is set", func() {
				fetchPage := func(ctx context.Context, after string) ([]string, string, error) {
					calls = append(calls, after)

					pageIndex := len(calls) - 1
					if pageIndex >= len(pages) {
						return []string{}, "", nil
					}

					page := pages[pageIndex]
					nextAfter := ""
					if pageIndex < len(pages)-1 {
						nextAfter = fmt.Sprintf("after_page_%d", pageIndex+1)
					}

					return page, nextAfter, nil
				}

				opts := PaginationOptions{
					Limit:       0, // No limit
					PageSize:    100,
					StopOnEmpty: true,
				}

				result, err := PaginateAll[string](ctx, fetchPage, opts)

				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal([]string{"item1", "item2", "item3", "item4", "item5", "item6"}))
				Expect(calls).To(Equal([]string{"", "after_page_1", "after_page_2"}))
			})

			It("should respect the limit parameter", func() {
				fetchPage := func(ctx context.Context, after string) ([]string, string, error) {
					calls = append(calls, after)

					pageIndex := len(calls) - 1
					if pageIndex >= len(pages) {
						return []string{}, "", nil
					}

					page := pages[pageIndex]
					nextAfter := fmt.Sprintf("after_page_%d", pageIndex+1)

					return page, nextAfter, nil
				}

				opts := PaginationOptions{
					Limit:       4, // Stop after 4 items
					PageSize:    100,
					StopOnEmpty: true,
				}

				result, err := PaginateAll[string](ctx, fetchPage, opts)

				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal([]string{"item1", "item2", "item3", "item4"}))
				Expect(calls).To(Equal([]string{"", "after_page_1"}))
			})

			It("should stop on empty pages when StopOnEmpty is true", func() {
				fetchPage := func(ctx context.Context, after string) ([]string, string, error) {
					calls = append(calls, after)

					if len(calls) == 1 {
						return []string{"item1", "item2"}, "after_page_1", nil
					}
					if len(calls) == 2 {
						return []string{}, "after_page_2", nil // Empty page with after token
					}

					return []string{"item3"}, "", nil // This shouldn't be reached
				}

				opts := PaginationOptions{
					Limit:       0,
					PageSize:    100,
					StopOnEmpty: true,
				}

				result, err := PaginateAll[string](ctx, fetchPage, opts)

				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal([]string{"item1", "item2"}))
				Expect(calls).To(Equal([]string{"", "after_page_1"}))
			})

			It("should continue on empty pages when StopOnEmpty is false", func() {
				fetchPage := func(ctx context.Context, after string) ([]string, string, error) {
					calls = append(calls, after)

					if len(calls) == 1 {
						return []string{"item1", "item2"}, "after_page_1", nil
					}
					if len(calls) == 2 {
						return []string{}, "after_page_2", nil // Empty page with after token
					}
					if len(calls) == 3 {
						return []string{"item3"}, "", nil // Final page
					}

					return []string{}, "", nil
				}

				opts := PaginationOptions{
					Limit:       0,
					PageSize:    100,
					StopOnEmpty: false,
				}

				result, err := PaginateAll[string](ctx, fetchPage, opts)

				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal([]string{"item1", "item2", "item3"}))
				Expect(calls).To(Equal([]string{"", "after_page_1", "after_page_2"}))
			})

			It("should handle errors from fetch function", func() {
				fetchPage := func(ctx context.Context, after string) ([]string, string, error) {
					calls = append(calls, after)

					if len(calls) == 1 {
						return []string{"item1", "item2"}, "after_page_1", nil
					}

					return nil, "", errors.New("fetch error")
				}

				opts := DefaultPaginationOptions()

				result, err := PaginateAll[string](ctx, fetchPage, opts)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fetch error"))
				Expect(result).To(BeNil())
				Expect(calls).To(Equal([]string{"", "after_page_1"}))
			})

			It("should handle context cancellation", func() {
				cancelCtx, cancel := context.WithCancel(ctx)

				fetchPage := func(ctx context.Context, after string) ([]string, string, error) {
					calls = append(calls, after)

					// Cancel context immediately after first call
					if len(calls) == 1 {
						defer cancel() // Cancel after this function returns
						return []string{"item1", "item2"}, "after_page_1", nil
					}

					return []string{"item3", "item4"}, "", nil
				}

				opts := DefaultPaginationOptions()

				result, err := PaginateAll[string](cancelCtx, fetchPage, opts)

				Expect(err).To(Equal(context.Canceled))
				Expect(result).To(BeNil())
				Expect(len(calls)).To(Equal(1)) // Only one call should have been made
			})

			It("should return error when fetchPage is nil", func() {
				opts := DefaultPaginationOptions()

				result, err := PaginateAll[string](ctx, nil, opts)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fetchPage function is required"))
				Expect(result).To(BeNil())
			})
		})
	})

	Describe("PaginateAfter", func() {
		Context("with afterItem specified", func() {
			type TestItem struct {
				ID   string
				Name string
			}

			It("should use the extracted after token for the first request", func() {
				var calls []string

				fetchPage := func(ctx context.Context, after string) ([]TestItem, string, error) {
					calls = append(calls, after)

					if len(calls) == 1 {
						return []TestItem{{ID: "2", Name: "item2"}, {ID: "3", Name: "item3"}}, "test_3", nil
					}

					return []TestItem{{ID: "4", Name: "item4"}}, "", nil
				}

				extractAfter := func(item TestItem) string {
					return "test_" + item.ID
				}

				afterItem := &TestItem{ID: "1", Name: "item1"}
				opts := DefaultPaginationOptions()

				result, err := PaginateAfter[TestItem](ctx, fetchPage, extractAfter, afterItem, opts)

				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal([]TestItem{{ID: "2", Name: "item2"}, {ID: "3", Name: "item3"}, {ID: "4", Name: "item4"}}))
				Expect(calls).To(Equal([]string{"test_1", "test_3"}))
			})

			It("should handle nil afterItem", func() {
				var calls []string

				fetchPage := func(ctx context.Context, after string) ([]TestItem, string, error) {
					calls = append(calls, after)
					return []TestItem{{ID: "1", Name: "item1"}}, "", nil
				}

				extractAfter := func(item TestItem) string {
					return "test_" + item.ID
				}

				opts := DefaultPaginationOptions()

				result, err := PaginateAfter[TestItem](ctx, fetchPage, extractAfter, nil, opts)

				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal([]TestItem{{ID: "1", Name: "item1"}}))
				Expect(calls).To(Equal([]string{""}))
			})

			It("should return error when fetchPage is nil", func() {
				extractAfter := func(item TestItem) string {
					return "test_" + item.ID
				}

				afterItem := &TestItem{ID: "1", Name: "item1"}
				opts := DefaultPaginationOptions()

				result, err := PaginateAfter[TestItem](ctx, nil, extractAfter, afterItem, opts)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fetchPage function is required"))
				Expect(result).To(BeNil())
			})

			It("should return error when extractAfter is nil", func() {
				fetchPage := func(ctx context.Context, after string) ([]TestItem, string, error) {
					return []TestItem{{ID: "1", Name: "item1"}}, "", nil
				}

				afterItem := &TestItem{ID: "1", Name: "item1"}
				opts := DefaultPaginationOptions()

				result, err := PaginateAfter[TestItem](ctx, fetchPage, nil, afterItem, opts)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("extractAfter function is required"))
				Expect(result).To(BeNil())
			})
		})
	})

	Describe("PaginateSingle", func() {
		Context("fetching a single page", func() {
			It("should return a single page of results", func() {
				fetchPage := func(ctx context.Context, after string) ([]string, string, error) {
					if after == "" {
						return []string{"item1", "item2", "item3"}, "next_token", nil
					}
					return []string{"item4", "item5"}, "", nil
				}

				result, err := PaginateSingle[string](ctx, fetchPage, "")

				Expect(err).ToNot(HaveOccurred())
				Expect(result.Items).To(Equal([]string{"item1", "item2", "item3"}))
				Expect(result.After).To(Equal("next_token"))
			})

			It("should handle fetch errors", func() {
				fetchPage := func(ctx context.Context, after string) ([]string, string, error) {
					return nil, "", errors.New("single page error")
				}

				result, err := PaginateSingle[string](ctx, fetchPage, "")

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("single page error"))
				Expect(result).To(BeNil())
			})

			It("should return error when fetchPage is nil", func() {
				result, err := PaginateSingle[string](ctx, nil, "")

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fetchPage function is required"))
				Expect(result).To(BeNil())
			})
		})
	})

	Describe("DefaultPaginationOptions", func() {
		It("should return sensible defaults", func() {
			opts := DefaultPaginationOptions()

			Expect(opts.Limit).To(Equal(0))
			Expect(opts.PageSize).To(Equal(100))
			Expect(opts.StopOnEmpty).To(BeTrue())
		})
	})

	Describe("Integration with Reddit-specific types", func() {
		Context("with Post-like structures", func() {
			It("should work with Post-like pagination", func() {
				type MockPost struct {
					ID    string
					Title string
				}

				fullnameFunc := func(p MockPost) string {
					return "t3_" + p.ID
				}
				var calls []string

				fetchPage := func(ctx context.Context, after string) ([]MockPost, string, error) {
					calls = append(calls, after)

					if len(calls) == 1 {
						return []MockPost{
							{ID: "post1", Title: "First Post"},
							{ID: "post2", Title: "Second Post"},
						}, "t3_post2", nil
					}

					return []MockPost{
						{ID: "post3", Title: "Third Post"},
					}, "", nil
				}

				extractAfter := func(post MockPost) string {
					return fullnameFunc(post)
				}

				afterPost := &MockPost{ID: "post0", Title: "Starting Post"}
				opts := PaginationOptions{
					Limit:       3,
					PageSize:    100,
					StopOnEmpty: true,
				}

				result, err := PaginateAfter[MockPost](ctx, fetchPage, extractAfter, afterPost, opts)

				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(3))
				Expect(result[0].ID).To(Equal("post1"))
				Expect(result[1].ID).To(Equal("post2"))
				Expect(result[2].ID).To(Equal("post3"))
				Expect(calls).To(Equal([]string{"t3_post0", "t3_post2"}))
			})
		})

		Context("with Comment-like structures", func() {
			It("should work with Comment-like pagination", func() {
				type MockComment struct {
					ID   string
					Body string
				}

				fullnameFunc := func(c MockComment) string {
					return "t1_" + c.ID
				}
				var calls []string

				fetchPage := func(ctx context.Context, after string) ([]MockComment, string, error) {
					calls = append(calls, after)

					if len(calls) == 1 {
						return []MockComment{
							{ID: "comment1", Body: "First comment"},
							{ID: "comment2", Body: "Second comment"},
						}, fullnameFunc(MockComment{ID: "comment2", Body: "Second comment"}), nil
					}

					return []MockComment{}, "", nil // No more comments
				}

				opts := DefaultPaginationOptions()

				result, err := PaginateAll[MockComment](ctx, fetchPage, opts)

				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(2))
				Expect(result[0].ID).To(Equal("comment1"))
				Expect(result[1].ID).To(Equal("comment2"))
				Expect(calls).To(Equal([]string{"", fullnameFunc(MockComment{ID: "comment2", Body: "Second comment"})}))
			})
		})
	})
})
