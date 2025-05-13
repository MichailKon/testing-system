import React, {useEffect, useState} from "react";
import Body from "../components/Body";
import axios from "axios";
import {Link, useSearchParams} from "react-router-dom";
import Verdict, {SubmissionVerdict} from "../components/Verdict";
import Pagination from "../components/Pagination";
import FiltersForm from "../components/submission/SubmissionsFilters";

export default function Submissions() {
  const [params, setParams] = useSearchParams()
  const page = params.get("page")
  const setPage = (page) => {
    setParams({
      ...Object.fromEntries(params),
      page: page,
    })
  }

  const [state, setState] = useState({
    loading: true,
    submissions: [],
    error: null,
  });

  const [showFilter, setShowFilter] = useState(false);
  const [filterParams, setFilterParams] = useState(Object.fromEntries(params))

  useEffect(() => {
    setFilterParams(Object.fromEntries(params))
    if (!page) {
      setPage(1)
      return
    }

    const apiURL = `/api/get/submissions?count=50&${params.toString()}`
    axios.get(apiURL).then((resp) => {
      setState({
        loading: false,
        submissions: resp.data.response,
        error: resp.data.error,
      })
    }).catch(
      (err) => {
        setState({
          loading: false,
          submissions: [],
          error: err.response.data.error,
        })
      }
    )
  }, [params]);

  const toggleFilter = (e) => {
    e.preventDefault();
    setShowFilter(!showFilter);
  }

  if (state.loading) {
    return wrapContent(null)
  }

  if (state.error) {
    return wrapContent(
      <p className="text-danger">
        {state.error}
      </p>
    )
  }

  return wrapContent(
    <div>
      <div className="row m-0">
        <div className="col ps-0">{Pagination(parseInt(page), setPage)}</div>
        <div className="col-auto text-end">
          <a href="#" onClick={toggleFilter}>{showFilter ? "Hide filters" : "Show filters"}</a>
        </div>
      </div>
      {showFilter ? FiltersForm(filterParams, setFilterParams, setParams) : null}

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
        {
          state.submissions.map(submission =>
            <tr key={submission.id}>
              <th scope="row"><Link to={`/admin/submission/${submission.id}`}>{submission.id}</Link></th>
              <td><Link to={`/admin/problem/${submission.problem_id}`}>{submission.problem_id}</Link></td>
              <td>{submission.language}</td>
              <td>{submission.score}</td>
              <td>{SubmissionVerdict(submission)}</td>
            </tr>
          )
        }
        </tbody>
      </table>
      {Pagination(parseInt(page), setPage)}
    </div>
  );
}

function wrapContent(value) {
  return Body(
    [
      {path: "/admin", text: "Admin"},
      {path: "/admin/submissions", text: "Submissions"},
    ],
    <div className="bg-white">
      <div className="px-4 px-sm-5 mx-2 pt-4">
        <div className="mb-3 mt-3">
          <h3>Submissions</h3>
        </div>
        <div className="mb-3">
          <Link to="/admin/new/submission">Send new submission</Link>
        </div>
      </div>
      <hr className="mt-4 mb-4"/>
      <div className="px-4 px-sm-5 mx-2 pb-5">
        {value}
      </div>
    </div>
  )
}