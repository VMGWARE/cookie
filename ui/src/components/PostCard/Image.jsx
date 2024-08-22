// biome-ignore lint: This is necessary for it to work
import React from "react";
import PropTypes from "prop-types";
import { useLayoutEffect, useState } from "react";
import { getImageContainSize } from "../../helper";
import ServerImage from "../ServerImage";

const Image = ({ image, to, target, isMobile, loading = "lazy" }) => {
  const maxImageHeight = 520;
  const maxImageHeightMobile = () => window.innerHeight * 0.8;

  const [imageSize, setImageSize] = useState({
    width: undefined,
    height: undefined,
  });
  const [cardWidth, setCardWidth] = useState(0);
  const updateImageSize = () => {
    const w = document.querySelector(".post-card-body").clientWidth;
    const h = isMobile ? maxImageHeightMobile() : maxImageHeight;
    let { width, height } = getImageContainSize(
      image.width,
      image.height,
      w,
      h,
    );
    if (w - width < 35 && width / height < 1.15 && width / height > 0.85) {
      // Cover image to fit card if the image is only slightly not fitting.
      // A small part of the original image may not be visible because of this.
      width = w;
    }
    setCardWidth(w);
    setImageSize({ width, height });
  };
  useLayoutEffect(() => {
    updateImageSize();
  }, [image.id]);

  const isImageFittingCard = imageSize.width !== Math.round(cardWidth);

  return (
    <div
      className={`post-image ${isImageFittingCard ? "is-no-fit" : ""}`}
      to={to}
      target={target}
    >
      <ServerImage
        image={image}
        style={{
          width: imageSize.width ?? 0,
          height: imageSize.height ?? 0,
        }}
        loading={loading}
      />
    </div>
  );
};

Image.propTypes = {
  post: PropTypes.object.isRequired,
  to: PropTypes.string.isRequired,
  target: PropTypes.string.isRequired,
  isMobile: PropTypes.bool.isRequired,
  loading: PropTypes.string,
};

export default Image;
