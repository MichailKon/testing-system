import React, {useEffect, useReducer, useState} from "react";
import {Link, useParams} from "react-router-dom";
import axios from "axios";
import {ProblemInitialState, ProblemReducer, RenderProblemForm} from "../components/problem/ProblemForm";
import Body from "../components/Body";
import ChangeAlert, {SendAlertRequest} from "../components/ChangeAlert";

export default function Problems() {
  const params = useParams();
  const id = params.id;

  const [state, setState] = useState({
    loading: true,
    error: null,
  });

  const [problem, changeProblem] = useReducer(ProblemReducer, ProblemInitialState())

  useEffect(() => {
    const apiURL = `/api/get/problem/${id}`
    axios.get(apiURL).then((resp) => {
      setState({
        loading: false,
        error: resp.data.error,
      })
      changeProblem({
        action: "problem",
        problem: resp.data.response,
      })
    }).catch(
      (err) => {
        setState({
          loading: false,
          error: err.response.data.error,
        })
      }
    )
  }, [id]);

  const [alert, setAlert] = useState({
    hasAlert: false,
  })

  const modifyProblem = () => {
    const apiUrl = `/api/modify/problem/${id}`
    SendAlertRequest(axios.post(apiUrl, problem), setAlert, null)
  }

  const wrapContent = (content) => {
    return Body(
      [
        {path: "/admin", text: "Admin"},
        {path: "/admin/problems", text: "Problems"},
        {path: `/admin/problem/${id}`, text: `${id}`},
      ],
      <div className="bg-white">
        <div className="px-4 px-sm-5 mx-2 pt-4">
          <div className="mb-3 mt-3">
            <h3>Problem {id}</h3>
          </div>
          <div className="mb-3">
            <Link to={`/admin/submissions?problem_id=${problem.id}`}>Submissions</Link>
          </div>
          <div className="mb-3">
            <Link to={`/admin/new/submission?problem_id=${problem.id}`}>Send submission</Link>
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
      {RenderProblemForm(problem, changeProblem, modifyProblem, "Save")}
      <div className="row mb-md-3 mb-0">{ChangeAlert(alert)}</div>
    </div>
  )
}
