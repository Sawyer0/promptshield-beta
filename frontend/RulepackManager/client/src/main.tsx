import { createRoot, hydrateRoot } from "react-dom/client";
import App from "./App";
import "./index.css";

// Mount with hydrateRoot if existing SSR markup is present; otherwise createRoot
const container = document.getElementById("root");
if (!container) throw new Error("#root not found");

if (container.hasChildNodes()) {
  hydrateRoot(container, <App />);
} else {
  createRoot(container).render(<App />);
}
