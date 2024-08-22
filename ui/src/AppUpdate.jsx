// biome-ignore lint: This is necessary for it to work
import React from "react";
import { useEffect, useState } from "react";
import { ButtonClose } from "./components/Button";
import Modal from "./components/Modal";
import { useIsMobile } from "./hooks";

export const forceSwUpdate = async () => {
  if ("serviceWorker" in navigator) {
    console.info("Force updating service worker");
    const registration = await navigator.serviceWorker.ready;
    return registration.update();
  }
};

const AppUpdate = () => {
  const [swWaiting, setSwWaiting] = useState(false);

  useEffect(() => {
    const id = setInterval(() => forceSwUpdate(), 1000 * 60 * 2); // every 2 minutes
    const listener = async () => {
      if (!document.hidden) {
        await forceSwUpdate();
      }
    };
    document.addEventListener("visibilitychange", listener);
    return () => {
      clearInterval(id);
      document.removeEventListener("visibilitychange", listener);
    };
  }, []);

  useEffect(() => {
    let effectCancelled = false;

    const detectSwUpdate = async () => {
      const registration = await navigator.serviceWorker.ready;
      registration.addEventListener("updatefound", () => {
        const newSw = registration.installing;
        newSw.addEventListener("statechange", () => {
          if (newSw.state === "installed") {
            if (!effectCancelled) {
              // New service worker is installed, but waiting activation
              // newSw.current = newSw;
              setSwWaiting(true);
            }
          }
        });
      });
    };

    if ("serviceWorker" in navigator) {
      detectSwUpdate();
    }

    return () => {
      effectCancelled = true;
    };
  }, []);

  function handleReload() {
    window.location.reload();
  }

  const [modalOpen, setModalOpen] = useState(true);
  const handleClose = () => setModalOpen(false);
  const isMobile = useIsMobile();

  if (isMobile && swWaiting) {
    /*
    return (
      <div style={{ marginTop: 'var(--navbar-height)' }}>
        <button className="button-main" onClick={handleReload}>
          Reload
        </button>
      </div>
    );
    */
    return (
      <Modal open={modalOpen} onClose={handleClose} noOuterClickClose>
        <div
          className="modal-card is-compact-mobile is-center"
          style={{ minWidth: "300px" }}
        >
          <div className="modal-card-head">
            <div className="modal-card-title">Update available!</div>
            <ButtonClose onClick={handleClose} />
          </div>
          <div className="modal-card-content">
            <p>
              A new version of this app is available. Reload the page to update.
              It won't take more than a second.
            </p>
          </div>
          <div className="modal-card-actions">
            <button
              type="button"
              className="button-main"
              onClick={handleReload}
            >
              Reload
            </button>
            <button type="button" onClick={handleClose}>
              Not now
            </button>
          </div>
        </div>
      </Modal>
    );
  }

  return null;
};

export default AppUpdate;
