// biome-ignore lint: This is necessary for it to work
import React from "react";
import PropTypes from "prop-types";
import { useState } from "react";
import { useSelector } from "react-redux";
import { useDispatch } from "react-redux";
import { ButtonClose } from "../../components/Button";
import Input from "../../components/Input";
import Modal from "../../components/Modal";
import { mfetch } from "../../helper";
import { snackAlertError } from "../../slices/mainSlice";

const Mods = ({ community }) => {
  const user = useSelector((state) => state.main.user);
  const dispatch = useDispatch();

  const [addModOpen, setAddModOpen] = useState(false);
  const handleAddModClose = () => setAddModOpen(false);
  const [newModName, setNewModName] = useState("");

  const baseUrl = `/api/communities/${community.id}/mods`;

  const handleAddMod = async (e) => {
    if (e) {
      e.preventDefault();
    }
    try {
      const res = await mfetch(baseUrl, {
        method: "POST",
        body: JSON.stringify({
          username: newModName,
        }),
      });
      if (res.ok) {
        alert(`${newModName} added as a mod of ${community.name}`);
        window.location.reload();
      } else if (res.status === 404) {
        alert("User not found");
      } else {
        throw new Error(await res.text());
      }
    } catch (error) {
      dispatch(snackAlertError(error));
    }
  };

  const handleRemoveMod = async (username) => {
    if (
      !confirm(
        `Are you sure you want to remove ${username} as a moderator of ${community.name}?`,
      )
    ) {
      return;
    }
    try {
      const res = await mfetch(`${baseUrl}/${username}`, {
        method: "DELETE",
      });
      if (res.ok) {
        alert(`${username} removed from moderators`);
        window.location.reload();
      } else {
        throw new Error(await res.text());
      }
    } catch (error) {
      dispatch(snackAlertError(error));
    }
  };

  const { mods } = community;
  let myPos;
  mods.forEach((mod, index) => {
    if (mod.id === user.id) {
      myPos = index;
    }
  });

  return (
    <div className="modtools-content modtools-mods">
      <Modal open={addModOpen} onClose={handleAddModClose}>
        <div className="modal-card">
          <div className="modal-card-head">
            <div className="modal-card-title">Add new moderator</div>
            <ButtonClose onClick={handleAddModClose} />
          </div>
          <form className="modal-card-content" onSubmit={handleAddMod}>
            <Input
              label="Username"
              value={newModName}
              errors={null}
              onChange={(e) => setNewModName(e.target.value)}
              autoFocus
            />
          </form>
          <div className="modal-card-actions">
            <button
              type="button"
              className="button-main"
              disabled={newModName === ""}
              onClick={handleAddMod}
            >
              Add mod
            </button>
            <button type="button" onClick={handleAddModClose}>
              Cancel
            </button>
          </div>
        </div>
      </Modal>
      <div className="modtools-content-head">
        <div className="modtools-title">Mods</div>
        <button
          type="button"
          className="button-main"
          onClick={() => setAddModOpen(true)}
        >
          Add mod
        </button>
      </div>
      <div className="modtools-mods-list">
        <div className="table">
          {mods.map((mod, index) => (
            <div className="table-row" key={mod.id}>
              <div className="table-column">{index}</div>
              <div className="table-column">{mod.username}</div>
              <div className="table-column">
                {(myPos <= index || user.isAdmin) && (
                  <button
                    type="button"
                    className="button-red"
                    onClick={() => handleRemoveMod(mod.username)}
                  >
                    Remove
                  </button>
                )}
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};

Mods.propTypes = {
  community: PropTypes.object.isRequired,
};

export default Mods;
