import { createRoot } from "react-dom/client";
import App from "./App";
import "./index.css";

// Clerk expects VITE_CLERK_PUBLISHABLE_KEY to be set; handled via env

createRoot(document.getElementById("root")!).render(<App />);
