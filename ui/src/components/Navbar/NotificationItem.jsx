// biome-ignore lint: This is necessary for it to work
import React from "react";
import PropTypes from "prop-types";
import { useRef, useState } from "react";
import { useDispatch, useSelector } from "react-redux";
import { useHistory } from "react-router";
import Favicon from "../../assets/imgs/favicon.png";
import { mfetchjson, selectImageCopyUrl, stringCount } from "../../helper";
import { badgeImage } from "../../pages/User/Badge";
import {
  markNotificationAsSeen,
  notificationsDeleted,
  snackAlertError,
} from "../../slices/mainSlice";
import { ButtonMore } from "../Button";
import Dropdown from "../Dropdown";
import Image from "../Image";
import TimeAgo from "../TimeAgo";

const NotificationItem = ({ notification, ...rest }) => {
  const { type, seen, createdAt, notif } = notification;

  const viewer = useSelector((state) => state.main.user);

  const [actionBtnHovering, setActionBtnHovering] = useState(false);
  const [dropdownActive, setDropdownActive] = useState(false);

  const dispatch = useDispatch();
  const handleMarkAsSeen = () =>
    dispatch(markNotificationAsSeen(notification, !seen));
  const handleDelete = async () => {
    try {
      await mfetchjson(`/api/notifications/${notification.id}`, {
        method: "DELETE",
      });
      dispatch(notificationsDeleted(notification));
    } catch (error) {
      dispatch(snackAlertError(error));
    }
  };

  const renderText = () => {
    switch (type) {
      case "new_comment": {
        if (notif.noComments === 1) {
          return (
            <>
              <b>@{notif.commentAuthor}</b> commented on your post{" "}
              <b>{notif.post.title}</b>.
            </>
          );
        }
        return (
          <>
            {notif.noComments} new comments on your post{" "}
            <b>{notif.post.title}</b>.
          </>
        );
      }
      case "comment_reply": {
        if (notif.noComments === 1) {
          return (
            <>
              <b>@{notif.commentAuthor}</b> replied to your comment on post{" "}
              <b>{notif.post.title}</b>.
            </>
          );
        }
        return (
          <>
            {notif.noComments} new replies to your comment on post{" "}
            <b>{notif.post.title}</b>.
          </>
        );
      }
      case "new_votes": {
        if (notif.targetType === "post") {
          return (
            <>
              {stringCount(notif.noVotes, false, "new upvote")} on your post{" "}
              <b>{notif.post.title}</b>.
            </>
          );
        }
        return (
          <>
            {stringCount(notif.noVotes, false, "new vote")} on your comment in{" "}
            <b>{`~${notif.post.title}`}</b>.
          </>
        );
      }
      case "deleted_post": {
        return (
          <>
            Your post <b>{notif.post.title}</b> has been removed by{" "}
            {notif.deletedAs === "mods" ? (
              <>
                moderators of <b>{notif.post.communityName}</b>
              </>
            ) : (
              "the admins"
            )}
            .
          </>
        );
      }
      case "mod_add": {
        return (
          <>
            You are added as a moderator of <b>{notif.communityName}</b> by{" "}
            <b>@{notif.addedBy}.</b>
          </>
        );
      }
      case "new_badge": {
        return (
          <>
            You are awarded the <b>supporter</b> badge for your contribution to
            Discuit and for sheer awesomeness!
          </>
        );
      }
      default: {
        return null; // unknown notification type
      }
    }
  };

  const defaultImage = { url: Favicon, backgroundColor: "#3d3d3d" };
  const getNotifImage = (notif) => {
    let image = Favicon;
    let background = "#3d3d3d";
    if (notif.post) {
      switch (notif.post.type) {
        case "image":
          if (notif.post.image) {
            image = selectImageCopyUrl("tiny", notif.post.image);
            background = notif.post.image.averageColor;
          }
          break;
        case "link":
          if (notif.post.link?.image) {
            image = selectImageCopyUrl("tiny", notif.post.link.image);
            background = notif.post.link.image.averageColor;
          }
          break;
      }
    } else if (
      typeof notif.community === "object" &&
      notif.community !== null
    ) {
      if (notif.community.proPic) {
        image = selectImageCopyUrl("small", notif.community.proPic);
        background = notif.community.proPic.averageColor;
      }
    }
    return { url: image, backgroundColor: background };
  };

  let image = defaultImage;
  let to = "";
  switch (type) {
    case "new_comment": {
      to = `/${CONFIG.communityPrefix}${notif.post.communityName}/post/${notif.post.publicId}`;
      if (notif.noComments === 1) {
        to += `/${notif.commentId}`;
      } else {
        to += `?notifId=${notification.id}`;
      }
      image = getNotifImage(notif);
      break;
    }
    case "comment_reply": {
      to = `/${CONFIG.communityPrefix}${notif.post.communityName}/post/${notif.post.publicId}`;
      if (notif.noComments === 1) {
        to += `/${notif.commentId}`;
      } else {
        to += `?notifId=${notification.id}`;
      }
      image = getNotifImage(notif);
      break;
    }
    case "new_votes": {
      if (notif.targetType === "post") {
        to = `/${CONFIG.communityPrefix}${notif.post.communityName}/post/${notif.post.publicId}`;
      } else {
        to = `/${CONFIG.communityPrefix}${notif.comment.communityName}/post/${notif.comment.postPublicId}/${notif.comment.id}`;
      }
      image = getNotifImage(notif);
      break;
    }
    case "deleted_post": {
      // Currently only deleted post notifications get here.
      to = `/${CONFIG.communityPrefix}${notif.post.communityName}/post/${notif.post.publicId}`;
      image = getNotifImage(notif);
      break;
    }
    case "mod_add": {
      to = `/${CONFIG.communityPrefix}${notif.communityName}`;
      image = getNotifImage(notif);
      break;
    }
    case "new_badge": {
      to = `/@${viewer.username}`;
      const { src } = badgeImage(notif.badgeType);
      image = {
        url: src,
        backgroundColor: "transparent",
      };
      break;
    }
  }

  const actionsRef = useRef();
  const history = useHistory();
  const handleClick = (e) => {
    e.preventDefault();
    if (
      !(
        actionsRef.current.contains(e.target) ||
        document.querySelector("#modal-root").contains(e.target)
      )
    ) {
      if (!seen) {
        handleMarkAsSeen();
      }
      history.push(to, {
        fromNotifications: true,
      });
    }
  };

  const notifText = renderText();
  if (notifText === null) {
    return null; // notification type is unknown
  }

  return (
    <a
      href={to}
      className={`link-reset notif ${seen ? "is-seen" : ""} ${actionBtnHovering ? "is-btn-hovering" : ""}`}
      onClick={handleClick}
      {...rest}
    >
      <div className="notif-icon">
        <Image src={image.url} backgroundColor={image.backgroundColor} alt="" />
      </div>
      <div className="notif-body">
        <div className="notif-text">{notifText}</div>
        <div className="notif-time">
          <TimeAgo time={createdAt} inline={false} />
        </div>
      </div>
      <div
        ref={actionsRef}
        className={`notif-action-btn ${dropdownActive ? "is-active" : ""}`}
        onMouseEnter={() => setActionBtnHovering(true)}
        onMouseLeave={() => setActionBtnHovering(false)}
      >
        <Dropdown
          target={<ButtonMore />}
          aligned="right"
          onActiveChange={(active) => setDropdownActive(active)}
        >
          <div className="dropdown-list">
            <button
              type="button"
              className="button-clear dropdown-item"
              onClick={handleMarkAsSeen}
            >
              {`Mark as ${seen ? "un" : ""}seen`}
            </button>
            <button
              type="button"
              className="button-clear dropdown-item"
              onClick={handleDelete}
            >
              Delete
            </button>
          </div>
        </Dropdown>
      </div>
    </a>
  );
};

NotificationItem.propTypes = {
  notification: PropTypes.object.isRequired,
};

export default NotificationItem;
