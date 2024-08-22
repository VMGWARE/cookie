// biome-ignore lint: This is necessary for it to work
import React from "react";
import PropTypes from "prop-types";
import { useDispatch } from "react-redux";
import Dropdown from "../../components/Dropdown";
import { copyToClipboard, publicUrl } from "../../helper";
import { snackAlert } from "../../slices/mainSlice";

export const CommentShareDropdownItems = ({ url }) => {
  const dispatch = useDispatch();
  const handleCopyUrl = () => {
    let text = "Failed to copy link to clipboard.";
    if (copyToClipboard(publicUrl(url))) {
      text = "Link copied to clipboard.";
    }
    dispatch(snackAlert(text, "comment_link_copied"));
  };

  return (
    <>
      {/* <div className="dropdown-item">{to}Facebook</div>
      <div className="dropdown-item">{to}Twitter</div> */}
      <div className="dropdown-item" onClick={handleCopyUrl}>
        Copy URL
      </div>
    </>
  );
};

CommentShareDropdownItems.propTypes = {
  prefix: PropTypes.string,
  url: PropTypes.string.isRequired,
};

const CommentShareButton = ({ url }) => {
  return (
    <Dropdown
      target={
        <button type="button" className="button-text post-comment-button">
          Share
        </button>
      }
    >
      <div className="dropdown-list">
        <CommentShareDropdownItems url={url} />
      </div>
    </Dropdown>
  );
};

CommentShareButton.propTypes = {
  url: PropTypes.string.isRequired,
};

export default CommentShareButton;
