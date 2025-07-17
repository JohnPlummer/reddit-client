package reddit

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Utils", func() {
	Describe("getStringField", func() {
		It("should extract string field successfully", func() {
			data := map[string]any{
				"test_field": "test_value",
			}
			result := getStringField(data, "test_field")
			Expect(result).To(Equal("test_value"))
		})

		It("should return empty string for missing field", func() {
			data := map[string]any{}
			result := getStringField(data, "missing_field")
			Expect(result).To(Equal(""))
		})

		It("should return default value for missing field", func() {
			data := map[string]any{}
			result := getStringField(data, "missing_field", "default")
			Expect(result).To(Equal("default"))
		})

		It("should return empty string for non-string field", func() {
			data := map[string]any{
				"test_field": 123,
			}
			result := getStringField(data, "test_field")
			Expect(result).To(Equal(""))
		})

		It("should return default value for non-string field", func() {
			data := map[string]any{
				"test_field": 123,
			}
			result := getStringField(data, "test_field", "default")
			Expect(result).To(Equal("default"))
		})
	})

	Describe("getFloat64Field", func() {
		It("should extract float64 field successfully", func() {
			data := map[string]any{
				"test_field": 123.45,
			}
			result := getFloat64Field(data, "test_field")
			Expect(result).To(Equal(123.45))
		})

		It("should convert float32 to float64", func() {
			data := map[string]any{
				"test_field": float32(123.45),
			}
			result := getFloat64Field(data, "test_field")
			Expect(result).To(BeNumerically("~", 123.45, 0.01))
		})

		It("should convert int to float64", func() {
			data := map[string]any{
				"test_field": 123,
			}
			result := getFloat64Field(data, "test_field")
			Expect(result).To(Equal(123.0))
		})

		It("should convert int64 to float64", func() {
			data := map[string]any{
				"test_field": int64(123),
			}
			result := getFloat64Field(data, "test_field")
			Expect(result).To(Equal(123.0))
		})

		It("should parse string as float64", func() {
			data := map[string]any{
				"test_field": "123.45",
			}
			result := getFloat64Field(data, "test_field")
			Expect(result).To(Equal(123.45))
		})

		It("should return 0 for missing field", func() {
			data := map[string]any{}
			result := getFloat64Field(data, "missing_field")
			Expect(result).To(Equal(0.0))
		})

		It("should return default value for missing field", func() {
			data := map[string]any{}
			result := getFloat64Field(data, "missing_field", 456.78)
			Expect(result).To(Equal(456.78))
		})

		It("should return 0 for invalid string", func() {
			data := map[string]any{
				"test_field": "not_a_number",
			}
			result := getFloat64Field(data, "test_field")
			Expect(result).To(Equal(0.0))
		})

		It("should return default value for invalid string", func() {
			data := map[string]any{
				"test_field": "not_a_number",
			}
			result := getFloat64Field(data, "test_field", 999.0)
			Expect(result).To(Equal(999.0))
		})
	})

	Describe("getBoolField", func() {
		It("should extract bool field successfully", func() {
			data := map[string]any{
				"test_field": true,
			}
			result := getBoolField(data, "test_field")
			Expect(result).To(BeTrue())
		})

		It("should parse string as bool", func() {
			data := map[string]any{
				"true_field":  "true",
				"false_field": "false",
				"1_field":     "1",
				"0_field":     "0",
			}
			Expect(getBoolField(data, "true_field")).To(BeTrue())
			Expect(getBoolField(data, "false_field")).To(BeFalse())
			Expect(getBoolField(data, "1_field")).To(BeTrue())
			Expect(getBoolField(data, "0_field")).To(BeFalse())
		})

		It("should convert int to bool", func() {
			data := map[string]any{
				"nonzero_field": 123,
				"zero_field":    0,
			}
			Expect(getBoolField(data, "nonzero_field")).To(BeTrue())
			Expect(getBoolField(data, "zero_field")).To(BeFalse())
		})

		It("should convert float64 to bool", func() {
			data := map[string]any{
				"nonzero_field": 123.45,
				"zero_field":    0.0,
			}
			Expect(getBoolField(data, "nonzero_field")).To(BeTrue())
			Expect(getBoolField(data, "zero_field")).To(BeFalse())
		})

		It("should return false for missing field", func() {
			data := map[string]any{}
			result := getBoolField(data, "missing_field")
			Expect(result).To(BeFalse())
		})

		It("should return default value for missing field", func() {
			data := map[string]any{}
			result := getBoolField(data, "missing_field", true)
			Expect(result).To(BeTrue())
		})

		It("should return false for invalid string", func() {
			data := map[string]any{
				"test_field": "not_a_bool",
			}
			result := getBoolField(data, "test_field")
			Expect(result).To(BeFalse())
		})
	})

	Describe("getIntField", func() {
		It("should convert float64 to int", func() {
			data := map[string]any{
				"test_field": 123.78,
			}
			result := getIntField(data, "test_field")
			Expect(result).To(Equal(123))
		})

		It("should return default value for missing field", func() {
			data := map[string]any{}
			result := getIntField(data, "missing_field", 456)
			Expect(result).To(Equal(456))
		})

		It("should return 0 for missing field without default", func() {
			data := map[string]any{}
			result := getIntField(data, "missing_field")
			Expect(result).To(Equal(0))
		})
	})

	Describe("getInt64Field", func() {
		It("should convert float64 to int64", func() {
			data := map[string]any{
				"test_field": 123.78,
			}
			result := getInt64Field(data, "test_field")
			Expect(result).To(Equal(int64(123)))
		})

		It("should return default value for missing field", func() {
			data := map[string]any{}
			result := getInt64Field(data, "missing_field", int64(456))
			Expect(result).To(Equal(int64(456)))
		})
	})

	Describe("getValidatedIntField", func() {
		It("should return value when validation passes", func() {
			data := map[string]any{
				"test_field": 123.0,
			}
			validator := func(v int) bool { return v >= 0 }
			result := getValidatedIntField(data, "test_field", validator)
			Expect(result).To(Equal(123))
		})

		It("should return 0 when validation fails", func() {
			data := map[string]any{
				"test_field": -123.0,
			}
			validator := func(v int) bool { return v >= 0 }
			result := getValidatedIntField(data, "test_field", validator)
			Expect(result).To(Equal(0))
		})

		It("should return default when validation fails", func() {
			data := map[string]any{
				"test_field": -123.0,
			}
			validator := func(v int) bool { return v >= 0 }
			result := getValidatedIntField(data, "test_field", validator, 999)
			Expect(result).To(Equal(999))
		})
	})

	Describe("parsePostData", func() {
		It("should parse valid post data", func() {
			data := map[string]any{
				"id":           "test_id",
				"title":        "Test Title",
				"selftext":     "Test content",
				"url":          "https://example.com",
				"created_utc":  1234567890.0,
				"subreddit":    "test_subreddit",
				"score":        100.0,
				"num_comments": 50.0,
			}

			post, err := parsePostData(data)
			Expect(err).NotTo(HaveOccurred())
			Expect(post.ID).To(Equal("test_id"))
			Expect(post.Title).To(Equal("Test Title"))
			Expect(post.SelfText).To(Equal("Test content"))
			Expect(post.URL).To(Equal("https://example.com"))
			Expect(post.Created).To(Equal(int64(1234567890)))
			Expect(post.Subreddit).To(Equal("test_subreddit"))
			Expect(post.RedditScore).To(Equal(100))
			Expect(post.CommentCount).To(Equal(50))
			Expect(post.ContentScore).To(Equal(0))
		})

		It("should return error for missing ID", func() {
			data := map[string]any{
				"title": "Test Title",
			}

			_, err := parsePostData(data)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("missing required field 'id'"))
		})

		It("should handle missing optional fields", func() {
			data := map[string]any{
				"id": "test_id",
			}

			post, err := parsePostData(data)
			Expect(err).NotTo(HaveOccurred())
			Expect(post.ID).To(Equal("test_id"))
			Expect(post.Title).To(Equal(""))
			Expect(post.SelfText).To(Equal(""))
			Expect(post.URL).To(Equal(""))
			Expect(post.Created).To(Equal(int64(0)))
			Expect(post.Subreddit).To(Equal(""))
			Expect(post.RedditScore).To(Equal(0))
			Expect(post.CommentCount).To(Equal(0))
		})

		It("should validate comment count is non-negative", func() {
			data := map[string]any{
				"id":           "test_id",
				"num_comments": -5.0,
			}

			post, err := parsePostData(data)
			Expect(err).NotTo(HaveOccurred())
			Expect(post.CommentCount).To(Equal(0)) // Should default to 0 for negative values
		})
	})

	Describe("parseCommentData", func() {
		It("should parse valid comment data", func() {
			data := map[string]any{
				"id":          "comment_id",
				"author":      "test_user",
				"body":        "Test comment body",
				"created_utc": 1234567890.0,
			}
			ingestedAt := int64(9876543210)

			comment, err := parseCommentData(data, ingestedAt)
			Expect(err).NotTo(HaveOccurred())
			Expect(comment.ID).To(Equal("comment_id"))
			Expect(comment.Author).To(Equal("test_user"))
			Expect(comment.Body).To(Equal("Test comment body"))
			Expect(comment.Created).To(Equal(int64(1234567890)))
			Expect(comment.IngestedAt).To(Equal(ingestedAt))
		})

		It("should return error for missing ID", func() {
			data := map[string]any{
				"author": "test_user",
				"body":   "Test comment body",
			}
			ingestedAt := int64(9876543210)

			_, err := parseCommentData(data, ingestedAt)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("missing required field 'id'"))
		})

		It("should handle missing optional fields", func() {
			data := map[string]any{
				"id": "comment_id",
			}
			ingestedAt := int64(9876543210)

			comment, err := parseCommentData(data, ingestedAt)
			Expect(err).NotTo(HaveOccurred())
			Expect(comment.ID).To(Equal("comment_id"))
			Expect(comment.Author).To(Equal(""))
			Expect(comment.Body).To(Equal(""))
			Expect(comment.Created).To(Equal(int64(0)))
			Expect(comment.IngestedAt).To(Equal(ingestedAt))
		})
	})
})
