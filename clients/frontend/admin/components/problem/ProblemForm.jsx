import React from "react";
import {e} from "react-router/dist/production/fog-of-war-BLArG-qZ";

export function ProblemInitialState() {
  return {
    problem_type: 1,
    test_groups: [],
  }
}

export function ProblemReducer(problem, action) {
  if (action.action === "problem") {
    if (!action.problem.test_groups) {
      action.problem.test_groups = []
    }
    return action.problem
  }
  let p = {...problem};

  const removeRequired = (idx) => {
    const oldName  = p.test_groups[idx].name

    for (let i = 0; i < p.test_groups.length; i++) {
      if (p.test_groups[i].required_group_names) {
        const idx = p.test_groups[i].required_group_names.indexOf(oldName)
        if (idx >= 0) {
          p.test_groups[i].required_group_names.splice(idx, 1)
        }
      }
    }
  }
  switch (action.action) {
    // String fields required
    case "name":
    case "time_limit":
    case "memory_limit":
      p[action.action] = action.value
      return p
    // Int fields required
    case "problem_type":
    case "tests_number":
      p[action.action] = parseInt(action.value)
      return p
    // String fields optional
    case "wall_time_limit":
    case "max_output_size":
      if (action.value === "") {
        delete p[action.action]
      } else {
        p[action.action] = action.value
      }
      return p
    // Int fields optional
    case "max_open_files":
    case "max_threads":
      if (action.value === "") {
        delete p[action.action]
      } else {
        p[action.action] = parseInt(action.value)
      }
      return p

    case "add_group":
      p.test_groups.push({})
      return p

    case "group":
      const idx = action.groupIndex
      switch (action.groupAction) {
        case "name":
          removeRequired(idx)
          p.test_groups[idx].name = action.value
          return p

        // Int fields
        case "first_test":
        case "last_test":
        case "feedback_type":
          p.test_groups[idx][action.groupAction] = parseInt(action.value)
          return p

        case "scoring_type":
          const newScoringType = parseInt(action.value)
          if (newScoringType !== 1) {
            removeRequired(idx)
          }

        // Score fields
        case "score":
        case "test_score":
          delete p.test_groups[idx]["score"]
          delete p.test_groups[idx]["test_score"]
          if (action.value !== "") {
            p[action.groupAction] = parseFloat(action.value)
          }
          return p

        case "add_required":
          if (!p.test_groups[idx].required_group_names) {
            p.test_groups[idx].required_group_names = []
          }
          p.test_groups[idx].required_group_names.push(action.value)

          p.test_groups[idx].required_group_names.sort((a, b) => {
            const aIdx = p.test_groups.findIndex((g) => g.name === a)
            const bIdx = p.test_groups.findIndex((g) => g.name === b)
            return aIdx < bIdx ? -1 : aIdx > bIdx ? 1 : 0
          })

          return p

        case "remove_required":
          const index = p.test_groups[idx].required_group_names.indexOf(action.value);
          p.test_groups[idx].required_group_names.splice(index, 1)
          return p

        case "remove":
          removeRequired(idx)
          p.test_groups.splice(idx, 1)
          return p

        default:
          throw new Error(`Invalid group action: ${action.groupAction}`)
      }

    default:
      throw new Error(`Invalid action: ${action.action}`)
  }
}

export function RenderProblemForm(
  problem,
  changeProblem,
  submitAction,
  buttonText
) {
  const action = (event) => {
    event.preventDefault();
    submitAction();
  }

  const formRowInput = (name, displayName, type, required) => {
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
          onChange={(e) => {changeProblem({
            action: name,
            value: e.target.value,
          })}}
        />
      </div>
    </div>
  }

  const formRowSelect = (name, displayName, required, values) => {
    return <div className="row mb-md-3 mb-0">
      <label htmlFor={name} className="col-xl-3 col-md-4 col-form-label">{displayName}</label>
      <div className="col-md-8 col-xl-6 col-form-label">
        <select
          className="form-control"
          id={name}
          name={name}
          required={required}
          value={problem[name] || (required ? 1 : "")}
          onChange={(e) => changeProblem({
            action: name,
            value: e.target.value,
          })}
        >
          {required ? null : (
            <option value="" key="none"></option>
          )}
          {values.map((value, index) => (
            <option key={index} value={value.value}>value.name</option>
          ))}
        </select>
      </div>
    </div>
  }

  return (
    <form onSubmit={action}>
      {formRowInput("name", "Name", "text", true)}
      {formRowSelect("problem_type", "Problem Type", true, [
        {value: 1, name: "ICPC"},
        {value: 2, name: "IOI"},
      ])}
      {formRowInput("time_limit", "Time limit", "text", true)}
      {formRowInput("memory_limit", "Memory limit", "text", true)}
      {formRowInput("tests_number", "Tests number", "number", true)}
      {formRowInput("wall_time_limit", "Wall time limit", "text", false)}
      {formRowInput("max_open_files", "Max open files", "number", false)}
      {formRowInput("max_threads", "Max threads", "number", false)}
      {formRowInput("max_output_size", "Max output size", "text", false)}

      {problem.problem_type === 2 ? renderGroups(problem, changeProblem) : null}

      <div className="row mt-md-4 mt-2">
        <div className="col-xl-3 d-xl-block d-none"></div>
        <div className="text-center col-12 col-xl-6">
          <button type="submit" className="btn btn-primary w-100">{buttonText}</button>
        </div>
      </div>
    </form>
  )
}

