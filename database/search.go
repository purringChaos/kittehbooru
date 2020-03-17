package database

import (
	"context"
	"math"
	"runtime/trace"
	"sort"
	"strings"
	"github.com/NamedKitten/kittehbooru/types"
	"github.com/NamedKitten/kittehbooru/utils"
	"github.com/bwmarrin/snowflake"
)

// paginate paginates a list of int64s
func paginate(x []int64, page int, pageSize int) []int64 {
	var limit int
	var start int
	numItems := len(x)
	// skips forward N items where N is how many items in a page multiplied by the page number
	skip := pageSize * page
	// prevents integer overflow if skip becomes negative
	if skip <= 0 {
		skip = 0
	}
	if skip > numItems {
		start = numItems
	} else {
		start = skip
	}
	if skip+pageSize > numItems {
		limit = numItems
	} else {
		limit = skip + pageSize
	}
	return x[start:limit]
}

// searchTag is a wrapper around AllPostIDs and TagPosts
// for searching for all results for a tag or a wildcard match.
func (db *DB) searchTag(ctx context.Context, tag string) (posts []int64) {
	defer trace.StartRegion(ctx, "DB/searchTag").End()

	var err error
	if val, ok := db.SearchCache.Get(ctx, tag); ok {
		posts = val
	} else {
		if tag == "*" {
			posts, err = db.AllPostIDs(ctx)
		} else {
			posts, err = db.TagPosts(ctx, tag)
		}
	}
	if err == nil {
		db.SearchCache.Add(ctx, tag, posts)
	}
	return
}

// getPostsForTags gets posts matching tags from DB
// it uses a tags table which maps a tag to all the posts containing a tag
func (db *DB) getPostsForTags(ctx context.Context, tags []string) []int64 {
	defer trace.StartRegion(ctx, "DB/getPostsForTags").End()
	// we need to make sure to keep track of how many times the post
	// is seen and only get which posts appear for all of the positive posts
	// basically a simple way of getting the intersection of all positive tags
	// so that we only get the posts that match ALL of the positive tags
	posCount := 0
	posCounts := make(map[int64]int)
	negMatch := make(map[int64]bool)

	tags = db.filterTags(tags)

	tagsPosts, _ := db.TagsPosts(ctx, tags)

	for _, tag := range tags {
		// is it a positive tag or a negative tag
		// true = positive, false = negative
		is := !strings.HasPrefix(tag, "-")

		if !is {
			// remove the - at start
			tag = tag[1:]
		} else {
			// increase the count of positive tags
			posCount += 1
		}

		// posts will be all the posts that are tagged with `tag`
		posts := tagsPosts[tag]
		if len(posts) == 0 && is {
			// Return early if it is a positive tag with no match.
			// This saves time on searches where a tag has no match because
			// we don't need to process any other tags after this one.
			return []int64{}
		}
		for _, post := range posts {
			if !is {
				// if its a negative match, aka post we DONT want, add it to this map instead
				negMatch[post] = true
			} else if i, ok := posCounts[post]; ok {
				// add to counter of positive counts
				posCounts[post] = i + 1
			} else {
				// add the count to map starting at 1 if not existing already
				posCounts[post] = 1
			}
		}
	}

	finalPostIDs := make([]int64, 0)

	for posPost, posCountTimes := range posCounts {
		// so we only get the posts that match ALL positive tags
		if posCountTimes == posCount {
			found := false
			for negPost := range negMatch {
				// if there is a post that is a negative match, do not add this to the finalPostIDs array
				if posPost == negPost {
					found = true
				}
			}
			if !found {
				finalPostIDs = append(finalPostIDs, posPost)
			}
		}
	}

	sort.Slice(finalPostIDs, func(i, j int) bool {
		return snowflake.ID(finalPostIDs[i]).Time() > snowflake.ID(finalPostIDs[j]).Time()
	})
	return finalPostIDs
}

