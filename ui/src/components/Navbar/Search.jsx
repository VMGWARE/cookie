// biome-ignore lint: This is necessary for it to work
import React from "react";
import PropTypes from "prop-types";
import { useCallback, useEffect, useRef, useState } from "react";
import { onKeyEnter } from "../../helper";
import { ButtonClose, ButtonSearch } from "../Button";
import Modal from "../Modal";

const Search = () => {
  const [searchQuery, setSearchQuery] = useState("");

  const inputRef = useRef(null);
  useEffect(() => {
    const onKeyDown = (e) => {
      const active = document.activeElement;
      if (active.nodeName === "BODY" && e.key === "/") {
        inputRef.current.focus();
        e.preventDefault();
      }
    };
    document.addEventListener("keydown", onKeyDown);
    return () => {
      document.removeEventListener("keydown", onKeyDown);
    };
  }, []);

  const [searchModalOpen, setSearchModalOpen] = useState(false);
  const getGoogleUrl = (query) => {
    const q = encodeURIComponent(`${query} site:${window.location.hostname}`);
    return `https://www.google.com/search?q=${q}`;
  };

  const getApiUrl = (query) => `/api/search?q=${encodeURIComponent(query)}&index=communities`;

  // Fallback on Google search until search is implemented.
  const handleSearch = () => {
    const win = window.open(getGoogleUrl(searchQuery), "_blank");
    if (!win || win.closed || typeof win.closed === "undefined") {
      // poppup was blocked
      setSearchModalOpen(true);
    }
  };
  const linkRef = useCallback((node) => {
    if (node !== null) {
      setTimeout(() => node.focus(), 10);
    }
  });

  return (
    <>
      <Modal open={searchModalOpen} onClose={() => setSearchModalOpen(false)}>
        <div className="modal-card">
          <div className="modal-card-head">
            <div className="modal-card-title">Search</div>
            <ButtonClose onClick={() => setSearchModalOpen(false)} />
          </div>
          <div className="modal-card-content">
            <p style={{ marginBottom: "var(--gap)" }}>
              {`Search is yet to be implemented, but you can click the button below to search on
              Google. It'll show only results from this website.`}
            </p>
            <a
              className="button button-main"
              href={getGoogleUrl(searchQuery)}
              target="_blank"
              rel="noreferrer"
              ref={linkRef}
              onClick={() => setSearchModalOpen(false)}
            >
              Search on Google for now
            </a>
          </div>
        </div>
      </Modal>
      <div className="input-search">
        <input
          ref={inputRef}
          type="text"
          placeholder="Search"
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          onKeyDown={(e) => onKeyEnter(e, handleSearch)}
          onSubmit={handleSearch}
        />
        <ButtonSearch onClick={handleSearch} />
      </div>
    </>
  );
};

Search.propTypes = {
  autoFocus: PropTypes.bool,
};

export default Search;