function renderGroups(problem, changeProblem) {
  if (problem.problem_type !== 2) {
    return null
  }
  return (
    <table>
      <thead>
      <tr>
        <th scope="row">Name</th>
        <th scope="row">First test</th>
        <th scope="row">Last test</th>
        <th scope="row">Score</th>
        <th scope="row">Scoring Type</th>
        <th scope="row">Feedback Type</th>
        <th scope="row">Required</th>
        <th scope="row">Delete</th>
      </tr>
      </thead>
      <tbody>
      {problem.test_groups.map((group, index) => renderGroup(group, index, problem, changeProblem))}
      </tbody>
    </table>
  )
}

function renderGroup(group, index, problem, changeProblem) {
  const tableInput = (name, type, required, note = null) => {
    return <td>
      <input
        className="form-control"
        id={name}
        name={name}
        type={type}
        required={required}
        value={group[name] || ""}
        onChange={(e) => {
          changeProblem({
            action: "group",
            groupIndex: index,
            groupAction: name,
            value: e.target.value,
          })
        }}
      />
      {note ? <span>{note}</span> : null}
    </td>
  }

  const tableSelect = (name, values, defaultValue = 1) => {
    return <td>
      <select
        className="form-control"
        name={`${index}-input-${name}`}
        required={true}
        value={group[name] || defaultValue}
        onChange={(e) => {
          changeProblem({
            action: "group",
            groupIndex: index,
            groupAction: name,
            value: e.target.value,
          })
        }}
      >
        {values.map((value, index) => (
          <option key={index} value={value.value}>{value.name}</option>
        ))}
      </select>
    </td>
  }

  return <tr key={index}>
    {tableInput("name", "text", true)}
    {tableInput("first_test", "number", true)}
    {tableInput("last_test", "number", true)}
    {group.scoring_type === 2 ? (
      tableInput("test_score", "number", true, "Test score")
    ) : (
      tableInput("test_score", "number", true, "Group score")
    )}
    {tableSelect("scoring_type", [
      {value: 1, name: "Complete"},
      {value: 2, name: "Each test"},
      {value: 3, name: "Min score"},
    ])}
    {tableSelect("feedback_type", [
      {value: 1, name: "None"},
      {value: 2, name: "Points"},
      {value: 3, name: "ICPC"},
      {value: 4, name: "Complete"},
      {value: 5, name: "Full"},
    ], 3)}
    {renderRequiredGroups(group, index, problem, changeProblem)}
    <td>
      <a href="#" onClick={(e) => {
        e.preventDefault();
        changeProblem({
          action: "group",
          groupIndex: index,
          groupAction: "delete",
        })
      }}>Delete</a>
    </td>
  </tr>
}

function renderRequiredGroups(group, index, problem, changeProblem) {
  const required = group.required_group_names || []
  let canAdd = []
  for (let i = 0; i < index; i++) {
    const gName = problem.test_groups[i].name
    if (problem.test_groups[i].scoring_type === 1 || !required.includes(gName)) {
      canAdd.push(gName)
    }
  }

  return <td>
    <div className="d-flex">
      {required.map((name, i) => {
        return <span
          key={`${index}-group-${i}`}
          className="badge text-bg-secondary"
        >{name} <a
            href="#"
            className="link-light link-underline-opacity-0"
            onClick={(e) => {
              e.preventDefault()
              changeProblem({
                action: "group",
                groupIndex: index,
                groupAction: "remove_required",
                value: name,
              })
            }}
          >X</a>
        </span>
      })}
    </div>
    <select
      className="form-control"
      name={`${index}-add-required`}
      required={false}
      value=""
      onChange={(e) => {
        if (e.target.value === "") {
          return
        }
        changeProblem({
          action: "group",
          groupIndex: index,
          groupAction: "add_required",
          value: e.target.value,
        })
      }}
    >
      <option value=""></option>
      {canAdd.map((name, i) => (
        <option key={`${index}-toadd-${i}`} value={name}>{name}</option>
      ))}
    </select>
  </td>
}