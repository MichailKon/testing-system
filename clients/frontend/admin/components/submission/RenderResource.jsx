import React from "react";

export default function RenderResource(name, data, error) {
  if (!data && !error) {
    return null
  }
  const wrapContent = (content) => (
    <div>
      <p className="m-0">{name}</p>
      <div>
        <pre className="bg-black bg-opacity-10 overflow-scroll d-inline-block submission_resource">
          {content}
        </pre>
      </div>
    </div>
  )

  if (error) {
    return wrapContent(<span className="text-danger">{error}</span>)
  } else {
    return wrapContent(data)
  }
}