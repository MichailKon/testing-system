import React, {useEffect, useState} from "react";
import Body from "../components/Body";
import axios from "axios";
import {Link, useSearchParams} from "react-router-dom";

export default function Problems() {
  const [params, setParams] = useSearchParams()
  const page = params.get("page")

  const [state, setState] = useState({
    loading: true,
    problems: [],
    error: null,
  });

  useEffect(() => {
    if (!page) {
      setParams({"page": "1"})
      return
    }
    const apiURL = `/api/get/problems?count=20&page=${page}`
    axios.get(apiURL).then((resp) => {
      setState({
        loading: false,
        problems: resp.data.response,
        error: resp.data.error,
      })
    }).catch(
      (err) => {
        setState({
          loading: false,
          problems: [],
          error: err.response.data.error,
        })
      }
    )
  }, [page]);

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
    <table className="table m-0">
      <thead>
        <tr>
          <th scope="row">ID</th>
          <th scope="row">Name</th>
        </tr>
      </thead>
      <tbody>
      {
        state.problems.map(problem =>
          <tr key={problem.id}>
            <th scope="row"><Link to={`/admin/problem/${problem.id}`}>{problem.id}</Link></th>
            <td>{problem.name}</td>
          </tr>
        )
      }
      </tbody>
    </table>
  );
}

function wrapContent(value) {
  return Body(
    [
      {path: "/admin", text: "Admin"},
      {path: "/admin/problems", text: "Problems"},
    ],
    <div className="bg-white">
      <div className="px-4 px-sm-5 mx-2 pt-4">
        <div className="mb-3 mt-3">
          <h3>Problems</h3>
        </div>
        <div className="mb-3">
          <Link to="/admin/new/problem">Create new</Link>
        </div>
      </div>
      <hr className="mt-4 mb-4"/>
      <div className="px-4 px-sm-5 mx-2 pb-5">
        {value}
      </div>
    </div>
)
}