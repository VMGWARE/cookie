package server

import (
	"slices"

	"github.com/discuitnet/discuit/internal/httperr"
	"github.com/discuitnet/discuit/internal/meilisearch"
)

//	@Summary		Search
//	@Description	Search for content in the MeiliSearch indexes.
//	@Router			/api/search [GET]
//	@Success		200
//	@Tags			Search
//	@Param			q		query	string	true	"Search query"
//	@Param			index	query	string	true	"Index to search in"	Enums(communities,posts,users)
//	@Param			sort	query	string	false	"Sort results by"
func (s *Server) search(w *responseWriter, r *request) error {
	if !s.config.MeiliEnabled {
		return httperr.NewBadRequest("meili_disabled", "MeiliSearch is disabled.")
	}

	query := r.urlQueryParams()
	r.req.ParseForm()

	// query
	q := query.Get("q")

	// sort
	sort := r.req.Form["sort"]

	// index
	index := query.Get("index")
	if index == "" {
		return httperr.NewBadRequest("missing_index", "Missing index.")
	}

	searchClient := meilisearch.NewSearchClient(s.config.MeiliHost, s.config.MeiliKey)

	if !slices.Contains(meilisearch.ValidIndexes, index) {
		return httperr.NewBadRequest("invalid_index", "Invalid index.")
	}

	results, err := searchClient.Search(index, q, sort)
	if err != nil {
		return httperr.NewBadRequest("bad_request", err.Error())
	}

	return w.writeJSON(results)
}
