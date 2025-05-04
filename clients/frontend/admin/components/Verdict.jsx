import React from "react";

export default function Verdict(verdict) {
  let className;
  switch (verdict) {
    case "OK":
      className = "text-success";
      break;
    case "PT":
      className = "text-warning"
      break;
    case "WA":
    case "WR":
    case "RT":
    case "ML":
    case "TL":
    case "WL":
    case "SE":
    case "CE":
      className = "text-danger"
      break;
    case "RU":
      className = "text-primary"
      break;
    case "CF":
      className = "bg-danger text-black"
      break;
    default:
      className = "text-black"
  }
  return <span className={className}>{verdict}</span>
}