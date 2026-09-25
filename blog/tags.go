package blog

// The cache tags pages declare for what they read from the CMS, and a webhook
// invalidates.
const (
	TagPosts      = "blog:posts"
	TagCategories = "blog:categories"
	TagTags       = "blog:tags"
	TagAuthors    = "blog:authors"
)

// TagPost is the tag of the page showing one post.
func TagPost(slug string) string { return "blog:post:" + slug }
