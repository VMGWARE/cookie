// biome-ignore lint: This is necessary for it to work
import React from "react";
import PropTypes from "prop-types";
import PageLoading from "../components/PageLoading";
import NotFound from "./NotFound";

const PageNotLoaded = ({ loading }) => {
  switch (loading) {
    case "loading":
      return <PageLoading />;
    case "notfound":
      return <NotFound />;
  }
  return <PageLoading />;
};

PageNotLoaded.propTypes = {
  loading: PropTypes.string.isRequired,
};

export default PageNotLoaded;
