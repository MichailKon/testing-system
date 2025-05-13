import React from "react";

export function SubmissionVerdict(submission) {
  const verdict = submission.verdict
  if (verdict === "RU") {
    let currentTest = 0
    if (submission.current_test) {
      currentTest = submission.current_test
    } else if (submission.test_results) {
      currentTest = submission.test_results.length
    }
    if (currentTest === 0) {
      return Verdict(verdict, " (compiling)")
    } else {
      return Verdict(verdict, ` (${currentTest})`)
    }
  }
  return Verdict(verdict)
}

export default function Verdict(verdict, text = null) {
  let className;
  switch (verdict) {
    case "OK":
    case "CD":
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
  return <span className={className}>{verdict}{text}</span>
}