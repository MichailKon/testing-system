import React from "react";
import {Link} from "react-router-dom";

export default function Body(navs, value) {
  const lastNav = navs[navs.length - 1];
  navs = navs.slice(0, -1);
  return (
    <div>
    <div className="container-fluid nav-links">
      <div className="container-lg px-5 py-2">
        <div className="d-flex">
          {navs.map((nav, i) => (
            <div
              key={i}
              className="text-white d-block"
            >
              <div className="d-flex">
                <Link className="nav-link d-block pe-1" to={nav.path}>{nav.text}</Link>
                <div className="d-flex d-block pe-1">/</div>
              </div>
            </div>
          ))}
          <div className="d-inline-block text-white text-opacity-75">{lastNav.text}</div>
        </div>
      </div>
    </div>
      <div className="container-lg px-xl-5 px-0 my-5">
      {value}
    </div>
    </div>
  );
}