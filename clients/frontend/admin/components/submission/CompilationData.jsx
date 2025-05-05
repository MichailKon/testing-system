import {Highlight, themes} from "prism-react-renderer";
import React from "react";
import {RenderTest} from "./TestData";
import Verdict from "../Verdict";
import RenderResource from "./RenderResource";
import axios from "axios";

export function InitialCompilationData() {
  return {
    show: false,
    requested: false,
    source: {},
    compilation_output: {},
  }
}

export function CompilationDataReducer(compilation, action) {
  let c = {...compilation};
  switch (action.action) {
    case "show":
      c.show = !c.show
      break;
    case "request":
      c.requested = true
      break;
    default:
      c[action.action].data = action.data
      c[action.action].filename = action.filename;
      c[action.action].error = action.error
      break;
  }
  return c
}

export function RenderCompilationData(compilationData, submission, changeCompilationData) {
  const compilationResult = submission.compilation_result
  if (!compilationResult) {
    return
  }

  const toggleShow = (e) => {
    e.preventDefault();
    changeCompilationData({
      action: "show",
    })
  }

  console.log(compilationData)
  return (
    <>
      <h5 className="mb-3">Compilation result</h5>
      <table className="table mb-3">
        <thead>
        <tr>
          <th scope="row">Verdict</th>
          <th scope="row">Time</th>
          <th scope="row">Memory</th>
          <th scope="row">Wall time</th>
          <th scope="row">Exit code</th>
          <th scope="row">Source code</th>
        </tr>
        </thead>
        <tbody>
        <tr>
          <td>{Verdict(compilationResult.verdict)}</td>
          <td>{compilationResult.time || ""}</td>
          <td>{compilationResult.memory || ""}</td>
          <td>{compilationResult.wall_time || ""}</td>
          <td>{compilationResult.exit_code == null ? "" : compilationResult.exit_code}</td>
          <td><a href="#" onClick={toggleShow}>Show</a></td>
        </tr>
        </tbody>
      </table>
      {compilationData.show ? (
        <>
          {renderSourceCode(submission, compilationData["source"])}
          {RenderResource(
            "Compilation message",
            compilationData["compilation_output"].data,
            compilationData["compilation_output"].error,
          )}
        </>
      ): null}
    </>
  )
}

export function WatchCompilationData(compilationData, changeCompilationData, submission) {
  if (!submission || !submission.compilation_result) {
    return
  }
  if (!compilationData.show || compilationData.requested) {
    return
  }
  changeCompilationData({
    action: "request",
  })
  const requestResource = (url, action) => {
    axios.get(url).then(resp => {
      changeCompilationData({
        action: action,
        data: resp.data.response.data,
        filename: resp.data.response.filename,
        error: resp.data.error,
      })
    }).catch(err => {
      changeCompilationData({
        action: action,
        error: err.response.data.error,
      })
    })
  }
  requestResource(`/api/get/submission/${submission.id}/source`, "source")
  requestResource(`/api/get/submission/${submission.id}/compile_output`, "compilation_output")
}

function renderSourceCode(submission, sourceCode) {
  if (sourceCode.error || !sourceCode.data) {
    return RenderResource("Source code", null, sourceCode.error)
  }
  return (
    <div>
      <p className="m-0">Source code</p>
      <div>
        <Highlight
          language={getLanguageExtension(sourceCode.filename, submission.language)}
          code={sourceCode.data}
          theme={themes.oneLight}
        >
          {({ className, style, tokens, getLineProps, getTokenProps }) => (
            <pre className={className} style={style}>
                {tokens.map((line, i) => (
                  <div key={i} {...getLineProps({ line, key: i })}>
                    <span className="line-num">{i + 1}</span>
                    <span className="line-content">
                      {line.map((token, key) => (
                        <span key={key} {...getTokenProps({ token, key })} />
                      ))}
                    </span>
                  </div>
                ))}
              </pre>
          )}
        </Highlight>
      </div>
    </div>
  )
}

function getLanguageExtension(filename, language) {
  let fileExt = filename.split('.').pop();
  if (fileExt.length > 5) {
    fileExt = language
  }

  let lang

  switch (fileExt) {
    case "c":
    case "gcc":
      lang = "c"
      break;
    case "py":
    case "python":
    case "python3":
    case "pypy":
    case "pypy3":
      lang = "py"
      break;
    case "java":
      lang = "java"
      break;
    case "go":
      lang = "go"
      break;
    case "js":
    case "jsx":
      lang = "jsx"
      break;
    default:
      lang = "cpp"
      break;
  }
  return lang
}