package server

import (
	"database/sql"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/discuitnet/discuit/core"
	"github.com/discuitnet/discuit/internal/hcaptcha"
	"github.com/discuitnet/discuit/internal/httperr"
	"github.com/discuitnet/discuit/internal/httputil"
	"github.com/discuitnet/discuit/internal/meilisearch"
	"github.com/discuitnet/discuit/internal/uid"
	"github.com/gorilla/mux"
)

// TODO: user.go, add swagger types for all functions and responses

// @Summary		Get a user by username.
// @Description	Get a user by username.
// @Router			/api/users/{username} [GET]
// @Success		200
// @Tags			Users
// @Param			username	path	string	true	"Username"
func (s *Server) getUser(w *responseWriter, r *request) error {
	username := r.muxVar("username")
	user, err := core.GetUserByUsername(r.ctx, s.db, username, r.viewer)
	if err != nil {
		return err
	}

	if user.IsGhost() {
		// For deleted accounts, expose the username for this API endpoint only.
		user.UnsetToGhost()
		username := user.Username
		user.SetToGhost()
		user.Username = username
	}

	if err := user.LoadModdingList(r.ctx); err != nil {
		return err
	}

	return w.writeJSON(user)
}

// @Summary		Delete a user.
// @Description	Delete a user.
// @Router			/api/users/{username} [DELETE]
// @Success		200
// @Tags			Users
// @Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
// @Param			username		path	string	true	"Username"
func (s *Server) deleteUser(w *responseWriter, r *request) error {
	if !r.loggedIn {
		return errNotLoggedIn
	}

	reqBody := struct {
		// Password is the password of the logged in user.
		Password string `json:"password"`
	}{}
	if err := r.unmarshalJSONBody(&reqBody); err != nil {
		return err
	}

	// Username might be the username of the logged in user or the username of
	// some other user. If it's the logged in user, then it's them deleting
	// their account. If it's some other user, then it's an admin deleting a
	// user account.
	username := r.muxVar("username")

	if err := s.rateLimit(r, "del_account_1_"+r.viewer.String(), time.Second*5, 1); err != nil {
		return err
	}

	doer, err := core.GetUser(r.ctx, s.db, *r.viewer, nil)
	if err != nil {
		return err
	}

	if _, err := core.MatchLoginCredentials(r.ctx, s.db, doer.Username, reqBody.Password); err != nil {
		if err == core.ErrWrongPassword {
			return httperr.NewForbidden("wrong_password", "Wrong password.")
		}
		return err
	}

	var toDelete *core.User
	if strings.ToLower(username) == doer.UsernameLowerCase {
		toDelete = doer
	} else {
		if !doer.Admin {
			// Doer is not an admin but trying to delete an account that isn't
			// theirs.
			return httperr.NewForbidden("not_admin", "You are not an admin.")
		}
		toDelete, err = core.GetUserByUsername(r.ctx, s.db, username, nil)
		if err != nil {
			return err
		}
	}

	// The user *must* be logged out of all active sessions before the account
	// is deleted.
	if err := s.LogoutAllSessionsOfUser(toDelete); err != nil {
		return err
	}

	// Finally, delete the user.
	if err := toDelete.Delete(r.ctx); err != nil {
		return err
	}

	meilisearch.UserDeleteDocumentIfEnabled(r.ctx, s.config, toDelete.ID.String())

	w.writeString(`{"success": true}`)
	return nil
}

