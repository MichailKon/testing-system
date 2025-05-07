import React, {useEffect, useReducer, useState} from "react";
import axios from "axios";
import Body from "../components/Body";
import {Link, useParams} from "react-router-dom";
import Verdict, {SubmissionVerdict} from "../components/Verdict";
import {RenderTest, TestDataReducer, WatchTestData} from "../components/submission/TestData";
import {
  CompilationDataReducer, InitialCompilationData, RenderCompilationData, WatchCompilationData
} from "../components/submission/CompilationData";

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

  const [compilationData, changeCompilationData] = useReducer(CompilationDataReducer, InitialCompilationData())
  useEffect(() => {
    WatchCompilationData(compilationData, changeCompilationData, submission)
  }, [compilationData, changeCompilationData])

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
            <td>{SubmissionVerdict(submission)}</td>
          </tr>
        </tbody>
      </table>
      {RenderCompilationData(compilationData, submission, changeCompilationData)}
      {submission.group_results ? (
        <>
        <h5 className="mb-3">Group results</h5>
        <div className="row">
          <div className="col-12 col-md-10 col-lg-8">
            <table className="table mb-3">
              <thead>
              <tr>
                <th scope="row">Name</th>
                <th scope="row">Score</th>
                <th scope="row">Passed</th>
              </tr>
              </thead>
              <tbody>
              {submission.group_results.map((group, index) => (
                <tr key={index}>
                  <td>{group.group_name}</td>
                  <td>{group.points}</td>
                  <td>{group.passed ? (
                    <span className="text-success">Yes</span>
                  ) : (
                    <span className="text-danger">No</span>
                  )}</td>
                </tr>
              ))}
              </tbody>
            </table>
          </div>
        </div>
        </>
      ) : null}
      {submission.test_results ? (
        <>
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
        </>
      ) : null}
    </div>
  )
}