package server

import (
	"database/sql"
	"encoding/json"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/discuitnet/discuit/core"
	"github.com/discuitnet/discuit/internal/httperr"
	msql "github.com/discuitnet/discuit/internal/sql"
	"github.com/discuitnet/discuit/internal/uid"
)

// @Summary		Get user's lists.
// @Description	Get user's lists.
// @Router			/api/users/{username}/lists [GET]
// @Success		200
// @Tags			Lists
// @Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
// @Param			username		path	string	true	"Username"
// @Param			sort			query	string	false	"Sort lists by"			Enums(name,created,updated)
// @Param			filter			query	string	false	"Filter lists by type"	Enums(public,private)
func (s *Server) getUsersLists(w *responseWriter, r *request) error {
	var (
		username     = strings.ToLower(r.muxVar("username"))
		user         *core.User
		userIsViewer = false
		err          error
	)

	user, err = core.GetUserByUsername(r.ctx, s.db, username, r.viewer)
	if err != nil {
		return err
	}

	if r.loggedIn {
		if user.ID == *r.viewer {
			userIsViewer = true
		}
	}

	lists, err := core.GetUsersLists(r.ctx, s.db, user.ID, r.urlQueryParamsValue("sort"), r.urlQueryParamsValue("filter"))
	if err != nil {
		return err
	}
	if !userIsViewer {
		// Viewer is requesting the lists of someone else. Only show them the
		// public lists.
		public := make([]*core.List, 0, len(lists))
		for _, list := range lists {
			if list.Public {
				public = append(public, list)
			}
		}
		lists = public
	}
	return w.writeJSON(lists)
}

// @Summary		Create a new list.
// @Description	Create a new list.
// @Router			/api/users/{username}/lists [POST]
// @Success		200
// @Tags			Lists
// @Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
// @Param			username		path	string	true	"Username"
func (s *Server) createList(w *responseWriter, r *request) error {
	var (
		username     = strings.ToLower(r.muxVar("username"))
		user         *core.User
		userIsViewer = false
		err          error
	)

	user, err = core.GetUserByUsername(r.ctx, s.db, username, r.viewer)
	if err != nil {
		return err
	}

	if r.loggedIn {
		if user.ID == *r.viewer {
			userIsViewer = true
		}
	}

	// Create a new list.

	// Check permissions first.
	if !r.loggedIn {
		return errNotLoggedIn
	}
	if !userIsViewer {
		return httperr.NewForbidden("not-your-list", "Not your list.")
	}

	form := struct {
		Name        string          `json:"name"`
		DisplayName string          `json:"displayName"` // Optional field, defaults to Name.
		Description msql.NullString `json:"description"`
		Public      bool            `json:"public"` // Optional field, defaults to false.
	}{}
	if err := r.unmarshalJSONBody(&form); err != nil {
		return err
	}

	if form.Name == "" {
		return httperr.NewBadRequest("list-name-empty", "List name cannot be empty.")
	}
	if form.DisplayName == "" {
		form.DisplayName = form.Name
	}

	if err := s.rateLimit(r, "list_c_1_"+r.viewer.String(), time.Second*2, 1); err != nil {
		return err
	}

	if err := s.rateLimit(r, "list_c_2_"+r.viewer.String(), time.Hour*24, 100); err != nil {
		return err
	}

	if err := core.CreateList(r.ctx, s.db, *r.viewer, form.Name, form.DisplayName, form.Description, form.Public); err != nil {
		return err
	}

	lists, err := core.GetUsersLists(r.ctx, s.db, user.ID, r.urlQueryParamsValue("sort"), r.urlQueryParamsValue("filter"))
	if err != nil {
		return err
	}
	if !userIsViewer {
		// Viewer is requesting the lists of someone else. Only show them the
		// public lists.
		public := make([]*core.List, 0, len(lists))
		for _, list := range lists {
			if list.Public {
				public = append(public, list)
			}
		}
		lists = public
	}
	return w.writeJSON(lists)
}

