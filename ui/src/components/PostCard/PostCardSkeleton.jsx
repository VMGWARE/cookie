// biome-ignore lint: This is necessary for it to work
import React from "react";
const PostCardSkeleton = () => {
  return (
    <div className="post-skeleton">
      <div className="skeleton">
        <div className="skeleton-bar" style={{ width: "40%", height: 16 }} />
        <div className="skeleton-bar" style={{ width: "70%", height: 32 }} />
        <div className="skeleton-bar" style={{ width: "100%", height: 240 }} />
      </div>
    </div>
  );
};

export default PostCardSkeleton;
