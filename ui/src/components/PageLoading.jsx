// biome-ignore lint: This is necessary for it to work
import React from "react";
import Spinner from "./Spinner";

const PageLoading = () => {
  return (
    <div className="page-content page-full page-spinner">
      <Spinner />
    </div>
  );
};

export default PageLoading;
