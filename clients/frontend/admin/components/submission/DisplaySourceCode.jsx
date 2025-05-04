import {Highlight, themes} from "prism-react-renderer";
import React from "react";

export default function DisplaySourceCode(submission, sourceCode, setSourceCode) {
  const show = sourceCode.show
  const setShow = (val) => {
    setSourceCode({
      ...sourceCode,
      show: val,
    })
  }

  const toggleSourceCode = (e) => {
    e.preventDefault()
    setShow(!show)
  }

  const wrapContent = (value) => (
    <>
      <div className="row mx-0 py-2 mb-4 border-bottom">
        <div className="col ps-0"><b>Source code</b></div>
        <div className="col-auto text-end">
          <a href="#" onClick={toggleSourceCode}>{show ? "Hide" : "Show"}</a>
        </div>
      </div>
      {value}
    </>
  )

  if (!show) {
    return wrapContent(null)
  } else if (!sourceCode.loaded) {
    return wrapContent(null)
  } else if (sourceCode.error) {
    return wrapContent(<p className="text-danger">{sourceCode.error}</p>)
  } else {

    return wrapContent(
      <div>
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