import React from "react";

export default function FiltersForm(params, setFilterParams, setParams) {
  const filter = (e) => {
    e.preventDefault();
    setParams(params);
  }

  const changeParam = (key, value) => {
    let p = {...params}
    if (value === "") {
      delete p[key]
    } else {
      p[key] = value
    }
    setFilterParams(p)
  }

  return (
    <div className="row">
      <div className="col-12 col-md-3"><div className="form-floating">
        <input
          className="form-control"
          type="number"
          name="problem_id"
          id="problem_id"
          placeholder="Problem ID"
          value={params.problem_id || ""}
          onChange={(e) => {
            changeParam("problem_id", e.target.value)
          }}
        />
        <label htmlFor="problem_id">ProblemID</label>
      </div></div>

      <div className="col-12 col-md-3"><div className="form-floating">
        <input
          className="form-control"
          type="text"
          name="language"
          id="language"
          placeholder="Language"
          value={params.language || ""}
          onChange={(e) => {
            changeParam("language", e.target.value)
          }}
        />
        <label htmlFor="language">Language</label>
      </div></div>
      <div className="col-12 col-md-3"><div className="form-floating">
        <input
          className="form-control"
          type="text"
          name="verdict"
          id="verdict"
          placeholder="Verdict"
          value={params.verdict || ""}
          onChange={(e) => {
            changeParam("verdict", e.target.value)
          }}
        />
        <label htmlFor="verdict">Verdict</label>
      </div></div>
      <button className="btn btn-primary col-12 col-md-3" onClick={filter}>Filter</button>
    </div>
  )
}