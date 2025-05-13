import React from "react";
import {Link} from "react-router-dom";

export default function Links(links) {
  return <div>
    {links.map((link, index) =>
      <div key={index} className="row border border-primary px-2 py-2 mb-3 rounded-3">
        <Link to={link.to}>{link.text}</Link>
      </div>
    )}
  </div>
}