func viewerListOwner(r *request, db *sql.DB, list *core.List) (bool, error) {
	if !r.loggedIn {
		return false, nil
	}

	viewer, err := core.GetUser(r.ctx, db, *r.viewer, r.viewer)
	if err != nil {
		return false, err
	}
	return viewer.ID == list.UserID, nil
}

// @Summary		Get a list.
// @Description	Get a list.
// @Router			/api/lists/{listId} [GET]
// @Success		200
// @Tags			Lists
// @Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
// @Param			listId			path	string	true	"List ID"
func (s *Server) getList(w *responseWriter, r *request, list *core.List) error {
	if !list.Public {
		owner, err := viewerListOwner(r, s.db, list)
		if err != nil {
			return err
		}
		if !owner {
			if !list.Public {
				return httperr.NewNotFound("list-not-found", "List not found.")

			} else {
				// Not a GET request.
				return httperr.NewForbidden("not-list-owner", "Not list owner.")
			}
		}
	}

	return w.writeJSON(list)
}

// @Summary		Update a list.
// @Description	Update a list.
// @Router			/api/lists/{listId} [PUT]
// @Success		200
// @Tags			Lists
// @Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
// @Param			listId			path	string	true	"List ID"
func (s *Server) updateList(w *responseWriter, r *request, list *core.List) error {
	if !r.loggedIn {
		return errNotLoggedIn
	}

	owner, err := viewerListOwner(r, s.db, list)
	if err != nil {
		return err
	}
	if !owner {
		if !list.Public {
			return httperr.NewNotFound("list-not-found", "List not found.")

		} else {
			// Not a GET request.
			return httperr.NewForbidden("not-list-owner", "Not list owner.")
		}
	}

	if err := s.rateLimit(r, "list_e_1_"+r.viewer.String(), time.Second*1, 1); err != nil {
		return err
	}

	data, err := io.ReadAll(r.req.Body)
	if err != nil {
		return err
	}
	if err := list.UnmarshalUpdatableFieldsJSON(data); err != nil {
		return err
	}
	if err := list.Update(r.ctx, s.db); err != nil {
		return err
	}

	return w.writeJSON(list)
}

// @Summary		Delete a list.
// @Description	Delete a list.
// @Router			/api/lists/{listId} [DELETE]
// @Success		200
// @Tags			Lists
// @Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
// @Param			listId			path	string	true	"List ID"
func (s *Server) deleteList(w *responseWriter, r *request, list *core.List) error {
	if !r.loggedIn {
		return errNotLoggedIn
	}

	owner, err := viewerListOwner(r, s.db, list)
	if err != nil {
		return err
	}
	if !owner {
		if !list.Public {
			return httperr.NewNotFound("list-not-found", "List not found.")

		} else {
			// Not a GET request.
			return httperr.NewForbidden("not-list-owner", "Not list owner.")
		}
	}

	if err := s.rateLimit(r, "list_e_1_"+r.viewer.String(), time.Second*1, 1); err != nil {
		return err
	}

	if err := list.Delete(r.ctx, s.db); err != nil {
		return err
	}

	return w.writeJSON(list)
}

func (s *Server) withListByName(f func(*responseWriter, *request, *core.List) error) handler {
	return handler(func(w *responseWriter, r *request) error {
		user, err := core.GetUserByUsername(r.ctx, s.db, r.muxVar("username"), nil)
		if err != nil {
			return err
		}

		list, err := core.GetListByName(r.ctx, s.db, user.ID, r.muxVar("listname"))
		if err != nil {
			return err
		}

		return f(w, r, list)
	})
}

func (s *Server) withListByID(f func(*responseWriter, *request, *core.List) error) handler {
	return handler(func(w *responseWriter, r *request) error {
		listID, err := strconv.Atoi(r.muxVar("listId"))
		if err != nil {
			return httperr.NewBadRequest("invalid-list-id", "Invalid list id.")
		}

		list, err := core.GetList(r.ctx, s.db, listID)
		if err != nil {
			return err
		}

		return f(w, r, list)
	})
}