// TopNCommonTags returns the top N common tags for a search of tags
func (db *DB) TopNCommonTags(ctx context.Context, n int, tags []string, individualTags bool) []types.TagCounts {
	defer trace.StartRegion(ctx, "DB/Top15CommonTags").End()

	combinedTags := utils.TagsListToString(tags)
	if val, ok := db.TagCountsCache.Get(ctx, combinedTags); ok {
		return val
	}
	var postsArray []int64
	if individualTags {
		posts := make(map[int64]bool)
		for _, tag := range tags {
			p := db.cacheSearch(ctx, []string{tag})
			for _, pid := range p {
				posts[pid] = true
			} 
		}
		postsArray = make([]int64, 0)
		for pid, _ := range posts {
			postsArray = append(postsArray, pid)
		}
	} else {
		postsArray = db.cacheSearch(ctx, tags)
	}


	tagCounts, _ := db.PostsTagsCounts(ctx, postsArray)

	tagCountsSlice := make([]types.TagCounts, 0, len(tagCounts))
	for k, v := range tagCounts {
		tagCountsSlice = append(tagCountsSlice, types.TagCounts{k, v})
	}


	sort.Slice(tagCountsSlice, func(i, j int) bool {
		if tagCountsSlice[i].Count ==  tagCountsSlice[j].Count {
			if strings.HasPrefix(tagCountsSlice[i].Tag, "user:") && strings.HasPrefix(tagCountsSlice[j].Tag, "user:") {
				return tagCountsSlice[i].Tag < tagCountsSlice[j].Tag
			} else if strings.HasPrefix(tagCountsSlice[i].Tag, "user:") {
				return true
			} else if strings.HasPrefix(tagCountsSlice[j].Tag, "user:") {
				return false
			} else {
				return tagCountsSlice[i].Tag < tagCountsSlice[j].Tag
			}
			return tagCountsSlice[i].Tag < tagCountsSlice[j].Tag
		} else {
			return tagCountsSlice[i].Count > tagCountsSlice[j].Count
		}
	})

	// Calculate the min between how many tags there are and N
	// Prevents panic when N > tag count
	x := math.Min(float64(n), float64(len(tagCountsSlice)))
	result := tagCountsSlice[:int(x)]

	//db.TagCountsCache.Add(ctx, combinedTags, result)
	return result
}

// cacheSearch searches for posts matching tags and returns a
// array of post IDs matching those tags.
func (db *DB) cacheSearch(ctx context.Context, searchTags []string) []int64 {
	defer trace.StartRegion(ctx, "DB/cacheSearch").End()

	var result []int64
	searchTags = db.filterTags(searchTags)
	combinedTags := utils.TagsListToString(searchTags)
	// If it is in the cache then great! use the cached result
	// otherise search for them and add to the cache.
	if val, ok := db.SearchCache.Get(ctx, combinedTags); ok {
		result = val
	} else {
		matching := db.getPostsForTags(ctx, searchTags)
		db.SearchCache.Add(ctx, combinedTags, matching)
		result = matching
	}
	// Sort by posted time
	// TODO add a parameter to have different sorting modes such as sorting from oldest to newest
	// or from filesize.
	sort.Slice(result, func(i, j int) bool {
		return snowflake.ID(result[i]).Time() > snowflake.ID(result[j]).Time()
	})
	return result
}

// GetSearchIDs returns a paginated list of Post IDs from a list of tags.
func (db *DB) GetSearchIDs(ctx context.Context, searchTags []string, page int) []int64 {
	defer trace.StartRegion(ctx, "DB/GetSearchIDs").End()

	matching := db.cacheSearch(ctx, searchTags)
	return paginate(matching, page, 20)
}

// getSearchPage returns a paginated list of posts from a list of tags.
func (db *DB) GetSearchPage(ctx context.Context, searchTags []string, page int) []types.Post {
	defer trace.StartRegion(ctx, "DB/GetSearchPage").End()

	matching := db.cacheSearch(ctx, searchTags)
	pageContent := paginate(matching, page, 20)
	posts, err := db.Posts(ctx, pageContent)
	if err != nil {
		return posts
	}
	return posts
}
