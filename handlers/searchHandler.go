package handlers

import (
	"net/http"
	"strconv"

	"github.com/NamedKitten/kittehbooru/types"
	"github.com/NamedKitten/kittehbooru/i18n"
	templates "github.com/NamedKitten/kittehbooru/template"
	"github.com/NamedKitten/kittehbooru/utils"
	"github.com/rs/zerolog/log"
)

// SearchResultsTemplate contains data to be used in the template.
type SearchResultsTemplate struct {
	// The posts that match the search for a page.
	Results []int64
	// RealPage is the real page number for the current page.
	RealPage int
	// Page is RealPage + 1 and is used to show a 1-based page number index.
	Page int
	// TotalPages is the total number of pages for a given search query
	TotalPages int
	// Next is the next page
	Next int
	// Prev is the previous page
	Prev int
	// NumPosts is the total number of posts for a search query
	NumPosts int
	// Tags is the tags from the search query args, used to refill
	// the search bar.
	Tags string

	TagCounts []types.TagCounts
	templates.T
}

// searchHandler is the search endpoint used for displaying results
// of a search query.
func SearchHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !DB.SetupCompleted {
		http.Redirect(w, r, "/setup", http.StatusFound)
		return
	}
	user, loggedIn := DB.CheckForLoggedInUser(ctx, r)
	tagsStr := r.URL.Query().Get("tags")
	if len(tagsStr) == 0 {
		tagsStr = "*"
	}
	tags := utils.SplitTagsString(tagsStr)
	pageStr := r.URL.Query().Get("page")
	if len(pageStr) == 0 {
		pageStr = "0"
	}
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		log.Error().Err(err).Msg("Can't convert pageStr to string")
		return
	}
	matchingPosts := DB.GetSearchIDs(ctx, tags, page)
	var prevPage int
	if page <= 0 {
		prevPage = 0
	} else {
		prevPage = page - 1
	}

	searchResults := SearchResultsTemplate{
		Results:    matchingPosts,
		RealPage:   page,
		Page:       page + 1,
		NumPosts:   DB.NumOfPostsForTags(ctx, tags),
		TotalPages: DB.NumOfPagesForTags(ctx, tags),
		Next:       page + 1,
		Prev:       prevPage,
		Tags:       tagsStr,
		TagCounts: DB.TopNCommonTags(ctx, 30, tags, false),

		T: templates.T{
			LoggedIn:     loggedIn,
			LoggedInUser: user,
			Translator:   i18n.GetTranslator(r),
		},
	}

	err = templates.RenderTemplate(w, "search.html", searchResults)
	if err != nil {
		log.Error().Err(err).Msg("Render Search")
		return
	}
}
