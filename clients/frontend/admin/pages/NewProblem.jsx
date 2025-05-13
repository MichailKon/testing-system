import React, {useReducer, useState} from "react";
import axios from "axios";
import {useNavigate} from "react-router-dom";
import ChangeAlert, {SendAlertRequest} from "../components/ChangeAlert";
import Body from "../components/Body";
import {ProblemInitialState, ProblemReducer, RenderProblemForm} from "../components/problem/ProblemForm";

export default function NewProblem() {
  const [problem, changeProblem] = useReducer(ProblemReducer, ProblemInitialState())

  const navigate = useNavigate();
  const [alert, setAlert] = useState({
    hasAlert: false,
  })

  const newProblem = () => {
    const apiUrl = `/api/new/problem`
    SendAlertRequest(axios.put(apiUrl, problem), setAlert, (problem) => {
      navigate(`/admin/problem/${problem.id}`);
    })
  }

  return Body(
    [
      {path: "/admin", text: "Admin"},
      {path: "/admin/problems", text: "Problem"},
      {path: `/admin/new/problem`, text: "New"},
    ],
    <div className="bg-white">
      <div className="px-4 px-sm-5 mx-2 pt-4">
        <div className="mb-3 mt-3">
          <h3>Create new problem</h3>
        </div>
      </div>
      <hr className="mt-4 mb-4"/>
      <div className="px-4 px-sm-5 mx-2 pb-5">
        {RenderProblemForm(problem, changeProblem, newProblem, "Create")}
        <div className="row mb-md-3 mb-0">{ChangeAlert(alert)}</div>
      </div>
    </div>
  )
}