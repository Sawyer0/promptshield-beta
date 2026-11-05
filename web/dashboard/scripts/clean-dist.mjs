// Empties the embedded dist directory but keeps .gitkeep, so `go build`
// still finds the embed target in a fresh checkout.
import { readdirSync, rmSync } from 'node:fs';
import { URL, fileURLToPath } from 'node:url';

const dist = fileURLToPath(new URL('../../../internal/interfaces/http/dashboard/dist', import.meta.url));
for (const entry of readdirSync(dist)) {
  if (entry !== '.gitkeep') {
    rmSync(`${dist}/${entry}`, { recursive: true, force: true });
  }
}