// @Summary		Get initial data.
// @Description	Get initial data.
// @Router			/api/_initial [GET]
// @Success		200
// @Tags			General
func (s *Server) initial(w *responseWriter, r *request) error {
	var err error
	response := struct {
		ReportReasons  []core.ReportReason `json:"reportReasons"`
		User           *core.User          `json:"user"`
		Lists          []*core.List        `json:"lists"`
		Communities    []*core.Community   `json:"communities"`
		NoUsers        int                 `json:"noUsers"`
		BannedFrom     []uid.ID            `json:"bannedFrom"`
		VAPIDPublicKey string              `json:"vapidPublicKey"`
		Mutes          struct {
			CommunityMutes []*core.Mute `json:"communityMutes"`
			UserMutes      []*core.Mute `json:"userMutes"`
		} `json:"mutes"`
	}{
		Lists:          []*core.List{},
		VAPIDPublicKey: s.webPushVAPIDKeys.Public,
	}

	response.Mutes.CommunityMutes = []*core.Mute{}
	response.Mutes.UserMutes = []*core.Mute{}

	if r.loggedIn {
		if response.User, err = core.GetUser(r.ctx, s.db, *r.viewer, r.viewer); err != nil {
			if httperr.IsNotFound(err) {
				// Possible deleted user.
				// Reset session.
				// s.logoutUser(response.User, ses, w, r)
				// TODO: Things are weird here.
			}
			return err
		}
		if response.BannedFrom, err = response.User.GetBannedFromCommunities(r.ctx); err != nil {
			return err
		}
		if communityMutes, err := core.GetMutedCommunities(r.ctx, s.db, *r.viewer, true); err != nil {
			return err
		} else if communityMutes != nil {
			response.Mutes.CommunityMutes = communityMutes
		}
		if userMutes, err := core.GetMutedUsers(r.ctx, s.db, *r.viewer, true); err != nil {
			return err
		} else if userMutes != nil {
			response.Mutes.UserMutes = userMutes
		}
		if lists, err := core.GetUsersLists(r.ctx, s.db, *r.viewer, "", ""); err != nil {
			return err
		} else if lists != nil {
			response.Lists = lists
		}
	}

	if response.ReportReasons, err = core.GetReportReasons(r.ctx, s.db); err != nil && err != sql.ErrNoRows {
		return err
	}

	commsSet := core.CommunitiesSetDefault
	if r.loggedIn {
		commsSet = core.CommunitiesSetSubscribed
	}

	if response.Communities, err = core.GetCommunities(r.ctx, s.db, core.CommunitiesSortDefault, commsSet, -1, r.viewer); err != nil && err != sql.ErrNoRows {
		return err
	}
	if response.NoUsers, err = core.CountAllUsers(r.ctx, s.db); err != nil {
		return err
	}

	return w.writeJSON(response)
}

// @Summary		Login a user.
// @Description	Login a user.
// @Router			/api/_login [POST]
// @Success		200
// @Tags			Users
func (s *Server) login(w *responseWriter, r *request) error {
	if r.loggedIn {
		user, err := core.GetUser(r.ctx, s.db, *r.viewer, r.viewer)
		if err != nil {
			return err
		}

		action := r.urlQueryParamsValue("action")
		if action != "" {
			switch action {
			case "logout":
				if err = s.logoutUser(user, r.ses, w, r.req); err != nil {
					return err
				}
				w.WriteHeader(http.StatusOK)
				return nil
			default:
				return httperr.NewBadRequest("invalid_action", "Unsupported action.")
			}
		}
		return w.writeJSON(user)
	}

	values, err := r.unmarshalJSONBodyToStringsMap(true)
	if err != nil {
		return err
	}
	username := values["username"]
	// Important: Passwords values have always been space trimmed (using strings.TrimSpace).
	password := values["password"]

	// TODO: Require a captcha if user is suspicious looking.

	ip := httputil.GetIP(r.req)
	if err := s.rateLimit(r, "login_1_"+ip, time.Second, 10); err != nil {
		return err
	}
	if err := s.rateLimit(r, "login_2_"+ip+username, time.Hour, 20); err != nil {
		return err
	}

	user, err := core.MatchLoginCredentials(r.ctx, s.db, username, password)
	if err != nil {
		return err
	}

	if err = s.loginUser(user, r.ses, w, r.req); err != nil {
		return err
	}

	return w.writeJSON(user)
}

// @Summary		Signup a user.
// @Description	Signup a user.
// @Router			/api/_signup [POST]
// @Success		201
// @Tags			Users
func (s *Server) signup(w *responseWriter, r *request) error {
	if r.loggedIn {
		return httperr.NewBadRequest("already_logged_in", "You are already logged in")
	}

	values, err := r.unmarshalJSONBodyToStringsMap(true)
	if err != nil {
		return err
	}

	username := values["username"]
	email := values["email"]
	password := values["password"]
	captchaToken := values["captchaToken"]

	// Verify captcha.
	if s.config.CaptchaSecret != "" {
		if ok, err := hcaptcha.VerifyReCaptcha(s.config.CaptchaSecret, captchaToken); err != nil {
			return httperr.NewForbidden("captcha_verify_fail_1", "Captha verification failed.")
		} else if !ok {
			return httperr.NewForbidden("captcha_verify_fail_2", "Captha verification failed.")
		}
	}

	ip := httputil.GetIP(r.req)
	if err := s.rateLimit(r, "signup_1_"+ip, time.Minute, 2); err != nil {
		return err
	}
	if err := s.rateLimit(r, "signup_2_"+ip, time.Hour*6, 10); err != nil {
		return err
	}

	user, err := core.RegisterUser(r.ctx, s.db, username, email, password)
	if err != nil {
		return err
	}

	// Try logging in user.
	s.loginUser(user, r.ses, w, r.req)

	meilisearch.UserUpdateOrCreateDocumentIfEnabled(r.ctx, s.config, user)

	w.WriteHeader(http.StatusCreated)
	return w.writeJSON(user)
}

