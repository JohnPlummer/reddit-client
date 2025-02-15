package reddit

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// mockPostGetter implements PostGetter for testing
type mockPostGetter struct {
	posts     []Post
	nextPage  string
	err       error
	callCount int
}

func (m *mockPostGetter) GetPosts(subreddit string, params map[string]string) ([]Post, string, error) {
	if m.err != nil {
		return nil, "", m.err
	}

	// For pagination test
	if m.callCount == 0 {
		m.callCount++
		return m.posts, m.nextPage, nil
	}
	// Second call returns different posts with no next page
	return []Post{{Title: "Post 2", ID: "2"}}, "", nil
}

var _ = Describe("Subreddit", func() {
	Describe("GetPosts", func() {
		It("fetches posts from a subreddit", func() {
			mockPosts := []Post{
				{Title: "Post 1", ID: "1"},
				{Title: "Post 2", ID: "2"},
			}
			client := &mockPostGetter{posts: mockPosts}

			subreddit := Subreddit{Name: "golang"}
			posts, err := subreddit.GetPosts(client, "new", 2)

			Expect(err).NotTo(HaveOccurred())
			Expect(posts).To(Equal(mockPosts))
		})

		It("handles pagination", func() {
			firstPage := []Post{{Title: "Post 1", ID: "1"}}
			client := &mockPostGetter{
				posts:    firstPage,
				nextPage: "t3_123",
			}

			subreddit := Subreddit{Name: "golang"}
			posts, err := subreddit.GetPosts(client, "new", 2)

			Expect(err).NotTo(HaveOccurred())
			Expect(posts).To(HaveLen(2))
			Expect(posts[0].Title).To(Equal("Post 1"))
			Expect(posts[1].Title).To(Equal("Post 2"))
		})

		It("handles errors", func() {
			client := &mockPostGetter{err: fmt.Errorf("API error")}

			subreddit := Subreddit{Name: "golang"}
			_, err := subreddit.GetPosts(client, "new", 2)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("API error"))
		})
	})
})
