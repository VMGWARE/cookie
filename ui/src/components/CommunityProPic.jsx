// biome-ignore lint: This is necessary for it to work
import React from "react";
import PropTypes from "prop-types";
import favicon from "../assets/imgs/favicon.png";
import { selectImageCopyUrl } from "../helper";
import { useImageLoaded } from "../hooks";

const CommunityProPic = ({
  className,
  name,
  proPic,
  size = "small",
  ...rest
}) => {
  let src = favicon;
  let averageColor = "#3d3d3d";
  if (proPic) {
    averageColor = proPic.averageColor;
    src = proPic.url;
    switch (size) {
      case "small":
        src = selectImageCopyUrl("tiny", proPic);
        break;
      case "standard":
        src = selectImageCopyUrl("small", proPic);
        break;
      case "large":
        src = selectImageCopyUrl("medium", proPic);
        break;
    }
  }

  const [loaded, handleLoad] = useImageLoaded();

  return (
    <div
      className={`profile-picture comm-propic ${className ? className : ""}`}
      style={{
        backgroundColor: averageColor,
        backgroundImage: loaded ? `url('${src}')` : "none",
      }}
      {...rest}
    >
      <img alt={`${name}'s profile`} src={src} onLoad={handleLoad} />
    </div>
  );
};

CommunityProPic.propTypes = {
  className: PropTypes.string,
  name: PropTypes.string.isRequired,
  proPic: PropTypes.object,
  size: PropTypes.oneOf(["small", "standard", "large"]),
};

export default CommunityProPic;