// @Summary		Get the logged in user.
// @Description	Get the logged in user.
// @Router			/api/_user [GET]
// @Success		200
// @Tags			Users
// @Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
func (s *Server) getLoggedInUser(w *responseWriter, r *request) error {
	if !r.loggedIn {
		return errNotLoggedIn
	}

	user, err := core.GetUser(r.ctx, s.db, *r.viewer, r.viewer)
	if err != nil {
		return err
	}

	if err := user.LoadModdingList(r.ctx); err != nil {
		return err
	}

	return w.writeJSON(user)
}

// @Summary		Update notifications.
// @Description	Update notifications.
// @Router			/api/notifications [POST]
// @Success		200
// @Tags			Notifications
// @Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
// @Param			action			query	string	true	"Action"
func (s *Server) updateNotifications(w *responseWriter, r *request) error {
	if !r.loggedIn {
		return errNotLoggedIn
	}

	if err := s.rateLimit(r, "update_notifs_1_"+r.viewer.String(), time.Second*1, 5); err != nil {
		return err
	}

	user, err := core.GetUser(r.ctx, s.db, *r.viewer, nil)
	if err != nil {
		return err
	}

	query := r.urlQueryParams()
	switch query.Get("action") {
	case "resetNewCount":
		if err = user.ResetNewNotificationsCount(r.ctx); err != nil {
			return err
		}
	case "markAllAsSeen":
		if err = user.MarkAllNotificationsAsSeen(r.ctx, core.NotificationType(query.Get("type"))); err != nil {
			return err
		}
	case "deleteAll":
		if err = user.DeleteAllNotifications(r.ctx); err != nil {
			return err
		}
	default:
		return httperr.NewBadRequest("invalid_action", "Unsupported action.")
	}

	return w.writeString(`{"success":true}`)
}

// @Summary		Get notifications.
// @Description	Get notifications.
// @Router			/api/notifications [GET]
// @Success		200
// @Tags			Notifications
// @Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
func (s *Server) getNotifications(w *responseWriter, r *request) error {
	if !r.loggedIn {
		return errNotLoggedIn
	}

	user, err := core.GetUser(r.ctx, s.db, *r.viewer, nil)
	if err != nil {
		return err
	}

	res := struct {
		Count    int                  `json:"count"`
		NewCount int                  `json:"newCount"`
		Items    []*core.Notification `json:"items"`
		Next     string               `json:"next"`
	}{}
	if res.Count, err = core.NotificationsCount(r.ctx, s.db, user.ID); err != nil {
		return err
	}
	res.NewCount = user.NumNewNotifications

	query := r.urlQueryParams()
	if res.Items, res.Next, err = core.GetNotifications(r.ctx, s.db, user.ID, 10, query.Get("next")); err != nil {
		return err
	}

	return w.writeJSON(res)
}

// @Summary		Get a notification.
// @Description	Get a notification.
// @Router			/api/notifications/{notificationID} [GET]
// @Success		200
// @Tags			Notifications
// @Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
// @Param			notificationID	path	string	true	"Notification ID"
func (s *Server) getNotification(w *responseWriter, r *request) error {
	if !r.loggedIn {
		return errNotLoggedIn
	}

	notifID := r.muxVar("notificationID")
	notif, err := core.GetNotification(r.ctx, s.db, notifID)
	if err != nil {
		if err == sql.ErrNoRows {
			return httperr.NewNotFound("notif_not_found", "Notification not found.")
		}
		return err
	}

	if !notif.UserID.EqualsTo(*r.viewer) {
		return httperr.NewForbidden("not_owner", "")
	}

	return w.writeJSON(notif)
}

