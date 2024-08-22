// biome-ignore lint: This is necessary for it to work
import React from "react";
import { useEffect, useState } from "react";
import { Helmet } from "react-helmet-async";
import { useDispatch, useSelector } from "react-redux";
import {
  Link,
  Redirect,
  Route,
  Switch,
  useLocation,
  useParams,
  useRouteMatch,
} from "react-router-dom";
import Sidebar from "../../components/Sidebar";
import { ApiError, mfetch } from "../../helper";
import { communityAdded, selectCommunity } from "../../slices/communitiesSlice";
import { snackAlertError } from "../../slices/mainSlice";
import PageNotLoaded from "../PageNotLoaded";
import Banned from "./Banned";
import Mods from "./Mods";
import Removed from "./Removed";
import Reports from "./Reports";
import Rules from "./Rules";
import Settings from "./Settings";

function isActiveCls(className, isActive, activeClass = "is-active") {
  return className + (isActive ? ` ${activeClass}` : "");
}

const Modtools = () => {
  const dispatch = useDispatch();
  const { name: communityName } = useParams();

  const user = useSelector((state) => state.main.user);

  const community = useSelector(selectCommunity(communityName));
  const [loading, setLoading] = useState(community ? "loaded" : "loading");
  useEffect(() => {
    if (community) {
      return;
    }
    (async () => {
      setLoading("loading");
      try {
        const res = await mfetch(
          `/api/communities/${communityName}?byName=true`,
        );
        if (!res.ok) {
          if (res.status === 404) {
            setLoading("notfound");
            return;
          }
          throw new ApiError(res.status, await res.json());
        }
        const rcomm = await res.json();
        dispatch(communityAdded(rcomm));
        setLoading("loaded");
      } catch (error) {
        setLoading("error");
        dispatch(snackAlertError(error));
      }
    })();
  }, [name, community]);

  const { path } = useRouteMatch();
  const { pathname } = useLocation();

  if (loading !== "loaded") {
    return <PageNotLoaded loading={loading} />;
  }

  if (!(community.userMod || user?.isAdmin)) {
    return (
      <div className="page-content page-full">
        <h1>Forbidden!</h1>
        <div>
          <Link to="/">Go home</Link>.
        </div>
      </div>
    );
  }

  return (
    <div className="page-content wrap modtools">
      <Helmet>
        <title>Modtools</title>
      </Helmet>
      <Sidebar />
      <div className="modtools-head">
        <h1>
          <Link to={`/${CONFIG.communityPrefix}${communityName}`}>
            {communityName}{" "}
          </Link>
          Modtools
        </h1>
      </div>
      <div className="modtools-dashboard">
        <div className="sidebar">
          <Link
            className={isActiveCls(
              "sidebar-item",
              pathname === "/modtools/settings",
            )}
            to={`/${CONFIG.communityPrefix}${communityName}/modtools/settings`}
          >
            Community settings
          </Link>
          <div className="sidebar-topic">Content</div>
          <Link
            className={isActiveCls(
              "sidebar-item",
              pathname === "/modtools/reports",
            )}
            to={`/${CONFIG.communityPrefix}${communityName}/modtools/reports`}
          >
            Reports
          </Link>
          <Link
            className={isActiveCls(
              "sidebar-item",
              pathname === "/modtools/removed",
            )}
            to={`/${CONFIG.communityPrefix}${communityName}/modtools/removed`}
          >
            Removed
          </Link>
          <Link
            className={isActiveCls(
              "sidebar-item",
              pathname === "/modtools/locked",
            )}
            to={`/${CONFIG.communityPrefix}${communityName}/modtools/locked`}
          >
            Locked
          </Link>
          <div className="sidebar-topic">Users</div>
          <Link
            className={isActiveCls(
              "sidebar-item",
              pathname === "/modtools/banned",
            )}
            to={`/${CONFIG.communityPrefix}${communityName}/modtools/banned`}
          >
            Banned
          </Link>
          <Link
            className={isActiveCls(
              "sidebar-item",
              pathname === "/modtools/mods",
            )}
            to={`/${CONFIG.communityPrefix}${communityName}/modtools/mods`}
          >
            Moderators
          </Link>
          <div className="sidebar-topic">Rules</div>
          <Link
            className={isActiveCls(
              "sidebar-item",
              pathname === "/modtools/rules",
            )}
            to={`/${CONFIG.communityPrefix}${communityName}/modtools/rules`}
          >
            Rules
          </Link>
        </div>
        <Switch>
          <Route exact path={path}>
            <Redirect
              to={`/${CONFIG.communityPrefix}${communityName}/modtools/settings`}
            />
          </Route>
          <Route exact path={`${path}/settings`}>
            <Settings community={community} />
          </Route>
          <Route path={`${path}/reports`}>
            <Reports community={community} />
          </Route>
          <Route path={`${path}/removed`}>
            <Removed community={community} filter="deleted" title="Removed" />
          </Route>
          <Route path={`${path}/locked`}>
            <Removed community={community} filter="locked" title="Locked" />
          </Route>
          <Route path={`${path}/banned`}>
            <Banned community={community} />
          </Route>
          <Route path={`${path}/mods`}>
            <Mods community={community} />
          </Route>
          <Route path={`${path}/rules`}>
            <Rules community={community} />
          </Route>
          <Route path="*">
            <div className="modtools-content flex flex-center">Not found.</div>
          </Route>
        </Switch>
      </div>
    </div>
  );
};

export default Modtools;
