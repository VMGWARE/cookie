// biome-ignore lint: This is necessary for it to work
import React from "react";
import PropTypes from "prop-types";
import { useEffect, useState } from "react";
import { useDispatch } from "react-redux";
import { useSelector } from "react-redux";
import { useLocation } from "react-router";
import { useHistory } from "react-router-dom";
import Feed from "../components/Feed";
import { MemorizedPostCard } from "../components/PostCard/PostCard";
import SelectBar from "../components/SelectBar";
import { mfetchjson } from "../helper";
import { useCanonicalTag } from "../hooks";
import {
  FeedItem,
  feedInViewItemsUpdated,
  feedItemHeightChanged,
  feedReloaded,
  feedUpdated,
  selectFeed,
  selectFeedInViewItems,
} from "../slices/feedsSlice";
import { snackAlertError } from "../slices/mainSlice";
import WelcomeBanner from "../views/WelcomeBanner";

const sortOptions = [
  { text: "Hot", id: "hot" },
  { text: "Activity", id: "activity" },
  { text: "New", id: "latest" },
  { text: "Day", id: "day" },
  { text: "Week", id: "week" },
  { text: "Month", id: "month" },
  { text: "Year", id: "year" },
  // { text: 'All', id: 'all' },
];
const sortDefault = CONFIG.defaultFeedSort;
const baseUrl = "/api/posts";

export const homeReloaded = (homeFeed = "all", rememberFeedSort = false) => {
  const params = new URLSearchParams();
  let sort = sortDefault;
  if (rememberFeedSort) {
    sort = window.localStorage.getItem("feedSort") || sortDefault;
  }
  params.set("sort", sort);
  if (homeFeed === "subscriptions") {
    params.set("feed", "home");
  }
  return feedReloaded(`${baseUrl}?${params.toString()}`);
};

function useFeedSort(rememberLastSort = false) {
  const location = useLocation();

  let sortSaved;
  if (rememberLastSort) {
    sortSaved = window.localStorage.getItem("feedSort");
  } else {
    const params = new URLSearchParams(location.search);
    sortSaved = params.get("sort");
  }
  sortSaved = sortSaved || sortDefault;

  // If sortOptions.to is preset, use query parameters and go to the link.
  for (const option of sortOptions) {
    if (rememberLastSort) {
      option.to = undefined;
    } else if (option.id === sortDefault) {
      option.to = location.pathname;
    } else {
      option.to = `${location.pathname}?sort=${option.id}`;
    }
  }

  const history = useHistory();
  const [sort, _setSort] = useState(sortSaved);

  const setSort = (newSort) => {
    _setSort(newSort);
    if (rememberLastSort) {
      window.localStorage.setItem("feedSort", newSort);
    } else {
      let to = "#";
      for (const option of sortOptions.filter(
        (option) => option.id === newSort,
      )) {
        to = option.to;
      }
      history.replace(to);
    }
  };

  useEffect(() => {
    if (!rememberLastSort) {
      const params = new URLSearchParams(location.search);
      _setSort(params.get("sort") || sortDefault);
    }
  }, [location]);

  return [sort, setSort];
}

