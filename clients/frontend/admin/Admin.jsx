import React from "react";
import ReactDOM from "react-dom/client";
import { BrowserRouter, Routes, Route } from "react-router-dom";
import Home from "./pages/Home";
import Problems from "./pages/Problems";
import Problem from "./pages/Problem";
import Submissions from "./pages/Submissions";
import Submission from "./pages/Submission";
import NewProblem from "./pages/NewProblem";
import NewSubmission from "./pages/NewSubmission";

const root = ReactDOM.createRoot(document.querySelector("#application"));
root.render(
  <BrowserRouter>
    <Routes>
      <Route path="/admin" element={<Home />} />
      <Route path="/admin/problems" element={<Problems />} />
      <Route path="/admin/problem/:id" element={<Problem />} />
      <Route path="/admin/new/problem" element={<NewProblem />} />
      <Route path="/admin/submissions" element={<Submissions />} />
      <Route path="/admin/submission/:id" element={<Submission />} />
      <Route path="/admin/new/submission" element={<NewSubmission />} />
    </Routes>
  </BrowserRouter>
);