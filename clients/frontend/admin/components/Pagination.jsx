import React from "react";

export default function Pagination(page, setPage) {
  const decPage = (e) => {
    e.preventDefault();
    setPage(page - 1);
  }

  const incPage = (e) => {
    e.preventDefault();
    setPage(page + 1);
  }

  const noPage = (e) => {
    e.preventDefault();
  }

  return (
    <nav aria-label="Pages">
      <ul className="pagination">
        {page > 1 ? (
          <li className="page-item">
            <a className="page-link" href="#" aria-label="Previous" onClick={decPage}>
              <span aria-hidden="true">&laquo;</span>
            </a>
          </li>
        ) : (
          <li className="page-item disabled">
            <a className="page-link" href="#" aria-label="Previous" onClick={noPage}>
              <span aria-hidden="true">&laquo;</span>
            </a>
          </li>
        )}
        {page > 1 ? (
          <li className="page-item">
            <a className="page-link" href="#" onClick={decPage}>{page - 1}</a>
          </li>
        ) : null}
        <li className="page-item active" aria-current="page">
          <a className="page-link" href="#" onClick={noPage}>{page}</a>
        </li>
        <li className="page-item">
          <a className="page-link" href="#" onClick={incPage}>{page + 1}</a>
        </li>
        <li className="page-item">
          <a className="page-link" href="#" aria-label="Next" onClick={incPage}>
            <span aria-hidden="true">&raquo;</span>
          </a>
        </li>
      </ul>
    </nav>
  )
}