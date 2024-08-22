// biome-ignore lint: This is necessary for it to work
import React from "react";
import PropTypes from "prop-types";
import BannerImg from "../../assets/imgs/community-banner-2.jpg";
import Image from "../../components/Image";
import { selectImageCopyUrl } from "../../helper";

const Banner = ({ community, ...rest }) => {
  let src = BannerImg;
  if (community.bannerImage) {
    src = selectImageCopyUrl("small", community.bannerImage);
  }

  return (
    <Image
      src={src}
      alt={`${community.name}'s banner`}
      backgroundColor={
        community.bannerImage ? community.bannerImage.averageColor : "#fff"
      }
      {...rest}
      isFullSize
    />
  );
};

Banner.propTypes = {
  community: PropTypes.object.isRequired,
};

export default Banner;
