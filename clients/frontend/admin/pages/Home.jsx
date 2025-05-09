import React from "react";
import { Link } from "react-router-dom";
import Body from "../components/Body";
import Links from "../components/Links";

export default function Home() {
  return (
    Body(
      [
        {path: "/admin", text: "Admin"}
      ],
      <div className="bg-white">
        <div className="px-4 px-sm-5 mx-2 pt-4">
          <div className="mb-3 mt-3">
            <h3>Website admin</h3>
          </div>
        </div>
        <hr className="mt-4 mb-4"/>
        <div className="px-4 px-sm-5 mx-2 pb-5">
          {Links([
            {to: "/admin/problems", text: "Problems"},
            {to: "/admin/submissions", text: "Submissions"},
            {to: "/admin/status", text: "Status"},
          ])}
        </div>
      </div>
    )
  );
}