// @Summary		Mark a notification as seen.
// @Description	Mark a notification as seen.
// @Router			/api/notifications/{notificationID} [PUT]
// @Success		200
// @Tags			Notifications
// @Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
// @Param			notificationID	path	string	true	"Notification ID"
// @Param			action			query	string	true	"Action"	Enums(markAsSeen)
func (s *Server) markAllNotificationAsSeen(w *responseWriter, r *request) error {
	if !r.loggedIn {
		return errNotLoggedIn
	}

	notifID := r.muxVar("notificationID")
	notif, err := core.GetNotification(r.ctx, s.db, notifID)
	if err != nil {
		if err == sql.ErrNoRows {
			return httperr.NewNotFound("notif_not_found", "Notification not found.")
		}
		return err
	}

	if !notif.UserID.EqualsTo(*r.viewer) {
		return httperr.NewForbidden("not_owner", "")
	}

	query := r.urlQueryParams()
	action := query.Get("action")
	switch action {
	case "markAsSeen":
		if err = notif.Saw(r.ctx, query.Get("seen") != "false"); err != nil {
			return err
		}
		if query.Get("seenFrom") == "webpush" {
			notif.ResetUserNewNotificationsCount(r.ctx) // attempt
		}
	default:
		return httperr.NewBadRequest("invalid_action", "Unsupported action.")
	}

	return w.writeJSON(notif)
}

// @Summary		Delete a notification.
// @Description	Delete a notification.
// @Router			/api/notifications/{notificationID} [DELETE]
// @Success		200
// @Tags			Notifications
// @Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
// @Param			notificationID	path	string	true	"Notification ID"
func (s *Server) deleteNotification(w *responseWriter, r *request) error {
	if !r.loggedIn {
		return errNotLoggedIn
	}

	notifID := r.muxVar("notificationID")
	notif, err := core.GetNotification(r.ctx, s.db, notifID)
	if err != nil {
		if err == sql.ErrNoRows {
			return httperr.NewNotFound("notif_not_found", "Notification not found.")
		}
		return err
	}

	if !notif.UserID.EqualsTo(*r.viewer) {
		return httperr.NewForbidden("not_owner", "")
	}

	if err = notif.Delete(r.ctx); err != nil {
		return err
	}

	return w.writeJSON(notif)
}

// @Summary		Push subscriptions.
// @Description	Push subscriptions.
// @Router			/api/push_subscriptions [POST]
// @Success		200	{string}	string	"{"success":true}"
// @Tags			Users
// @Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
func (s *Server) pushSubscriptions(w *responseWriter, r *request) error {
	if !r.loggedIn {
		return errNotLoggedIn
	}

	var sub webpush.Subscription
	if err := r.unmarshalJSONBody(&sub); err != nil {
		return err
	}

	if err := core.SaveWebPushSubscription(r.ctx, s.db, r.ses.ID, *r.viewer, sub); err != nil {
		return err
	}

	return w.writeString(`{"success":true}`)
}

// @Summary		Update user settings.
// @Description	Update user settings.
// @Router			/api/_settings [POST]
// @Success		200
// @Tags			Users
// @Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
func (s *Server) updateUserSettings(w *responseWriter, r *request) error {
	if !r.loggedIn {
		return errNotLoggedIn
	}

	if err := s.rateLimit(r, "update_settings_1_"+r.viewer.String(), time.Second*1, 5); err != nil {
		return err
	}
	if err := s.rateLimit(r, "update_settings_2_"+r.viewer.String(), time.Hour, 100); err != nil {
		return err
	}

	user, err := core.GetUser(r.ctx, s.db, *r.viewer, r.viewer)
	if err != nil {
		return err
	}

	query := r.urlQueryParams()
	switch query.Get("action") {
	case "updateProfile":
		if err = r.unmarshalJSONBody(&user); err != nil {
			return err
		}

		if err = user.Update(r.ctx); err != nil {
			return err
		}
	case "changePassword":
		values, err := r.unmarshalJSONBodyToStringsMap(true)
		if err != nil {
			return err
		}
		password := values["password"]
		newPassword := values["newPassword"]
		repeatPassword := values["repeatPassword"]
		if newPassword != repeatPassword {
			return httperr.NewBadRequest("password_not_match", "Passwords do not match.")
		}
		if err = user.ChangePassword(r.ctx, password, newPassword); err != nil {
			return err
		}
	default:
		return httperr.NewBadRequest("invalid_action", "Unsupported action.")
	}

	meilisearch.UserUpdateOrCreateDocumentIfEnabled(r.ctx, s.config, user)

	return w.writeJSON(user)
}