const PostsFeed = ({ feedType = "all", communityId = null }) => {
  const dispatch = useDispatch();
  // const history = useHistory();

  const user = useSelector((state) => state.main.user);
  const loggedIn = user !== null;

  const location = useLocation();
  // const params = new URLSearchParams(location.search);
  // const sort = params.get('sort') || sortDefault;
  const [sort, setSort] = useFeedSort(user?.rememberFeedSort);

  // The ordering of the urlparams here is important because the items are
  // stored by the url as the key.
  const urlParams = new URLSearchParams();
  urlParams.set("sort", sort);
  if (loggedIn && feedType === "subscriptions") {
    urlParams.set("feed", "home");
  }
  if (communityId !== null) {
    urlParams.set("communityId", communityId);
  }
  const endpoint = `${baseUrl}?${urlParams.toString()}`; // api endpoint.

  // Only called on button clicks (not history API changes)
  const handleSortChange = (value) => {
    setSort(value);
    dispatch(feedReloaded(endpoint));
  };

  const feed = useSelector(selectFeed(endpoint));
  const setFeed = (res, url) => {
    const feedItems = (res.posts ?? []).map(
      (post) => new FeedItem(post, "post", post.publicId),
    );
    dispatch(feedUpdated(url, feedItems, res.next));
  };

  const loading = feed ? feed.loading : true;
  const [_, setError] = useState(null);
  useEffect(() => {
    if (!loading) {
      return;
    }
    (async () => {
      try {
        const res = await mfetchjson(endpoint);
        setFeed(res, endpoint);
      } catch (error) {
        setError(error);
        dispatch(snackAlertError(error));
      }
    })();
  }, [endpoint, loading]);

  const [feedReloading, setFeedReloading] = useState(false);
  const fetchNextPosts = async () => {
    if (feedReloading) {
      return;
    }
    setFeedReloading(true);
    try {
      const params = new URLSearchParams(urlParams.toString());
      params.set("next", feed.next);
      const res = await mfetchjson(`${baseUrl}?${params.toString()}`);
      setFeed(res, endpoint);
    } catch (error) {
      dispatch(snackAlertError(error));
    } finally {
      setFeedReloading(false);
    }
  };

  const handleItemHeightChange = (height, item) => {
    dispatch(feedItemHeightChanged(item.key, height));
  };
  const handleRenderItem = (item, index) => (
    <MemorizedPostCard
      initialPost={item.item}
      index={index}
      disableEmbeds={user?.embedsOff}
    />
  );
  const itemsInitiallyInView = useSelector(selectFeedInViewItems(endpoint));
  const handleSaveVisibleItems = (items) => {
    dispatch(feedInViewItemsUpdated(endpoint, items));
  };

  const canonicalUrl = () => {
    const sortValid =
      sortOptions.filter((option) => option.id === sort).length !== 0;
    if (!sortValid) {
      return "";
    }
    const url = window.location;
    const search = sort === sortDefault ? "" : `?sort=${sort}`;
    return url.origin + url.pathname + search;
  };
  useCanonicalTag(canonicalUrl(), [location]);

  const posts = feed ? feed.items : [];
  let name = "Posts";
  if (!communityId) {
    if (feedType === "all") {
      name = "Home";
    } else if (feedType === "subscriptions") {
      name = "Subscriptions";
    }
  }

  return (
    <div className="posts-feed">
      {/*<PostsFilterBar name={name} sort={sort} onChange={handleSortChange} />*/}
      <SelectBar
        name={name}
        options={sortOptions}
        value={sort}
        onChange={handleSortChange}
      />
      <Feed
        loading={loading}
        items={posts}
        hasMore={Boolean(feed ? feed.next : false)}
        onNext={fetchNextPosts}
        isMoreItemsLoading={feedReloading}
        onRenderItem={handleRenderItem}
        onItemHeightChange={handleItemHeightChange}
        itemsInitiallyInView={itemsInitiallyInView}
        onSaveVisibleItems={handleSaveVisibleItems}
        banner={loggedIn ? null : <WelcomeBanner className="is-m is-in-feed" />}
      />
    </div>
  );
};

PostsFeed.propTypes = {
  communityId: PropTypes.string,
  feedType: PropTypes.oneOf(["all", "subscriptions", "community"]),
};

export default PostsFeed;

const PostsFilterBar = ({ name, sort = "latest", onChange }) => {
  return (
    <SelectBar
      name={name}
      options={sortOptions}
      value={sort}
      onChange={onChange}
    />
  );
};

PostsFilterBar.propTypes = {
  name: PropTypes.string,
  sort: PropTypes.string,
  onChange: PropTypes.func,
  rememberLastSort: PropTypes.bool,
};