// @Summary		Get list items.
// @Description	Get list items.
// @Router			/api/lists/{listId}/items [GET]
// @Success		200
// @Tags			Lists
// @Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
// @Param			listId			path	string	true	"List ID"
// @Param			limit			query	int		false	"Limit"
// @Param			next			query	string	false	"Next"
// @Param			sort			query	string	false	"Sort"	Enums(default,created,updated)
func (s *Server) getListItems(w *responseWriter, r *request, list *core.List) error {
	if !list.Public {
		owner, err := viewerListOwner(r, s.db, list)
		if err != nil {
			return err
		}
		if !owner {
			if !list.Public {
				return httperr.NewNotFound("list-not-found", "List not found.")

			} else {
				// Not a GET request.
				return httperr.NewForbidden("not-list-owner", "Not list owner.")
			}
		}
	}

	var (
		err   error
		limit int
		next  *string
		sort  core.ListItemsSort = core.ListOrderingDefault
	)

	if limit, err = r.urlQueryParamsValueInt("limit", 10); err != nil {
		return httperr.NewBadRequest("invalid-limit", "Invalid limit value.")
	}
	if nextString := r.urlQueryParamsValue("next"); nextString != "" {
		next = &nextString
	}
	if sortString := r.urlQueryParamsValue("sort"); sortString != "" {
		if err := sort.UnmarshalText([]byte(sortString)); err != nil {
			return err
		}
	}

	resultSet, err := core.GetListItems(r.ctx, s.db, list.ID, limit, sort, next, r.viewer)
	if err != nil {
		return err
	}
	return w.writeJSON(resultSet)
}

// @Summary		Add an item to a list.
// @Description	Add an item to a list.
// @Router			/api/lists/{listId}/items [POST]
// @Success		200
// @Tags			Lists
// @Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
// @Param			listId			path	string	true	"List ID"
func (s *Server) addItemToList(w *responseWriter, r *request, list *core.List) error {
	if !r.loggedIn {
		return errNotLoggedIn
	}

	owner, err := viewerListOwner(r, s.db, list)
	if err != nil {
		return err
	}
	if !owner {
		if !list.Public {
			return httperr.NewNotFound("list-not-found", "List not found.")

		} else {
			// Not a GET request.
			return httperr.NewForbidden("not-list-owner", "Not list owner.")
		}
	}

	if err := s.rateLimit(r, "l_item_c_1_"+r.viewer.String(), time.Second, 2); err != nil {
		return err
	}
	if err := s.rateLimit(r, "l_item_c_2_"+r.viewer.String(), time.Hour, 1000); err != nil {
		return err
	}

	form := struct {
		TargetType core.ContentType `json:"targetType"`
		TargetID   uid.ID           `json:"targetId"`
	}{
		TargetType: -1, // sentinel value
	}
	bytes, err := io.ReadAll(r.req.Body)
	if err != nil {
		return err
	}
	if len(bytes) > 0 {
		if err := json.Unmarshal(bytes, &form); err != nil {
			return httperr.NewBadRequest("", "Bad JSON body.")
		}
	}
	if err := list.AddItem(r.ctx, s.db, form.TargetType, form.TargetID); err != nil {
		return err
	}

	var (
		limit int
		next  *string
		sort  core.ListItemsSort = core.ListOrderingDefault
	)

	if limit, err = r.urlQueryParamsValueInt("limit", 10); err != nil {
		return httperr.NewBadRequest("invalid-limit", "Invalid limit value.")
	}
	if nextString := r.urlQueryParamsValue("next"); nextString != "" {
		next = &nextString
	}
	if sortString := r.urlQueryParamsValue("sort"); sortString != "" {
		if err := sort.UnmarshalText([]byte(sortString)); err != nil {
			return err
		}
	}

	resultSet, err := core.GetListItems(r.ctx, s.db, list.ID, limit, sort, next, r.viewer)
	if err != nil {
		return err
	}
	return w.writeJSON(resultSet)
}

