// biome-ignore lint: This is necessary for it to work
import React from "react";
import Spinner from "../components/Spinner";

const AppLoading = () => {
  return (
    <div className="app-loading">
      <Spinner />
    </div>
  );
};

export default AppLoading;
