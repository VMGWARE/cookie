// biome-ignore lint: This is necessary for it to work
import React from "react";
import { useSelector } from "react-redux";
import { Redirect } from "react-router-dom";
import LoginForm from "../views/LoginForm";

const Login = () => {
  const user = useSelector((state) => state.main.user);
  const loggedIn = user !== null;

  if (loggedIn) {
    return <Redirect to="/" />;
  }

  return (
    <div className="page-content page-login wrap">
      <div className="card login-card">
        <div className="title">Login to continue</div>
        <LoginForm />
      </div>
    </div>
  );
};

export default Login;