// @Summary		Delete all items from a list.
// @Description	Delete all items from a list.
// @Router			/api/lists/{listId}/items [DELETE]
// @Success		200
// @Tags			Lists
// @Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
// @Param			listId			path	string	true	"List ID"
func (s *Server) deleteAllItemsFromAList(w *responseWriter, r *request, list *core.List) error {
	if !r.loggedIn {
		return errNotLoggedIn
	}

	owner, err := viewerListOwner(r, s.db, list)
	if err != nil {
		return err
	}
	if !owner {
		if !list.Public {
			return httperr.NewNotFound("list-not-found", "List not found.")

		} else {
			// Not a GET request.
			return httperr.NewForbidden("not-list-owner", "Not list owner.")
		}
	}

	form := struct {
		TargetType core.ContentType `json:"targetType"`
		TargetID   uid.ID           `json:"targetId"`
	}{
		TargetType: -1, // sentinel value
	}
	bytes, err := io.ReadAll(r.req.Body)
	if err != nil {
		return err
	}
	if len(bytes) > 0 {
		if err := json.Unmarshal(bytes, &form); err != nil {
			return httperr.NewBadRequest("", "Bad JSON body.")
		}
	}
	if form.TargetID.Zero() && form.TargetType == -1 {
		if err := list.DeleteAllItems(r.ctx, s.db); err != nil {
			return err
		}
	} else {
		if err := list.DeleteItem(r.ctx, s.db, form.TargetType, form.TargetID); err != nil {
			return err
		}
	}

	var (
		limit int
		next  *string
		sort  core.ListItemsSort = core.ListOrderingDefault
	)

	if limit, err = r.urlQueryParamsValueInt("limit", 10); err != nil {
		return httperr.NewBadRequest("invalid-limit", "Invalid limit value.")
	}
	if nextString := r.urlQueryParamsValue("next"); nextString != "" {
		next = &nextString
	}
	if sortString := r.urlQueryParamsValue("sort"); sortString != "" {
		if err := sort.UnmarshalText([]byte(sortString)); err != nil {
			return err
		}
	}

	resultSet, err := core.GetListItems(r.ctx, s.db, list.ID, limit, sort, next, r.viewer)
	if err != nil {
		return err
	}
	return w.writeJSON(resultSet)
}

// @Summary		Delete a list item.
// @Description	Delete a list item.
// @Router			/api/lists/{listId}/items/{itemId} [DELETE]
// @Success		200
// @Tags			Lists
// @Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
// @Param			listId			path	string	true	"List ID"
// @Param			itemId			path	string	true	"Item ID"
func (s *Server) deleteListItem(w *responseWriter, r *request, list *core.List) error {
	if !r.loggedIn {
		return errNotLoggedIn
	}

	itemID, err := strconv.Atoi(r.muxVar("itemId"))
	if err != nil {
		return httperr.NewBadRequest("invalid-item-id", "Invalid item id.")
	}

	if list.UserID != *r.viewer { // Check permissions.
		return httperr.NewForbidden("not-your-list", "Not your list.")
	}

	item, err := core.GetListItem(r.ctx, s.db, list.ID, itemID)
	if err != nil {
		return err
	}

	if err := item.Delete(r.ctx, s.db); err != nil {
		return err
	}

	return w.writeJSON(item)
}

// @Summary		Get items saved to lists.
// @Description	Get items saved to lists.
// @Router			/api/lists/_saved_to [GET]
// @Success		200
// @Tags			Lists
// @Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
// @Param			id				query	string	true	"Target ID"
func (s *Server) getSaveToLists(w *responseWriter, r *request) error {
	if !r.loggedIn {
		return errNotLoggedIn
	}

	targetID, err := uid.FromString(r.urlQueryParamsValue("id"))
	if err != nil {
		return httperr.NewBadRequest("invalid-target-id", "Invalid target id.")
	}

	var targetType core.ContentType
	if err := targetType.UnmarshalText([]byte(r.urlQueryParamsValue("type"))); err != nil {
		return httperr.NewBadRequest("invalid-content-type", "Invalid content type.")
	}

	ids, err := core.ListsItemIsSavedTo(r.ctx, s.db, *r.viewer, targetID, targetType)
	if err != nil {
		return err
	}

	return w.writeJSON(ids)
}
