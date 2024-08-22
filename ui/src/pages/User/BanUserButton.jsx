// biome-ignore lint: This is necessary for it to work
import React from "react";
import PropTypes from "prop-types";
import { useEffect, useState } from "react";
import { useDispatch } from "react-redux";
import { ButtonClose } from "../../components/Button";
import Input from "../../components/Input";
import Modal from "../../components/Modal";
import { mfetch } from "../../helper";
import { snackAlertError } from "../../slices/mainSlice";

const BanUserButton = ({ user }) => {
  const [open, setOpen] = useState();

  const [deleteContentChecked, setDeleteContentChecked] = useState(false);
  const [page, setPage] = useState(1);
  const [banInProgress, setBanInProgress] = useState(false);
  useEffect(() => {
    setPage(1);
    setBanInProgress(false);
  }, [open]);
  const [confirmText, setConfirmText] = useState("");

  const handleClose = () => {
    if (!banInProgress) {
      setOpen(false);
    }
  };

  const dispatch = useDispatch();
  const handleBanUser = async () => {
    if (deleteContentChecked) {
      if (page !== 2) {
        setPage(2);
        return;
      }
      if (confirmText.toLowerCase() !== user.username.toLowerCase()) {
        alert("Usernames do not match!");
        return;
      }
    } else if (!window.confirm("Are you sure?")) {
      return;
    }
    try {
      setBanInProgress(true);
      const body = {
        action: user.isBanned ? "unban_user" : "ban_user",
        username: user.username,
      };
      if (body.action === "ban_user" && deleteContentChecked) {
        body.deleteContentDays = 0;
      }
      const res = await mfetch("/api/_admin", {
        method: "POST",
        body: JSON.stringify(body),
      });
      if (res.status === 200) {
        alert(`User ${user.isBanned ? "un" : ""}banned successfully.`);
        window.location.reload();
      } else {
        alert(`Failed to ban user: ${await res.text()}`);
      }
    } catch (error) {
      dispatch(snackAlertError(error));
    } finally {
      setBanInProgress(false);
    }
  };

  const handleButtonClick = () => {
    if (user.isBanned) {
      handleBanUser();
    } else {
      setOpen(true);
    }
  };

  const renderModalContent = () => {
    if (page === 1) {
      return (
        <div className="modal-card-content">
          <div className="checkbox">
            <input
              id="c1"
              type="checkbox"
              checked={deleteContentChecked}
              onChange={(e) => setDeleteContentChecked(e.target.checked)}
            />
            <label htmlFor="c1">
              Delete all of the user's posts and comments
            </label>
          </div>
        </div>
      );
    }
    return (
      <div className="modal-card-content">
        <p>Enter the username of the user to be banned: </p>
        <Input
          type="text"
          value={confirmText}
          onChange={(e) => setConfirmText(e.target.value)}
        />
      </div>
    );
  };

  const modalTitle = user.isBanned ? "Unban user" : "Ban user";

  return (
    <>
      <button type="button" className="button-red" onClick={handleButtonClick}>
        {modalTitle}
      </button>
      <Modal open={open} onClose={handleClose}>
        <div className="modal-card is-compact-mobile is-desktop-style">
          <div className="modal-card-head">
            <div className="modal-card-title">{modalTitle}</div>
            <ButtonClose onClick={handleClose} />
          </div>
          {renderModalContent()}
          <div className="modal-card-actions">
            <button
              type="button"
              className="button-red"
              onClick={handleBanUser}
              disabled={banInProgress}
            >
              {modalTitle}
            </button>
            <button type="button" onClick={handleClose}>
              Cancel
            </button>
          </div>
        </div>
      </Modal>
    </>
  );
};

BanUserButton.propTypes = {
  user: PropTypes.object.isRequired,
};

export default BanUserButton;
