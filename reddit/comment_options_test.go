package reddit_test

import (
	"strconv"

	"github.com/JohnPlummer/reddit-client/reddit"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Comment Options", func() {
	var params map[string]string

	BeforeEach(func() {
		params = make(map[string]string)
	})

	Describe("WithCommentSort", func() {
		It("sets the sort parameter with valid sort options", func() {
			sortOptions := []string{"confidence", "top", "new", "controversial", "old", "random", "qa", "live"}

			for _, sort := range sortOptions {
				params = make(map[string]string) // Reset params for each test
				option := reddit.WithCommentSort(sort)
				option(params)

				Expect(params).To(HaveKeyWithValue("sort", sort))
			}
		})

		It("sets custom sort parameter", func() {
			option := reddit.WithCommentSort("custom_sort")
			option(params)

			Expect(params).To(HaveKeyWithValue("sort", "custom_sort"))
		})

		It("does not set parameter when sort is empty string", func() {
			option := reddit.WithCommentSort("")
			option(params)

			Expect(params).To(BeEmpty())
		})

		It("overwrites existing sort parameter", func() {
			params["sort"] = "old_value"
			option := reddit.WithCommentSort("new_value")
			option(params)

			Expect(params).To(HaveKeyWithValue("sort", "new_value"))
		})

		It("handles sort with special characters", func() {
			option := reddit.WithCommentSort("sort-with_special.chars")
			option(params)

			Expect(params).To(HaveKeyWithValue("sort", "sort-with_special.chars"))
		})

		It("preserves other parameters when setting sort", func() {
			params["existing_param"] = "existing_value"
			option := reddit.WithCommentSort("confidence")
			option(params)

			Expect(params).To(HaveKeyWithValue("sort", "confidence"))
			Expect(params).To(HaveKeyWithValue("existing_param", "existing_value"))
		})
	})

	Describe("WithCommentAfter", func() {
		It("sets the after parameter using comment fullname", func() {
			comment := &reddit.Comment{
				ID:     "test123",
				Author: "testuser",
				Body:   "test comment",
			}

			option := reddit.WithCommentAfter(comment)
			option(params)

			Expect(params).To(HaveKeyWithValue("after", "t1_test123"))
		})

		It("does not set parameter when comment is nil", func() {
			option := reddit.WithCommentAfter(nil)
			option(params)

			Expect(params).To(BeEmpty())
		})

		It("handles comment with empty ID", func() {
			comment := &reddit.Comment{
				ID:     "",
				Author: "testuser",
				Body:   "test comment",
			}

			option := reddit.WithCommentAfter(comment)
			option(params)

			Expect(params).To(HaveKeyWithValue("after", "t1_"))
		})

		It("handles comment with special characters in ID", func() {
			comment := &reddit.Comment{
				ID:     "abc_123-xyz",
				Author: "testuser",
				Body:   "test comment",
			}

			option := reddit.WithCommentAfter(comment)
			option(params)

			Expect(params).To(HaveKeyWithValue("after", "t1_abc_123-xyz"))
		})

		It("overwrites existing after parameter", func() {
			params["after"] = "old_value"
			comment := &reddit.Comment{
				ID: "new123",
			}

			option := reddit.WithCommentAfter(comment)
			option(params)

			Expect(params).To(HaveKeyWithValue("after", "t1_new123"))
		})

		It("preserves other parameters when setting after", func() {
			params["existing_param"] = "existing_value"
			comment := &reddit.Comment{
				ID: "test456",
			}

			option := reddit.WithCommentAfter(comment)
			option(params)

			Expect(params).To(HaveKeyWithValue("after", "t1_test456"))
			Expect(params).To(HaveKeyWithValue("existing_param", "existing_value"))
		})
	})

	Describe("WithCommentLimit", func() {
		It("sets the limit parameter for positive values", func() {
			testCases := []int{1, 25, 100, 500, 1000}

			for _, limit := range testCases {
				params = make(map[string]string) // Reset params for each test
				option := reddit.WithCommentLimit(limit)
				option(params)

				Expect(params).To(HaveKeyWithValue("limit", strconv.Itoa(limit)))
			}
		})

		It("does not set parameter for zero limit", func() {
			option := reddit.WithCommentLimit(0)
			option(params)

			Expect(params).To(BeEmpty())
		})

		It("does not set parameter for negative limit", func() {
			option := reddit.WithCommentLimit(-10)
			option(params)

			Expect(params).To(BeEmpty())
		})

		It("handles very large limit values", func() {
			option := reddit.WithCommentLimit(999999)
			option(params)

			Expect(params).To(HaveKeyWithValue("limit", "999999"))
		})

		It("overwrites existing limit parameter", func() {
			params["limit"] = "old_value"
			option := reddit.WithCommentLimit(50)
			option(params)

			Expect(params).To(HaveKeyWithValue("limit", "50"))
		})

		It("preserves other parameters when setting limit", func() {
			params["existing_param"] = "existing_value"
			option := reddit.WithCommentLimit(100)
			option(params)

			Expect(params).To(HaveKeyWithValue("limit", "100"))
			Expect(params).To(HaveKeyWithValue("existing_param", "existing_value"))
		})
	})

	Describe("WithCommentDepth", func() {
		It("sets the depth parameter for positive values", func() {
			testCases := []int{1, 5, 10, 15, 20}

			for _, depth := range testCases {
				params = make(map[string]string) // Reset params for each test
				option := reddit.WithCommentDepth(depth)
				option(params)

				Expect(params).To(HaveKeyWithValue("depth", strconv.Itoa(depth)))
			}
		})

		It("does not set parameter for zero depth", func() {
			option := reddit.WithCommentDepth(0)
			option(params)

			Expect(params).To(BeEmpty())
		})

		It("does not set parameter for negative depth", func() {
			option := reddit.WithCommentDepth(-5)
			option(params)

			Expect(params).To(BeEmpty())
		})

		It("handles very large depth values", func() {
			option := reddit.WithCommentDepth(100)
			option(params)

			Expect(params).To(HaveKeyWithValue("depth", "100"))
		})

		It("overwrites existing depth parameter", func() {
			params["depth"] = "old_value"
			option := reddit.WithCommentDepth(8)
			option(params)

			Expect(params).To(HaveKeyWithValue("depth", "8"))
		})

		It("preserves other parameters when setting depth", func() {
			params["existing_param"] = "existing_value"
			option := reddit.WithCommentDepth(5)
			option(params)

			Expect(params).To(HaveKeyWithValue("depth", "5"))
			Expect(params).To(HaveKeyWithValue("existing_param", "existing_value"))
		})
	})

	Describe("WithCommentContext", func() {
		It("sets the context parameter for positive values", func() {
			testCases := []int{1, 3, 5, 8, 10}

			for _, context := range testCases {
				params = make(map[string]string) // Reset params for each test
				option := reddit.WithCommentContext(context)
				option(params)

				Expect(params).To(HaveKeyWithValue("context", strconv.Itoa(context)))
			}
		})

		It("does not set parameter for zero context", func() {
			option := reddit.WithCommentContext(0)
			option(params)

			Expect(params).To(BeEmpty())
		})

		It("does not set parameter for negative context", func() {
			option := reddit.WithCommentContext(-3)
			option(params)

			Expect(params).To(BeEmpty())
		})

		It("handles large context values", func() {
			option := reddit.WithCommentContext(50)
			option(params)

			Expect(params).To(HaveKeyWithValue("context", "50"))
		})

		It("overwrites existing context parameter", func() {
			params["context"] = "old_value"
			option := reddit.WithCommentContext(7)
			option(params)

			Expect(params).To(HaveKeyWithValue("context", "7"))
		})

		It("preserves other parameters when setting context", func() {
			params["existing_param"] = "existing_value"
			option := reddit.WithCommentContext(3)
			option(params)

			Expect(params).To(HaveKeyWithValue("context", "3"))
			Expect(params).To(HaveKeyWithValue("existing_param", "existing_value"))
		})
	})

	Describe("WithCommentShowMore", func() {
		It("sets show_more parameter to true", func() {
			option := reddit.WithCommentShowMore(true)
			option(params)

			Expect(params).To(HaveKeyWithValue("show_more", "true"))
		})

		It("sets show_more parameter to false", func() {
			option := reddit.WithCommentShowMore(false)
			option(params)

			Expect(params).To(HaveKeyWithValue("show_more", "false"))
		})

		It("overwrites existing show_more parameter", func() {
			params["show_more"] = "old_value"
			option := reddit.WithCommentShowMore(true)
			option(params)

			Expect(params).To(HaveKeyWithValue("show_more", "true"))
		})

		It("preserves other parameters when setting show_more", func() {
			params["existing_param"] = "existing_value"
			option := reddit.WithCommentShowMore(false)
			option(params)

			Expect(params).To(HaveKeyWithValue("show_more", "false"))
			Expect(params).To(HaveKeyWithValue("existing_param", "existing_value"))
		})
	})

	Describe("Multiple Options Combined", func() {
		It("applies multiple comment options correctly", func() {
			comment := &reddit.Comment{
				ID: "test789",
			}

			options := []reddit.CommentOption{
				reddit.WithCommentSort("confidence"),
				reddit.WithCommentAfter(comment),
				reddit.WithCommentLimit(50),
				reddit.WithCommentDepth(5),
				reddit.WithCommentContext(3),
				reddit.WithCommentShowMore(true),
			}

			for _, option := range options {
				option(params)
			}

			Expect(params).To(HaveKeyWithValue("sort", "confidence"))
			Expect(params).To(HaveKeyWithValue("after", "t1_test789"))
			Expect(params).To(HaveKeyWithValue("limit", "50"))
			Expect(params).To(HaveKeyWithValue("depth", "5"))
			Expect(params).To(HaveKeyWithValue("context", "3"))
			Expect(params).To(HaveKeyWithValue("show_more", "true"))
		})

		It("handles conflicting options correctly (last one wins)", func() {
			options := []reddit.CommentOption{
				reddit.WithCommentSort("confidence"),
				reddit.WithCommentSort("top"),
				reddit.WithCommentLimit(25),
				reddit.WithCommentLimit(100),
			}

			for _, option := range options {
				option(params)
			}

			Expect(params).To(HaveKeyWithValue("sort", "top"))
			Expect(params).To(HaveKeyWithValue("limit", "100"))
		})

		It("ignores invalid options while applying valid ones", func() {
			comment := &reddit.Comment{
				ID: "valid123",
			}

			options := []reddit.CommentOption{
				reddit.WithCommentSort("confidence"), // valid
				reddit.WithCommentAfter(comment),     // valid
				reddit.WithCommentLimit(0),           // invalid (zero)
				reddit.WithCommentDepth(-1),          // invalid (negative)
				reddit.WithCommentContext(5),         // valid
				reddit.WithCommentShowMore(true),     // valid
			}

			for _, option := range options {
				option(params)
			}

			Expect(params).To(HaveKeyWithValue("sort", "confidence"))
			Expect(params).To(HaveKeyWithValue("after", "t1_valid123"))
			Expect(params).NotTo(HaveKey("limit")) // Should be omitted
			Expect(params).NotTo(HaveKey("depth")) // Should be omitted
			Expect(params).To(HaveKeyWithValue("context", "5"))
			Expect(params).To(HaveKeyWithValue("show_more", "true"))
		})
	})
})
