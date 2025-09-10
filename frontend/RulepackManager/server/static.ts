import express, { type Express } from "express";
import fs from "fs";
import path from "path";

// Use process.cwd()-relative path so it works in both ESM/CJS bundles
export function serveStatic(app: Express) {
  const distPath = process.env.DIST_DIR || path.resolve(process.cwd(), "dist", "public");

  if (!fs.existsSync(distPath)) {
    throw new Error(
      `Could not find the build directory: ${distPath}, make sure to build the client first`,
    );
  }

  app.use(express.static(distPath));

  // fall through to index.html if the file doesn't exist
  app.use("*", (_req, res) => {
    const html = fs.readFileSync(path.join(distPath, "index.html"), "utf8");
    res.setHeader('content-type','text/html');
    res.status(200).send(html);
  });
}


