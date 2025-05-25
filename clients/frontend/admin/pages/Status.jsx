import React, {useEffect, useState} from 'react';
import axios from "axios";
import Body from "../components/Body";
import {Link} from "react-router-dom";
import ChangeAlert, {SendAlertRequest} from "../components/ChangeAlert";

export default function Status() {
  const [state, setState] = useState({
    loading: true,
  })

  useEffect(() => {
    if (!state.loading) {
      return
    }
    const apiURL = `/api/get/master_status`
    axios.get(apiURL).then((resp) => {
      setState({
        loading: false,
        status: resp.data.response,
        error: resp.data.error,
      })
    }).catch(
      (err) => {
        setState({
          loading: false,
          error: err.response.data.error,
        })
      }
    )
  }, [state]);

  const [resetInvokerCacheAlert, setResetInvokerCacheAlert] = useState({})

  const resetInvokerCache = (e) => {
    e.preventDefault()
    const apiUrl = `/api/reset/invoker_cache`
    SendAlertRequest(axios.post(apiUrl), setResetInvokerCacheAlert, (_) => {
      setResetInvokerCacheAlert({
        hasAlert: true,
        ok: true,
        message: "Reseted invoker cache",
      })
    })
  }

  if (state.loading) {
    return wrapContent(null)
  }

  const status = state.status;

  if (state.error) {
    return wrapContent(<p className="text-danger">{state.error}</p>)
  }

  return wrapContent(
    <div>
      <h5 className="mb-4"><Link to="/admin/submissions?verdict=RU">Submissions
        testing</Link>: {status.testing_submissions.length}</h5>
      <h5 className="mb-4">Invokers:</h5>
      <div className="mb-3">
        <a href="#" className="mb-3" onClick={resetInvokerCache}>Reset cache</a>
      </div>
      <div className="row">{ChangeAlert(resetInvokerCacheAlert)}</div>
      <div className="mx-3 mx-md-4">
        {status.invokers.map((invoker, index) => (
          <div key={index}>
            <div className="row mb-2">
              <div className="col-xl-2 col-sm-3 col-12"><b>Address:</b></div>
              <div className="col-sm-9 col-12">
                {invoker.address}
              </div>
            </div>
            <div className="row mb-2">
              <div className="col-xl-2 col-sm-3 col-12"><b>Time added:</b></div>
              <div className="col-sm-9 col-12">
                {invoker.time_added}
              </div>
            </div>
            <div className="row mb-2">
              <div className="col-xl-2 col-sm-3 col-12"><b>Max new jobs:</b></div>
              <div className="col-sm-9 col-12">
                {invoker.max_new_jobs}
              </div>
            </div>
            <table className="table mb-3">
              <thead>
              <tr>
                <th scope="row">Job Type</th>
                <th scope="row">Submission</th>
                <th scope="row">Test</th>
                <th scope="row">ID</th>
              </tr>
              </thead>
              <tbody>
              {invoker.testing_jobs.map((job, jobIndex) => (
                <tr key={`${index}-${jobIndex}`}>
                  <td>{getJobType(job.type)}</td>
                  <td>{job.submit_id}</td>
                  <td>{job.test ? job.test : ""}</td>
                  <td>{job.id}</td>
                </tr>
              ))}
              </tbody>
            </table>
          </div>
        ))}
      </div>
    </div>
  )
}

function getJobType(jobType) {
  switch (jobType) {
    case 1:
      return "Compilation";
    case 2:
      return "Test";
    default:
      return "Unknown";
  }
}

function wrapContent(value) {
  return Body(
    [
      {path: "/admin", text: "Admin"},
      {path: "/admin/status", text: "Status"},
    ],
    <div className="bg-white">
      <div className="px-4 px-sm-5 mx-2 pt-4">
        <div className="mb-3 mt-3">
          <h3>Master status</h3>
        </div>
      </div>
      <hr className="mt-4 mb-4"/>
      <div className="px-4 px-sm-5 mx-2 pb-5">
        {value}
      </div>
    </div>
  )
}