// @Summary		Upload a user profile picture.
// @Description	Upload a user profile picture.
// @Router			/api/users/{username}/pro_pic [POST]
// @Success		200
// @Tags			Users
// @Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
// @Param			username		path	string	true	"Username"
func (s *Server) UploadUserProPic(w *responseWriter, r *request) error {
	if !r.loggedIn {
		return errNotLoggedIn
	}

	user, err := core.GetUserByUsername(r.ctx, s.db, r.muxVar("username"), r.viewer)
	if err != nil {
		return err
	}

	// Only the owner of the account and admins can proceed.
	if !(user.ID == *r.viewer || user.Admin) {
		return httperr.NewForbidden("not_owner", "")
	}

	r.req.Body = http.MaxBytesReader(w, r.req.Body, int64(s.config.MaxImageSize)) // limit max upload size
	if err := r.req.ParseMultipartForm(int64(s.config.MaxImageSize)); err != nil {
		return httperr.NewBadRequest("file_size_exceeded", "Max file size exceeded.")
	}

	file, _, err := r.req.FormFile("image")
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return err
	}
	if err := user.UpdateProPic(r.ctx, data); err != nil {
		return err
	}

	return w.writeJSON(user)
}

// @Summary		Delete a user profile picture.
// @Description	Delete a user profile picture.
// @Router			/api/users/{username}/pro_pic [DELETE]
// @Success		200
// @Tags			Users
// @Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
// @Param			username		path	string	true	"Username"
func (s *Server) deleteUserProPic(w *responseWriter, r *request) error {
	if !r.loggedIn {
		return errNotLoggedIn
	}

	user, err := core.GetUserByUsername(r.ctx, s.db, r.muxVar("username"), r.viewer)
	if err != nil {
		return err
	}

	// Only the owner of the account and admins can proceed.
	if !(user.ID == *r.viewer || user.Admin) {
		return httperr.NewForbidden("not_owner", "")
	}

	if err := user.DeleteProPic(r.ctx); err != nil {
		return err
	}

	return w.writeJSON(user)
}

// @Summary		Delete a badge from a user.
// @Description	Delete a badge from a user.
// @Router			/api/users/{username}/badges/{badgeId} [DELETE]
// @Success		200
// @Tags			Users
// @Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
// @Param			username		path	string	true	"Username"
// @Param			badgeId			path	string	true	"Badge ID"
// @Param			byType			query	string	false	"By type"	Enums(true,false)
func (s *Server) deleteBadge(w *responseWriter, r *request) error {
	if !r.loggedIn {
		return errNotLoggedIn
	}

	admin, err := core.GetUser(r.ctx, s.db, *r.viewer, nil)
	if err != nil {
		return err
	}

	if !admin.Admin {
		return httperr.NewForbidden("not_admin", "Not admin.")
	}

	muxVars := mux.Vars(r.req)
	badgeID, username := muxVars["badgeId"], muxVars["username"]
	user, err := core.GetUserByUsername(r.ctx, s.db, username, nil)
	if err != nil {
		return err
	}

	byType := strings.ToLower(r.urlQueryParamsValue("byType")) == "true"
	if byType {
		if err = user.RemoveBadgesByType(badgeID); err != nil {
			return err
		}
	} else {
		intID, err := strconv.Atoi(badgeID)
		if err != nil {
			return httperr.NewBadRequest("bad_badge_id", "Bad badge id.")
		}
		if err := user.RemoveBadge(intID); err != nil {
			return err
		}
	}

	return w.writeString(`{"success":true}`)
}

// @Summary		Add a badge to a user.
// @Description	Add a badge to a user.
// @Router			/api/users/{username}/badges [POST]
// @Success		200
// @Tags			Users
// @Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
// @Param			username		path	string	true	"Username"
func (s *Server) addBadge(w *responseWriter, r *request) error {
	if !r.loggedIn {
		return errNotLoggedIn
	}

	admin, err := core.GetUser(r.ctx, s.db, *r.viewer, nil)
	if err != nil {
		return err
	}

	if !admin.Admin {
		return httperr.NewForbidden("not_admin", "User not admin.")
	}

	username := r.muxVar("username")
	user, err := core.GetUserByUsername(r.ctx, s.db, username, nil)
	if err != nil {
		return err
	}

	reqBody := struct {
		BadgeType string `json:"type"`
	}{}

	if err = r.unmarshalJSONBody(&reqBody); err != nil {
		return err
	}

	if err := user.AddBadge(r.ctx, reqBody.BadgeType); err != nil {
		return err
	}

	return w.writeJSON(user.Badges)
}
