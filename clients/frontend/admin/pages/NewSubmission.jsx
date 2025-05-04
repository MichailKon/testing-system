import React, { useState } from 'react';
import Body from "../components/Body";
import ProblemForm from "../components/problem/ProblemForm";
import ChangeAlert, {SendAlertRequest} from "../components/ChangeAlert";
import axios from "axios";
import {useNavigate, useSearchParams} from "react-router-dom";

export default function NewSubmission() {
  const [params, setParams] = useSearchParams()

  const [alert, setAlert] = useState({
    hasAlert: false,
  })

  const newSubmission = (e) => {
    e.preventDefault();
    const data = Object.fromEntries(new FormData(e.target).entries());
    let formData = new FormData();
    formData.append("problem_id", data.problem_id);
    formData.append("language", data.language);
    formData.append("solution", data.solution);

    const apiUrl = `/api/new/submission`
    SendAlertRequest(axios.putForm(apiUrl, formData), setAlert, (submissionID) => {
      setAlert({
        hasAlert: true,
        ok: true,
        message: `Submission ${submissionID} is created`,
      })
    })
  }

  return Body(
    [
      {path: "/admin", text: "Admin"},
      {path: "/admin/submissions", text: "Submission"},
      {path: `/admin/new/submission`, text: "New"},
    ],
    <div className="bg-white">
      <div className="px-4 px-sm-5 mx-2 pt-4">
        <div className="mb-3 mt-3">
          <h3>Send new submission</h3>
        </div>
      </div>
      <hr className="mt-4 mb-4"/>
      <div className="px-4 px-sm-5 mx-2 pb-5">
        <form onSubmit={newSubmission} encType="multipart/form-data">
          <div className="row mb-md-3 mb-0">
            <label htmlFor="problem_id" className="col-xl-3 col-md-4 col-form-label pb-0 pb-md-2 pt-3">Problem
              ID</label>
            <div className="col-md-8 col-xl-6 col-form-label">
              <input
                type="number"
                className="form-control"
                name="problem_id"
                id="problem_id"
                required={true}
                defaultValue={params.problem_id || ""}
              />
            </div>
          </div>
          <div className="row mb-md-3 mb-0">
            <label htmlFor="language" className="col-xl-3 col-md-4 col-form-label pb-0 pb-md-2 pt-3">Language</label>
            <div className="col-md-8 col-xl-6 col-form-label">
              <input
                type="text"
                className="form-control"
                id="language"
                name="language"
                required={true}
              />
            </div>
          </div>
          <div className="row mb-md-3 mb-0">
            <label htmlFor="solution" className="col-xl-3 col-md-4 col-form-label pb-0 pb-md-2 pt-3">Solution
              file</label>
            <div className="col-md-8 col-xl-6 col-form-label">
              <input
                type="file"
                className="form-control"
                id="solution"
                name="solution"
                required={true}
              />
            </div>
          </div>
          <div className="row mt-md-4 mt-2">
            <div className="col-xl-3 d-xl-block d-none"></div>
            <div className="text-center col-12 col-xl-6">
              <button type="submit" className="btn btn-primary w-100">Send</button>
            </div>
          </div>
        </form>
        <div className="row mb-md-3 mb-0">{ChangeAlert(alert)}</div>
      </div>
    </div>
  )
}