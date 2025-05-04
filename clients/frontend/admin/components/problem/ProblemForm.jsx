import React from "react";

export default function ProblemForm(
  problem,
  setProblem,
  submitAction,
  buttonText
) {
  const action = (event) => {
    event.preventDefault();
    submitAction();
  }

  const formRow = (name, displayName, type, required) => {
    return <div className="row mb-md-3 mb-0">
      <label htmlFor={name} className="col-xl-3 col-md-4 col-form-label pb-0 pb-md-2 pt-3">{displayName}</label>
      <div className="col-md-8 col-xl-6 col-form-label">
        <input
          type={type}
          className="form-control"
          id={name}
          name={name}
          value={problem[name] || ""}
          required={required}
          onChange={(e) => {
            if (!required && e.target.value === "") {
              let prob = {...problem};
              delete prob[name];
              setProblem(prob)
              return;
            }
            if (type === "text") {
              setProblem({
                ...problem,
                [name]: e.target.value,
              })
            } else {
              setProblem({
                ...problem,
                [name]: parseInt(e.target.value),
              })
            }
          }}
        />
      </div>
    </div>
  }

  return (
    <form onSubmit={action}>
      {formRow("name", "Name", "text", true)}
      <div className="row mb-md-3 mb-0">
        <label htmlFor="problem_type" className="col-xl-3 col-md-4 col-form-label pb-0 pb-md-2 pt-3">Problem Type</label>
        <div className="col-md-8 col-xl-6 col-form-label">
          <select
            className="form-control"
            id="problem_type"
            name="problem_type"
            required={true}
            value={problem["problem_type"] || 1}
            onChange={(e) => {
              setProblem({
                ...problem,
                "problem_type": parseInt(e.target.value),
              })
            }}
          >
            <option value="1">ICPC</option>
            <option value="2">IOI</option>
          </select>
        </div>
      </div>
      {formRow("time_limit", "Time limit", "text", true)}
      {formRow("memory_limit", "Memory limit", "text", true)}
      {formRow("tests_number", "Tests number", "number", true)}
      {formRow("wall_time_limit", "Wall time limit", "text", false)}
      {formRow("max_open_files", "Max open files", "number", false)}
      {formRow("max_threads", "Max threads", "number", false)}
      {formRow("max_output_size", "Max output size", "text", false)}
      <div className="row mt-md-4 mt-2">
        <div className="col-xl-3 d-xl-block d-none"></div>
        <div className="text-center col-12 col-xl-6">
          <button type="submit" className="btn btn-primary w-100">{buttonText}</button>
        </div>
      </div>
    </form>
  )
}