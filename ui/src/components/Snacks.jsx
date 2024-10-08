// biome-ignore lint: This is necessary for it to work
import React from "react";
import reactDom from "react-dom";
import { useDispatch, useSelector } from "react-redux";
import { ButtonClose } from "./Button";

const Snacks = () => {
  const alerts = useSelector((state) => state.main.alerts);

  const dispatch = useDispatch();
  const handleClose = (id) => {
    dispatch({ type: "main/alertRemoved", payload: id });
  };

  if (alerts.length === 0) {
    return null;
  }

  return reactDom.createPortal(
    <div className="snacks">
      {alerts.map((alert) => (
        <div key={alert.id} className="snack">
          {alert.text}
          <ButtonClose onClick={() => handleClose(alert.id)} />
        </div>
      ))}
    </div>,
    document.getElementById("snacks-root"),
  );
};

export default Snacks;
