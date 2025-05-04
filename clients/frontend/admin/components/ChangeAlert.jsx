import React from "react";

export default function ChangeAlert(alert) {
  console.log(alert);
  if (!alert.hasAlert) {
    return <div></div>
  }
  if (alert.ok) {
    return <div className="alert alert-success mt-3" role="alert">
      {alert.message}
    </div>
  } else {
    return <div className="alert alert-danger mt-3" role="alert">
      {alert.message}
    </div>
  }
}



export function SendAlertRequest(axiosPromise, setAlert, onOK) {
  const clearAlert = () => {
    setAlert({
      hasAlert: false,
      ok: false,
      message: "",
    })
  }

  axiosPromise.then((resp) => {
    if (!resp.data.error) {
      if (onOK) {
        onOK(resp.data.response)
      } else {
        setAlert({
          hasAlert: true,
          ok: true,
          message: "Saved changes",
        })
      }
    } else {
      setAlert({
        hasAlert: false,
        ok: false,
        message: resp.data.message,
      })
    }
    setTimeout(clearAlert, 3000)
  }).catch((err) => {
    setAlert({
      hasAlert: true,
      ok: false,
      message: err.response.data.error,
    })
    setTimeout(clearAlert, 3000)
  })
}