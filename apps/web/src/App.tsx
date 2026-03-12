import { BrowserRouter, Routes, Route, Navigate } from "react-router";
import ProtectedRoute from "./components/ProtectedRoute";
import Layout from "./components/Layout";
import LoginPage from "./pages/LoginPage";
import RegisterPage from "./pages/RegisterPage";
import DashboardPage from "./pages/DashboardPage";
import WorkflowCreatePage from "./pages/WorkflowCreatePage";
import WorkflowEditPage from "./pages/WorkflowEditPage";
import WorkflowDetailPage from "./pages/WorkflowDetailPage";
import RunDetailPage from "./pages/RunDetailPage";

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/register" element={<RegisterPage />} />
        <Route element={<ProtectedRoute />}>
          <Route element={<Layout />}>
            <Route path="/" element={<DashboardPage />} />
            <Route path="/workflows/new" element={<WorkflowCreatePage />} />
            <Route path="/workflows/:id" element={<WorkflowDetailPage />} />
            <Route path="/workflows/:id/edit" element={<WorkflowEditPage />} />
            <Route path="/runs/:id" element={<RunDetailPage />} />
          </Route>
        </Route>
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  );
}
