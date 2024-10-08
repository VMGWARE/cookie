// biome-ignore lint: This is necessary for it to work
import React from "react";
import { useEffect, useState } from "react";
import { Helmet } from "react-helmet-async";
import { useDispatch, useSelector } from "react-redux";
import { Link } from "react-router-dom";
import {
  getNotificationsPermissions,
  shouldAskForNotificationsPermissions,
} from "../../PushNotifications";
import { ButtonUpload } from "../../components/Button";
import CommunityProPic from "../../components/CommunityProPic";
import Dropdown from "../../components/Dropdown";
import Input from "../../components/Input";
import CommunityLink from "../../components/PostCard/CommunityLink";
import { mfetch, mfetchjson, validEmail } from "../../helper";
import { useIsChanged } from "../../hooks";
import {
  mutesAdded,
  settingsChanged,
  snackAlert,
  snackAlertError,
  unmuteCommunity,
  unmuteUser,
  userLoggedIn,
} from "../../slices/mainSlice";
import ChangePassword from "./ChangePassword";
import DeleteAccount from "./DeleteAccount";
import { getDevicePreference, setDevicePreference } from "./devicePrefs";

const Settings = () => {
  const dispatch = useDispatch();
  const user = useSelector((state) => state.main.user);
  const loggedIn = user !== null;

  const mutes = useSelector((state) => state.main.mutes);
  const [aboutMe, setAboutMe] = useState(user.aboutMe || "");
  const [email, setEmail] = useState(user.email || "");

  const [notifsSettings, _setNotifsSettings] = useState({
    upvoteNotifs: !user.upvoteNotificationsOff,
    replyNotifs: !user.replyNotificationsOff,
  });
  const setNotifsSettings = (key, val) => {
    _setNotifsSettings((prev) => {
      return {
        ...prev,
        [key]: val,
      };
    });
  };

  const homeFeedOptions = {
    all: "All",
    subscriptions: "Subscriptions",
  };
  const [homeFeed, setHomeFeed] = useState(user.homeFeed);

  const [rememberFeedSort, setRememberFeedSort] = useState(
    user.rememberFeedSort,
  );
  const [enableEmbeds, setEnableEmbeds] = useState(!user.embedsOff);
  const [showUserProfilePictures, setShowUserProfilePictures] = useState(
    !user.hideUserProfilePictures,
  );

  // Per-device preferences:
  const [font, setFont] = useState(getDevicePreference("font") ?? "custom");
  const fontOptions = {
    custom: "Custom", // value -> display name
    system: "System",
  };

  const [changed, resetChanged] = useIsChanged([
    aboutMe /*, email*/,
    notifsSettings,
    homeFeed,
    rememberFeedSort,
    enableEmbeds,
    email,
    showUserProfilePictures,
    font,
  ]);

  const applicationServerKey = useSelector(
    (state) => state.main.vapidPublicKey,
  );
  const [notificationsPermissions, setNotificationsPermissions] = useState(
    window.Notification && Notification.permission,
  );
  useEffect(() => {
    let cleanupFunc;
    let cancelled = false;
    const f = async () => {
      if ("permissions" in navigator) {
        const status = await navigator.permissions.query({
          name: "notifications",
        });
        const listener = () => {
          if (!cancelled) {
            setNotificationsPermissions(status.state);
          }
        };
        status.addEventListener("change", listener);
        cleanupFunc = () => status.removeEventListener("change", listener);
      }
    };
    f();
    return () => {
      cancelled = true;
      if (cleanupFunc) {
        cleanupFunc();
      }
    };
  }, []);
  const [canEnableWebPushNotifications, setCanEnableWebPushNotifications] =
    useState(
      shouldAskForNotificationsPermissions(
        loggedIn,
        applicationServerKey,
        false,
      ),
    );
  useEffect(() => {
    setCanEnableWebPushNotifications(
      shouldAskForNotificationsPermissions(
        loggedIn,
        applicationServerKey,
        false,
      ),
    );
  }, [notificationsPermissions]);

  const handleEnablePushNotifications = async () => {
    await getNotificationsPermissions(loggedIn, applicationServerKey);
  };

  const handleSave = async () => {
    if (email !== "" && !validEmail(email)) {
      dispatch(snackAlert("Please enter a valid email"));
      return;
    }
    // Save device preferences first:
    setDevicePreference("font", font);
    try {
      const ruser = await mfetchjson("/api/_settings?action=updateProfile", {
        method: "POST",
        body: JSON.stringify({
          aboutMe,
          upvoteNotificationsOff: !notifsSettings.upvoteNotifs,
          replyNotificationsOff: !notifsSettings.replyNotifs,
          homeFeed,
          rememberFeedSort,
          embedsOff: !enableEmbeds,
          email,
          hideUserProfilePictures: !showUserProfilePictures,
        }),
      });
      dispatch(userLoggedIn(ruser));
      dispatch(snackAlert("Settings saved.", "settings_saved"));
      resetChanged();
      dispatch(settingsChanged());
    } catch (error) {
      dispatch(snackAlertError(error));
    }
  };

  const proPicApiEndpoint = `/api/users/${user.username}/pro_pic`;
  const [isProPicUploading, setIsProPicUploading] = useState(false);
  const handleProPicUpload = async (files) => {
    if (isProPicUploading) {
      return;
    }
    try {
      const formData = new FormData();
      formData.append("image", files[0]);
      setIsProPicUploading(true);
      const res = await mfetch(proPicApiEndpoint, {
        method: "POST",
        body: formData,
      });
      if (!res.ok) {
        if (res.status === 400) {
          const error = await res.json();
          if (error.code === "file_size_exceeded") {
            dispatch(snackAlert("Maximum file size exceeded."));
            return;
          }
          if (error.code === "unsupported_image") {
            dispatch(snackAlert("Unsupported image."));
            return;
          }
          throw new APIError(res.status, await res.json());
        }
      }
      const ruser = await res.json();
      dispatch(userLoggedIn(ruser));
    } catch (error) {
      dispatch(snackAlertError(error));
    } finally {
      setIsProPicUploading(false);
    }
  };
  const handleProPicDelete = async () => {
    if (isProPicUploading) {
      return;
    }
    try {
      const ruser = await mfetchjson(proPicApiEndpoint, { method: "DELETE" });
      dispatch(userLoggedIn(ruser));
    } catch (error) {
      dispatch(snackAlertError(error));
    } finally {
      setIsProPicUploading(false);
    }
  };

  const handleUnmute = (mute) => {
    // try {
    //   await mfetchjson(`/api/mutes/${mute.id}`, {
    //     method: 'DELETE',
    //   });
    //   setMutes((mutes) => {
    //     let array, fieldName;
    //     if (mute.type === 'community') {
    //       array = mutes.communityMutes;
    //       fieldName = 'communityMutes';
    //     } else {
    //       array = mutes.userMutes;
    //       fieldName = 'userMutes';
    //     }
    //     array = array.filter((m) => m.id !== mute.id);
    //     return {
    //       ...mutes,
    //       [fieldName]: array,
    //     };
    //   });
    // } catch (error) {
    //   dispatch(snackAlertError(error));
    // }
    if (mute.type === "community") {
      const community = mute.mutedCommunity;
      dispatch(unmuteCommunity(community.id, community.name));
    } else {
      const user = mute.mutedUser;
      dispatch(unmuteUser(user.id, user.username));
    }
  };

  const handleUnmuteAll = async (type = "") => {
    try {
      await mfetchjson(`/api/mutes?type=${type}`, {
        method: "DELETE",
      });
      // setMutes((mutes) => {
      //   const fieldName = type === 'community' ? 'communityMutes' : 'userMutes';
      //   return {
      //     ...mutes,
      //     [fieldName]: [],
      //   };
      // });
      const newMutes = {
        ...mutes,
      };
      newMutes[`${type}Mutes`] = [];
      dispatch(mutesAdded(newMutes));
    } catch (error) {
      dispatch(snackAlertError(error));
    }
  };

  const renderMute = (mute) => {
    if (mute.type === "community") {
      const community = mute.mutedCommunity;
      return (
        <div>
          <CommunityLink name={community.name} proPic={community.proPic} />
          <button type="button" onClick={() => handleUnmute(mute)}>
            Unmute
          </button>
        </div>
      );
    }
    if (mute.type === "user") {
      const user = mute.mutedUser;
      return (
        <div>
          <Link to={`/@${user.username}`}>@{user.username}</Link>
          <button type="button" onClick={() => handleUnmute(mute)}>
            Unmute
          </button>
        </div>
      );
    }
    return "Unkonwn muting type.";
  };

  return (
    <div className="page-content wrap page-settings">
      <Helmet>
        <title>Settings</title>
      </Helmet>
      <div className="account-settings card">
        <h1>Account settings</h1>
        <div className="settings-propic">
          <CommunityProPic
            name={user.username}
            proPic={user.proPic}
            size="standard"
          />
          <ButtonUpload
            onChange={handleProPicUpload}
            disabled={isProPicUploading}
          >
            Change
          </ButtonUpload>
          <button
            type="button"
            onClick={handleProPicDelete}
            disabled={isProPicUploading}
          >
            Delete
          </button>
        </div>
        <div className="flex-column">
          <Input label="Username" value={user.username || ""} disabled />
          <p className="input-desc">Username cannot be changed.</p>
        </div>
        <div className="flex-column">
          <Input
            label="Email"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
          />
        </div>
        <div className="input-with-label">
          <div className="input-label-box">
            <div className="label">About me</div>
          </div>
          <textarea
            rows="5"
            placeholder="Write something about yourself..."
            value={aboutMe}
            onChange={(e) => setAboutMe(e.target.value)}
          />
        </div>
        <ChangePassword />
        <DeleteAccount user={user} />
        <div className="input-with-label settings-prefs">
          <div className="input-label-box">
            <div className="label">Preferences</div>
          </div>
          <div className="settings-list">
            <div>
              <div>Home feed</div>
              <Dropdown
                aligned="right"
                target={
                  <button type="button" className="select-bar-dp-target">
                    {homeFeedOptions[homeFeed]}
                  </button>
                }
              >
                <div className="dropdown-list">
                  {Object.keys(homeFeedOptions)
                    .filter((key) => key !== homeFeed)
                    .map((key) => (
                      <div
                        key={key}
                        className="dropdown-item"
                        onClick={() => setHomeFeed(key)}
                      >
                        {homeFeedOptions[key]}
                      </div>
                    ))}
                </div>
              </Dropdown>
            </div>
            <div className="checkbox is-check-last">
              <label htmlFor="c3">Remember last feed sort</label>
              <input
                className="switch"
                id="c3"
                type="checkbox"
                checked={rememberFeedSort}
                onChange={(e) => setRememberFeedSort(e.target.checked)}
              />
            </div>
            <div className="checkbox is-check-last">
              <label htmlFor="c4">Enable embeds</label>
              <input
                className="switch"
                id="c4"
                type="checkbox"
                checked={enableEmbeds}
                onChange={(e) => setEnableEmbeds(e.target.checked)}
              />
            </div>
            <div className="checkbox is-check-last">
              <label htmlFor="c5">Show user profile pictures</label>
              <input
                className="switch"
                id="c5"
                type="checkbox"
                checked={showUserProfilePictures}
                onChange={(e) => setShowUserProfilePictures(e.target.checked)}
              />
            </div>
          </div>
        </div>
        <div className="input-with-label settings-device">
          <div className="input-label-box">
            <div className="label">Device preferences</div>
          </div>
          <div className="settings-list">
            <div>
              <div>Font</div>
              <Dropdown
                aligned="right"
                target={
                  <button type="button" className="select-bar-dp-target">
                    {fontOptions[font]}
                  </button>
                }
              >
                <div className="dropdown-list">
                  {Object.keys(fontOptions).map((key) => (
                    <div
                      key={key}
                      className="dropdown-item"
                      onClick={() => setFont(key)}
                    >
                      {fontOptions[key]}
                    </div>
                  ))}
                </div>
              </Dropdown>
            </div>
          </div>
        </div>
        <div className="input-with-label settings-notifs">
          <div className="input-label-box">
            <div className="label">Notifications</div>
          </div>
          <div className="settings-list">
            <div className="checkbox is-check-last">
              <label htmlFor="c1">Enable upvote notifications</label>
              <input
                className="switch"
                id="c1"
                type="checkbox"
                checked={notifsSettings.upvoteNotifs}
                onChange={(e) =>
                  setNotifsSettings("upvoteNotifs", e.target.checked)
                }
              />
            </div>
            <div className="checkbox is-check-last">
              <label htmlFor="c2">Enable reply notifications</label>
              <input
                className="switch"
                id="c2"
                type="checkbox"
                checked={notifsSettings.replyNotifs}
                onChange={(e) =>
                  setNotifsSettings("replyNotifs", e.target.checked)
                }
              />
            </div>
            {/*notificationsPermissions === 'granted' && (
              <button onClick={handleDisablePushNotifications} style={{ alignSelf: 'flex-start' }}>
                Disable push notifications
              </button>
            )*/}
            {canEnableWebPushNotifications && (
              <button
                type="button"
                onClick={handleEnablePushNotifications}
                style={{ alignSelf: "flex-start" }}
              >
                Enable push notifications
              </button>
            )}
          </div>
        </div>
        <div className="input-with-label settings-prefs">
          <div className="input-label-box">
            <div className="label">Muted communities</div>
          </div>
          <div className="settings-list">
            {mutes.communityMutes.length === 0 && <div>None</div>}
            {mutes.communityMutes.map((mute) => renderMute(mute))}
            {mutes.communityMutes.length > 0 && (
              <button
                type="button"
                style={{ alignSelf: "flex-end" }}
                onClick={() => handleUnmuteAll("community")}
              >
                Unmute all
              </button>
            )}
          </div>
        </div>
        <div className="input-with-label settings-prefs">
          <div className="input-label-box">
            <div className="label">Muted users</div>
          </div>
          <div className="settings-list">
            {mutes.userMutes.length === 0 && <div>None</div>}
            {mutes.userMutes.map((mute) => renderMute(mute))}
            {mutes.userMutes.length > 0 && (
              <button
                type="button"
                style={{ alignSelf: "flex-end" }}
                onClick={() => handleUnmuteAll("user")}
              >
                Unmute all
              </button>
            )}
          </div>
        </div>
        <button
          type="button"
          className="button-main"
          disabled={!changed}
          onClick={handleSave}
        >
          Save
        </button>
      </div>
    </div>
  );
};

export default Settings;
