import React, {useEffect, useReducer, useState} from "react";
import axios from "axios";
import Body from "../components/Body";
import {Link, useParams} from "react-router-dom";
import Verdict from "../components/Verdict";
import DisplaySourceCode from "../components/submission/DisplaySourceCode";
import {RenderTest, TestDataReducer, WatchTestData} from "../components/submission/TestData";

export default function Submission() {
  const params = useParams();
  const id = params.id;

  const [state, setState] = useState({
    loading: true,
    error: null,
  });

  const [submission, setSubmission] = useState(null)

  useEffect(() => {
    const apiURL = `/api/get/submission/${id}`
    axios.get(apiURL).then((resp) => {
      setState({
        loading: false,
        error: resp.data.error,
      })
      setSubmission(resp.data.response)
    }).catch(
      (err) => {
        setState({
          loading: false,
          error: err.response.data.error,
        })
      }
    )
  }, [id]);

  const [sourceCode, setSourceCode] = useState({
    loaded: false,
    show: false,
  })

  useEffect(() => {
    if (!sourceCode.show || sourceCode.loaded) {
      return;
    }

    if (!submission) {
      return;
    }

    const apiURL = `/api/get/submission/${submission.id}/source`
    axios.get(apiURL).then((resp) => {
      setSourceCode({
        ...sourceCode,
        ...resp.data.response,
        loaded: true,
        error: resp.data.error,
      })
    }).catch(
      (err) => {
        setSourceCode({
          ...sourceCode,
          loaded: true,
          error: err.response.data.error,
        })
      }
    )
  }, [sourceCode, submission]);

  const [tests, changeTests] = useReducer(TestDataReducer, [])
  useEffect(() => {
    WatchTestData(tests, changeTests, submission)
  }, [tests, submission])

  const wrapContent = (content) => {
    return Body(
      [
        {path: "/admin", text: "Admin"},
        {path: "/admin/submissions", text: "Submissions"},
        {path: `/admin/submission/${id}`, text: `${id}`},
      ],
      <div className="bg-white">
        <div className="px-4 px-sm-5 mx-2 pt-4">
          <div className="mb-3 mt-3">
            <h3>Submission {id}</h3>
          </div>
        </div>
        <hr className="mt-4 mb-4"/>
        <div className="px-4 px-sm-5 mx-2 pb-5">
          {content}
        </div>
      </div>
    )
  }

  if (state.loading) {
    return wrapContent(null)
  }

  if (state.error) {
    return wrapContent(<p className="text-danger">{state.error}</p>)
  }

  return wrapContent(
    <div>
      <table className="table mb-3">
        <thead>
        <tr>
          <th scope="row">ID</th>
          <th scope="row">Problem ID</th>
          <th scope="row">Language</th>
          <th scope="row">Score</th>
          <th scope="row">Verdict</th>
        </tr>
        </thead>
        <tbody>
          <tr key={submission.id}>
            <th scope="row">{submission.id}</th>
            <td><Link to={`/admin/problem/${submission.problem_id}`}>{submission.problem_id}</Link></td>
            <td>{submission.language}</td>
            <td>{submission.score}</td>
            <td>{Verdict(submission.verdict)}</td>
          </tr>
        </tbody>
      </table>
      {DisplaySourceCode(id, sourceCode, setSourceCode)}
      <h5 className="mb-3">Test results</h5>
      <table className="table mb-3">
        <thead>
        <tr>
          <th scope="row">#</th>
          <th scope="row">Verdict</th>
          <th scope="row">Points</th>
          <th scope="row">Time</th>
          <th scope="row">Memory</th>
          <th scope="row">Wall time</th>
          <th scope="row">Exit code</th>
          <th scope="row">More info</th>
        </tr>
        </thead>
        <tbody>
        {
          submission.test_results.map(testResult => (
            RenderTest(tests[testResult.test_number - 1], testResult, changeTests)
          ))
        }
        </tbody>
      </table>
    </div>
  )
}