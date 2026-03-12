import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { AuthContext } from "./hooks/useAuth";
import { useAuthProvider } from "./hooks/useAuth";
import App from "./App";
import "./index.css";

function Root() {
  const auth = useAuthProvider();
  return (
    <AuthContext.Provider value={auth}>
      <App />
    </AuthContext.Provider>
  );
}

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <Root />
  </StrictMode>,
);
