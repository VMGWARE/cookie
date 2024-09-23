package server

// file named feeds.go as we already have feed.go but that's for a user feed of subs, etc...

import (
	"fmt"
	"log"

	"github.com/discuitnet/discuit/core"
	"github.com/gorilla/feeds"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"github.com/microcosm-cc/bluemonday"
)

// // /api/feed/u/{username}
// func (s *Server) getUserFeed(w *responseWriter, r *request) error {
// 	// var (
// 	// 	username      = r.muxVar("username")
// 	// 	query         = r.urlQueryParams()
// 	// 	feedResultSet *core.UserFeedResultSet
// 	// 	err           error
// 	// )

// 	return nil
// }

// @Summary		Get community feed
// @Description	Get the feed of a community
// @Router			/api/feed/c/{communityName} [GET]
// @Success		200
// @Tags			Feed
// @Param			communityName	path	string	true	"Community name"
// @Param			type			query	string	false	"Feed type (atom, rss, json)"	Enums(atom, rss, json)	Default(rss)
func (s *Server) getCommunityFeed(w *responseWriter, r *request) error {
	var (
		communityName = r.muxVar("communityName")
		query         = r.urlQueryParams()
		comm          *core.Community
		set           *core.FeedResultSet
		feedResponse  string
		err           error
	)

	comm, err = core.GetCommunityByName(r.ctx, s.db, communityName, r.viewer)
	if err != nil {
		return err
	}

	set, err = core.GetFeed(r.ctx, s.db, &core.FeedOptions{
		Sort:        core.FeedSortLatest,
		DefaultSort: false,
		Viewer:      r.viewer,
		Community:   &comm.ID,
		Homefeed:    false,
		Limit:       50,
		Next:        "",
	})
	if err != nil {
		return err
	}

	commPublicUrl := fmt.Sprintf("%s/%s%s", s.config.PublicUrl, s.config.CommunityPrefix, comm.Name)

	feed := &feeds.Feed{
		Title:       comm.Name + " Feed - " + s.config.SiteName,
		Link:        &feeds.Link{Href: commPublicUrl},
		Description: comm.About.String,
		Id:          comm.ID.String(),
	}

	if comm.ProPic != nil {
		feed.Image = &feeds.Image{
			Url:   fmt.Sprintf("%s%s", s.config.PublicUrl, *comm.ProPic.URL),
			Title: comm.Name,
			Link:  commPublicUrl,
		}
	} else {
		feed.Image = &feeds.Image{
			Url:   fmt.Sprintf("%s/favicon.png", s.config.PublicUrl),
			Title: comm.Name,
			Link:  commPublicUrl,
		}
	}

	feed.Items = []*feeds.Item{}

	for _, item := range set.Posts {
		itemPublicUrl := fmt.Sprintf("%s/%s%s/post/%s", s.config.PublicUrl, s.config.CommunityPrefix, comm.Name, item.PublicID)
		var author string
		if item.AuthorDeleted || item.AuthorGhostID != "" {
			fmt.Println(item.AuthorUsername, item.AuthorDeleted, item.AuthorGhostID)
			author = "Unknown"
		} else if item.AuthorUsername != "" {
			author = item.AuthorUsername
		} else {
			author = "Unknown"
		}

		fi := &feeds.Item{
			Title:       item.Title,
			Link:        &feeds.Link{Href: itemPublicUrl},
			Author:      &feeds.Author{Name: author},
			Created:     item.CreatedAt,
			Id:          item.PublicID,
			Description: "",
		}

		switch item.Type {
		case core.PostTypeImage:
			fi.Title = fi.Title + " (Image)"
			fi.Content = fi.Content + fmt.Sprintf(`<br><img src="%s" alt="Image" />`, *item.Image.URL)
		case core.PostTypeLink:
			fi.Title = fi.Title + " (Link)"
			fi.Content = fi.Content + fmt.Sprintf(`<br>Submitted link: <a href="%s">%s</a>`, item.Link.URL, item.Link.Hostname)
		case core.PostTypeText:
			html := mdToHTML([]byte(item.Body.String))
			fi.Content = string(html)
		}

		// Credits: https://github.com/ttaylor-st/discuit-rss/blob/master/src/index.ts
		fi.Description = fi.Description + `<br><br>`
		fi.Description = fi.Description + fmt.Sprintf(`%d upvotes, %d downvotes, %d overall`, item.Upvotes, item.Downvotes, (item.Upvotes-item.Downvotes))
		fi.Description = fi.Description + fmt.Sprintf(`<br><a href="%s">View Post</a>`, itemPublicUrl)
		fi.Description = fi.Description + fmt.Sprintf(` • <a href="%s">View %d`, itemPublicUrl, item.NumComments)
		if item.NumComments == 1 {
			fi.Description = fi.Description + " comment"
		} else {
			fi.Description = fi.Description + " comments"
		}
		fi.Description = fi.Description + `</a>`
		fi.Description = fi.Description + fmt.Sprintf(` • Posted by <a href="%s/@%s">@%s</a>`, s.config.PublicUrl, author, author)

		if item.Type == core.PostTypeText {
			fi.Content = fi.Content + fmt.Sprintf(`<br><br>%s`, fi.Description)
		} else {
			fi.Content = fi.Content + fi.Description
		}
		fi.Description = ""

		feed.Items = append(feed.Items, fi)
	}

	var feedContentType string

	// Switch based on query params.
	switch query.Get("type") {
	case "atom":
		feedContentType = "application/atom+xml"
		feedResponse, err = feed.ToAtom()
		if err != nil {
			log.Fatal(err)
		}
	case "rss":
		feedContentType = "application/rss+xml"
		feedResponse, err = feed.ToRss()
		if err != nil {
			log.Fatal(err)
		}
	case "json":
		feedContentType = "application/json"
		feedResponse, err = feed.ToJSON()
		if err != nil {
			log.Fatal(err)
		}
	default:
		feedContentType = "application/rss+xml"
		feedResponse, err = feed.ToRss()
		if err != nil {
			log.Fatal(err)
		}
	}

	w.Header().Del("Content-Type")
	w.Header().Add("Content-Type", fmt.Sprintf("%s; charset=UTF-8", feedContentType))
	return w.writeString(feedResponse)
}

func mdToHTML(md []byte) []byte {
	// create markdown parser with extensions
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse(md)

	// create HTML renderer with extensions
	htmlFlags := html.CommonFlags | html.HrefTargetBlank | html.NofollowLinks | html.NoopenerLinks
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	maybeUnsafeHTML := markdown.Render(doc, renderer)
	return bluemonday.UGCPolicy().SanitizeBytes(maybeUnsafeHTML)
}
