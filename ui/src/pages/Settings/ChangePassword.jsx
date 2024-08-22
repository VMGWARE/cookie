// biome-ignore lint: This is necessary for it to work
import React from "react";
import { useEffect, useState } from "react";
import { useDispatch } from "react-redux";
import { ButtonClose } from "../../components/Button";
import { InputPassword } from "../../components/Input";
import Modal from "../../components/Modal";
import { ApiError, mfetch } from "../../helper";
import { snackAlert, snackAlertError } from "../../slices/mainSlice";

const ChangePassword = () => {
  const [open, setOpen] = useState(false);
  const handleClose = () => setOpen(false);

  const [password, setPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [repeatPassword, setRepeatPassword] = useState("");
  useEffect(() => {
    setPassword("");
    setNewPassword("");
    setRepeatPassword("");
  }, [open]);

  const dispatch = useDispatch();
  const changePassword = async () => {
    if (newPassword !== repeatPassword) {
      alert("Passwords do not match.");
      return;
    }
    if (newPassword.length < 8) {
      alert("Password too short.");
      return;
    }
    try {
      const res = await mfetch("/api/_settings?action=changePassword", {
        method: "POST",
        body: JSON.stringify({
          password,
          newPassword,
          repeatPassword,
        }),
      });
      if (!res.ok) {
        if (res.status === 401) {
          alert("Incorrect previous password");
          return;
        }
        throw new ApiError(res.status, await res.json());
      }
      dispatch(snackAlert("Password changed succesfully."));
      setOpen(false);
    } catch (error) {
      dispatch(snackAlertError(error));
    }
  };

  return (
    <>
      <button
        type="button"
        onClick={() => setOpen(true)}
        style={{ alignSelf: "flex-start" }}
      >
        Change Password
      </button>
      <Modal open={open} onClose={handleClose}>
        <div className="modal-card modal-change-password">
          <div className="modal-card-head">
            <div className="modal-card-title">Change password</div>
            <ButtonClose onClick={handleClose} />
          </div>
          <div
            className="modal-card-content"
            onKeyDown={(e) => e.key === "Enter" && changePassword()}
          >
            <InputPassword
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              label="Previous password"
              autoFocus
            />
            <InputPassword
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              label="New password"
            />
            <InputPassword
              value={repeatPassword}
              onChange={(e) => setRepeatPassword(e.target.value)}
              label="Repeat password"
            />
          </div>
          <div className="modal-card-actions">
            <button
              type="button"
              className="button-main"
              onClick={changePassword}
            >
              Change password
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

export default ChangePassword;
