import React from 'react';
import Verdict from "../Verdict";
import axios from "axios";
import RenderResource from "./RenderResource";

export function TestDataReducer(tests, action) {
  if (action.action === "submission") {
    const submission = action.submission
    let t = []
    for (let i = 0; i < submission.test_results.length; i++) {
      t.push({
        show: false,
        requested: false,
        input: {},
        output: {},
        answer: {},
        stderr: {},
        checker: {},
      })
    }
    return t
  }
  let t = tests.slice()
  const testID = action.test
  if (testID < 1 || testID > tests.length) {
    throw Error("Invalid test ID")
  }
  switch (action.action) {
    case "show":
      t[testID - 1].show = !t[testID - 1].show
      break;
    case "request":
      t[testID - 1].requested = true
      break;
    default:
      t[testID - 1][action.action].data = action.data
      t[testID - 1][action.action].error = action.error
      break;
  }
  return t
}

export function RenderTest(test, testResult, changeTest) {
  const toggleTest = (e) => {
    e.preventDefault();
    changeTest({
      action: "show",
      test: testResult.test_number,
    })
  }

  return <>
    <tr key={testResult.test_number}>
      <td>{testResult.test_number}</td>
      <td>{Verdict(testResult.verdict)}</td>
      <td>{testResult.points || ""}</td>
      <td>{testResult.time || ""}</td>
      <td>{testResult.memory || ""}</td>
      <td>{testResult.wall_time || ""}</td>
      <td>{testResult.exit_code == null ? "" : testResult.exit_code}</td>
      <td>
        <a href="#" onClick={toggleTest}>
          {test && test.show ? "Hide test data" : "Show test data"}
        </a>
      </td>
    </tr>
    {test && test.show ? (
      <tr key={`${testResult.test_number}-data`}>
        <td colSpan="8">
          {RenderResource("Error", testResult.error, null)}
          {RenderResource("Input", test["input"].data, test["input"].error)}
          {RenderResource("Output", test["output"].data, test["output"].error)}
          {RenderResource("Stderr", test["stderr"].data, test["stderr"].error)}
          {RenderResource("Answer", test["answer"].data, test["answer"].error)}
          {RenderResource("Checker", test["checker"].data, test["checker"].error)}
        </td>
      </tr>
    ) : null }
  </>
}

export function WatchTestData(tests, changeTest, submission) {
  if (!submission) {
    return
  } else if (tests.length === 0) {
    changeTest({
      action: "submission",
      submission: submission,
    })
    return
  }
  for (let i = 0; i < tests.length; i++) {
    const test = tests[i]

    if (test.show && !test.requested) {
      changeTest({
        action: "request",
        test: i + 1,
      })

      const requestTest = (url, action) => {
        axios.get(url).then(resp => {
          changeTest({
            action: action,
            test: i + 1,
            data: resp.data.response.data,
            error: resp.data.error,
          })
        }).catch(err => {
          changeTest({
            action: action,
            test: i + 1,
            error: err.response.data.error,
          })
        })
      }

      requestTest(`/api/get/problem/${submission.problem_id}/test/${i + 1}/input`, "input")
      requestTest(`/api/get/problem/${submission.problem_id}/test/${i + 1}/answer`, "answer")
      requestTest(`/api/get/submission/${submission.id}/test/${i + 1}/output`, "output")
      requestTest(`/api/get/submission/${submission.id}/test/${i + 1}/check`, "checker")
      requestTest(`/api/get/submission/${submission.id}/test/${i + 1}/stderr`, "stderr")
      return
    }
  }
}